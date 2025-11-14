package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"caatsm/internal/domain"
	"caatsm/internal/iface"
	applog "caatsm/internal/infra/log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TelegramRepository struct {
	pool *pgxpool.Pool
}

func NewTelegramRepository(pool *pgxpool.Pool) *TelegramRepository {
	return &TelegramRepository{
		pool: pool,
	}
}

// Create inserts a new telegram message
func (r *TelegramRepository) Create(ctx context.Context, msg *domain.ParsedMessage) error {
	bodyDataJSON, err := json.Marshal(msg.BodyData)
	if err != nil {
		return fmt.Errorf("failed to marshal body data: %w", err)
	}

	secondaryAddressesJSON, err := json.Marshal(msg.SecondaryAddresses)
	if err != nil {
		return fmt.Errorf("failed to marshal secondary addresses: %w", err)
	}

	msgUUID, err := uuid.Parse(msg.Uuid)
	if err != nil {
		msgUUID = uuid.New()
	}

	query := `
		INSERT INTO aviation.telegrams (
			uuid, message_id, date_time, priority_indicator,
			primary_address, secondary_addresses, originator,
			originator_date_time, category, content, body_data,
			received_at, parsed_at, dispatched_at, need_dispatch
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	_, err = r.pool.Exec(ctx, query,
		msgUUID,
		msg.MessageID,
		msg.DateTime,
		msg.PriorityIndicator,
		msg.PrimaryAddress,
		string(secondaryAddressesJSON),
		msg.Originator,
		msg.OriginatorDateTime,
		msg.Category,
		msg.Content,
		bodyDataJSON,
		msg.ReceivedAt,
		msg.ParsedAt,
		msg.DispatchedAt,
		msg.NeedDispatch,
	)

	if err != nil {
		return fmt.Errorf("failed to insert telegram: %w", err)
	}

	return nil
}

// CreateBatch inserts multiple telegram messages in a single transaction
func (r *TelegramRepository) CreateBatch(ctx context.Context, messages []*domain.ParsedMessage) error {
	if len(messages) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO aviation.telegrams (
			uuid, message_id, date_time, priority_indicator,
			primary_address, secondary_addresses, originator,
			originator_date_time, category, content, body_data,
			received_at, parsed_at, dispatched_at, need_dispatch
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	batch := &pgx.Batch{}
	for _, msg := range messages {
		bodyDataJSON, _ := json.Marshal(msg.BodyData)
		secondaryAddressesJSON, _ := json.Marshal(msg.SecondaryAddresses)

		msgUUID, err := uuid.Parse(msg.Uuid)
		if err != nil {
			msgUUID = uuid.New()
		}

		batch.Queue(query,
			msgUUID,
			msg.MessageID,
			msg.DateTime,
			msg.PriorityIndicator,
			msg.PrimaryAddress,
			string(secondaryAddressesJSON),
			msg.Originator,
			msg.OriginatorDateTime,
			msg.Category,
			msg.Content,
			bodyDataJSON,
			msg.ReceivedAt,
			msg.ParsedAt,
			msg.DispatchedAt,
			msg.NeedDispatch,
		)
	}

	results := tx.SendBatch(ctx, batch)
	defer results.Close()

	for i := 0; i < len(messages); i++ {
		_, err := results.Exec()
		if err != nil {
			return fmt.Errorf("failed to execute batch insert at index %d: %w", i, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetByID retrieves a telegram message by UUID
func (r *TelegramRepository) GetByID(ctx context.Context, id string) (*domain.ParsedMessage, error) {
	msgUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid UUID format: %w", err)
	}

	query := `
		SELECT uuid, message_id, date_time, priority_indicator,
		       primary_address, secondary_addresses, originator,
		       originator_date_time, category, content, body_data,
		       received_at, parsed_at, dispatched_at, need_dispatch
		FROM aviation.telegrams
		WHERE uuid = $1
	`

	var msg domain.ParsedMessage
	var bodyDataJSON []byte
	var secondaryAddressesJSON string
	var parsedAt, dispatchedAt *time.Time

	err = r.pool.QueryRow(ctx, query, msgUUID).Scan(
		&msg.Uuid,
		&msg.MessageID,
		&msg.DateTime,
		&msg.PriorityIndicator,
		&msg.PrimaryAddress,
		&secondaryAddressesJSON,
		&msg.Originator,
		&msg.OriginatorDateTime,
		&msg.Category,
		&msg.Content,
		&bodyDataJSON,
		&msg.ReceivedAt,
		&parsedAt,
		&dispatchedAt,
		&msg.NeedDispatch,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("telegram not found: %s", id)
		}
		return nil, fmt.Errorf("failed to query telegram: %w", err)
	}

	if parsedAt != nil {
		msg.ParsedAt = *parsedAt
	}
	if dispatchedAt != nil {
		msg.DispatchedAt = *dispatchedAt
	}

	if len(bodyDataJSON) > 0 {
		if err := json.Unmarshal(bodyDataJSON, &msg.BodyData); err != nil {
			logger := applog.Sugared()
			logger.Warnf("Failed to unmarshal body data for UUID %s: %v", id, err)
		}
	}

	if secondaryAddressesJSON != "" {
		msg.SecondaryAddresses = secondaryAddressesJSON
	}

	msg.Parsed = msg.Category != ""

	return &msg, nil
}

// List retrieves telegram messages with filters and pagination
func (r *TelegramRepository) List(ctx context.Context, filters *iface.TelegramFilters) ([]*domain.ParsedMessage, int64, error) {
	whereClauses := []string{}
	args := []interface{}{}
	argIndex := 1

	if filters.Category != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("category = $%d", argIndex))
		args = append(args, filters.Category)
		argIndex++
	}

	if filters.MessageID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("message_id = $%d", argIndex))
		args = append(args, filters.MessageID)
		argIndex++
	}

	if filters.PrimaryAddress != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("primary_address = $%d", argIndex))
		args = append(args, filters.PrimaryAddress)
		argIndex++
	}

	if filters.PriorityIndicator != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("priority_indicator = $%d", argIndex))
		args = append(args, filters.PriorityIndicator)
		argIndex++
	}

	if filters.StartTime != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("received_at >= $%d", argIndex))
		args = append(args, *filters.StartTime)
		argIndex++
	}

	if filters.EndTime != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("received_at <= $%d", argIndex))
		args = append(args, *filters.EndTime)
		argIndex++
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM aviation.telegrams %s", whereClause)
	var total int64
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count telegrams: %w", err)
	}

	// List query
	orderBy := "received_at"
	if filters.OrderBy != "" {
		orderBy = filters.OrderBy
	}

	orderDirection := "DESC"
	if filters.OrderDirection == "ASC" {
		orderDirection = "ASC"
	}

	limit := 100
	if filters.Limit > 0 {
		limit = filters.Limit
	}

	offset := 0
	if filters.Offset > 0 {
		offset = filters.Offset
	}

	query := fmt.Sprintf(`
		SELECT uuid, message_id, date_time, priority_indicator,
		       primary_address, secondary_addresses, originator,
		       originator_date_time, category, content, body_data,
		       received_at, parsed_at, dispatched_at, need_dispatch
		FROM aviation.telegrams
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, orderDirection, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query telegrams: %w", err)
	}
	defer rows.Close()

	messages := []*domain.ParsedMessage{}
	for rows.Next() {
		var msg domain.ParsedMessage
		var bodyDataJSON []byte
		var secondaryAddressesJSON string
		var parsedAt, dispatchedAt *time.Time

		err := rows.Scan(
			&msg.Uuid,
			&msg.MessageID,
			&msg.DateTime,
			&msg.PriorityIndicator,
			&msg.PrimaryAddress,
			&secondaryAddressesJSON,
			&msg.Originator,
			&msg.OriginatorDateTime,
			&msg.Category,
			&msg.Content,
			&bodyDataJSON,
			&msg.ReceivedAt,
			&parsedAt,
			&dispatchedAt,
			&msg.NeedDispatch,
		)

		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan telegram: %w", err)
		}

		if parsedAt != nil {
			msg.ParsedAt = *parsedAt
		}
		if dispatchedAt != nil {
			msg.DispatchedAt = *dispatchedAt
		}

		if len(bodyDataJSON) > 0 {
			json.Unmarshal(bodyDataJSON, &msg.BodyData)
		}

		if secondaryAddressesJSON != "" {
			msg.SecondaryAddresses = secondaryAddressesJSON
		}

		msg.Parsed = msg.Category != ""

		messages = append(messages, &msg)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating rows: %w", err)
	}

	return messages, total, nil
}

// GetByTimeRange retrieves messages within a time range
func (r *TelegramRepository) GetByTimeRange(ctx context.Context, start, end time.Time) ([]*domain.ParsedMessage, error) {
	query := `
		SELECT uuid, message_id, date_time, priority_indicator,
		       primary_address, secondary_addresses, originator,
		       originator_date_time, category, content, body_data,
		       received_at, parsed_at, dispatched_at, need_dispatch
		FROM aviation.telegrams
		WHERE received_at >= $1 AND received_at <= $2
		ORDER BY received_at DESC
	`

	rows, err := r.pool.Query(ctx, query, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query telegrams by time range: %w", err)
	}
	defer rows.Close()

	messages := []*domain.ParsedMessage{}
	for rows.Next() {
		var msg domain.ParsedMessage
		var bodyDataJSON []byte
		var secondaryAddressesJSON string
		var parsedAt, dispatchedAt *time.Time

		err := rows.Scan(
			&msg.Uuid,
			&msg.MessageID,
			&msg.DateTime,
			&msg.PriorityIndicator,
			&msg.PrimaryAddress,
			&secondaryAddressesJSON,
			&msg.Originator,
			&msg.OriginatorDateTime,
			&msg.Category,
			&msg.Content,
			&bodyDataJSON,
			&msg.ReceivedAt,
			&parsedAt,
			&dispatchedAt,
			&msg.NeedDispatch,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan telegram: %w", err)
		}

		if parsedAt != nil {
			msg.ParsedAt = *parsedAt
		}
		if dispatchedAt != nil {
			msg.DispatchedAt = *dispatchedAt
		}

		if len(bodyDataJSON) > 0 {
			json.Unmarshal(bodyDataJSON, &msg.BodyData)
		}

		if secondaryAddressesJSON != "" {
			msg.SecondaryAddresses = secondaryAddressesJSON
		}

		msg.Parsed = msg.Category != ""

		messages = append(messages, &msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return messages, nil
}

// Count returns the total count of messages matching filters
func (r *TelegramRepository) Count(ctx context.Context, filters *iface.TelegramFilters) (int64, error) {
	whereClauses := []string{}
	args := []interface{}{}
	argIndex := 1

	if filters.Category != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("category = $%d", argIndex))
		args = append(args, filters.Category)
		argIndex++
	}

	if filters.MessageID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("message_id = $%d", argIndex))
		args = append(args, filters.MessageID)
		argIndex++
	}

	if filters.PrimaryAddress != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("primary_address = $%d", argIndex))
		args = append(args, filters.PrimaryAddress)
		argIndex++
	}

	if filters.PriorityIndicator != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("priority_indicator = $%d", argIndex))
		args = append(args, filters.PriorityIndicator)
		argIndex++
	}

	if filters.StartTime != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("received_at >= $%d", argIndex))
		args = append(args, *filters.StartTime)
		argIndex++
	}

	if filters.EndTime != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("received_at <= $%d", argIndex))
		args = append(args, *filters.EndTime)
		argIndex++
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	query := fmt.Sprintf("SELECT COUNT(*) FROM aviation.telegrams %s", whereClause)
	var count int64
	err := r.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count telegrams: %w", err)
	}

	return count, nil
}

