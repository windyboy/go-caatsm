package postgres

import (
	"caatsm/internal/adapter"
	"caatsm/internal/adapter/mapper"
	"caatsm/internal/domain"
	obsmetrics "caatsm/internal/observability/metrics"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

// Repository implements the adapter.Repository interface using PostgreSQL
type Repository struct {
	pool   *pgxpool.Pool
	mapper *mapper.TelegramMapper
	logger *zap.Logger
}

// ProvideRepository creates a PostgreSQL repository
func ProvideRepository(pool *pgxpool.Pool, logger *zap.Logger) (adapter.Repository, error) {
	return &Repository{
		pool:   pool,
		mapper: mapper.NewTelegramMapper(),
		logger: logger,
	}, nil
}

// InsertOne inserts a single telegram message
func (r *Repository) InsertOne(ctx context.Context, msg *domain.ParsedMessage) error {
	ctx, span := otel.Tracer("caatsm/postgres").Start(ctx, "Repository.InsertOne")
	defer span.End()
	span.SetAttributes(attribute.String("db.table", "aviation.telegrams"))

	// Optional idempotency check based on business message identity. If we have a
	// non-empty message ID and date/time, we can cheaply skip duplicates here to
	// avoid applying the same business event multiple times.
	if msg != nil && msg.MessageID != "" && msg.DateTime != "" {
		exists, err := r.messageExists(ctx, msg.MessageID, msg.DateTime)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return fmt.Errorf("failed to check existing message: %w", err)
		}
		if exists {
			r.logger.Info("Duplicate message detected by message_id/date_time; skipping insert",
				zap.String("message_id", msg.MessageID),
				zap.String("date_time", msg.DateTime),
			)
			return nil
		}
	}

	row, err := r.mapper.ToDBRow(msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("failed to map message to DB row: %w", err)
	}

	query := `
		INSERT INTO aviation.telegrams (
			uuid, message_id, date_time, priority_indicator, primary_address,
			secondary_addresses, originator, originator_date_time, category,
			content, body_data, status, error_reason,
			received_at, parsed_at, dispatched_at, need_dispatch
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)
		ON CONFLICT (uuid, received_at) DO NOTHING
	`

	start := time.Now()
	tag, err := r.pool.Exec(ctx, query,
		row[0], row[1], row[2], row[3], row[4], row[5], row[6], row[7], row[8],
		row[9], row[10], row[11], row[12], row[13], row[14], row[15], row[16],
	)
	elapsed := time.Since(start)

	result := obsmetrics.DBResultOK
	if err != nil {
		result = obsmetrics.DBResultError
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		obsmetrics.RecordDBQuery("insert_one", result, elapsed)
		return fmt.Errorf("failed to insert message: %w", err)
	}

	obsmetrics.RecordDBQuery("insert_one", result, elapsed)

	if tag.RowsAffected() == 0 {
		r.logger.Info("Duplicate message skipped",
			zap.String("uuid", msg.Uuid),
			zap.String("message_id", msg.MessageID),
		)
		return nil
	}

	r.logger.Debug("Inserted message",
		zap.String("uuid", msg.Uuid),
		zap.String("message_id", msg.MessageID),
	)

	return nil
}

// InsertBatch inserts multiple telegram messages in a batch using CopyFrom
func (r *Repository) InsertBatch(ctx context.Context, msgs []*domain.ParsedMessage) error {
	ctx, span := otel.Tracer("caatsm/postgres").Start(ctx, "Repository.InsertBatch")
	defer span.End()
	span.SetAttributes(attribute.String("db.table", "aviation.telegrams"))

	if len(msgs) == 0 {
		return nil
	}

	// Convert messages to rows
	rows := make([][]interface{}, len(msgs))
	for i, msg := range msgs {
		row, err := r.mapper.ToDBRow(msg)
		if err != nil {
			return fmt.Errorf("failed to map message %d to DB row: %w", i, err)
		}
		rows[i] = row
	}

	// Use CopyFrom for efficient batch insert
	start := time.Now()
	copyCount, err := r.pool.CopyFrom(
		ctx,
		pgx.Identifier{"aviation", "telegrams"},
		[]string{
			"uuid", "message_id", "date_time", "priority_indicator", "primary_address",
			"secondary_addresses", "originator", "originator_date_time", "category",
			"content", "body_data", "status", "error_reason",
			"received_at", "parsed_at", "dispatched_at", "need_dispatch",
		},
		pgx.CopyFromRows(rows),
	)
	elapsed := time.Since(start)

	result := obsmetrics.DBResultOK
	if err != nil {
		result = obsmetrics.DBResultError
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		obsmetrics.RecordDBQuery("insert_batch", result, elapsed)
		return fmt.Errorf("failed to batch insert messages: %w", err)
	}

	obsmetrics.RecordDBQuery("insert_batch", result, elapsed)

	span.SetAttributes(attribute.Int64("db.inserted", copyCount))
	r.logger.Info("Batch inserted messages",
		zap.Int("count", int(copyCount)),
		zap.Int("attempted", len(msgs)),
	)

	return nil
}

// InsertRaw inserts a failed telegram into aviation.telegrams_raw for post-processing.
func (r *Repository) InsertRaw(ctx context.Context, msg *domain.ParsedMessage) error {
	ctx, span := otel.Tracer("caatsm/postgres").Start(ctx, "Repository.InsertRaw")
	defer span.End()
	span.SetAttributes(attribute.String("db.table", "aviation.telegrams_raw"))

	if msg == nil {
		err := fmt.Errorf("message is nil")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("message is nil")
	}
	if msg.Uuid == "" {
		msg.Uuid = uuid.NewString()
	}
	if msg.ReceivedAt.IsZero() {
		msg.ReceivedAt = time.Now()
	}

	metadata := map[string]interface{}{
		"message_id": msg.MessageID,
		"category":   msg.Category,
		"comments":   msg.Comments,
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO aviation.telegrams_raw (
			uuid, status, error_reason, content, received_at, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)
		ON CONFLICT (uuid, received_at) DO UPDATE
		SET status = EXCLUDED.status,
		    error_reason = EXCLUDED.error_reason,
			content = EXCLUDED.content,
			metadata = EXCLUDED.metadata
	`

	start := time.Now()
	_, err = r.pool.Exec(ctx, query,
		msg.Uuid,
		string(msg.Status),
		msg.ErrorReason,
		msg.Content,
		msg.ReceivedAt,
		metadataJSON,
	)
	elapsed := time.Since(start)

	result := obsmetrics.DBResultOK
	if err != nil {
		result = obsmetrics.DBResultError
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		obsmetrics.RecordDBQuery("insert_raw", result, elapsed)
		return fmt.Errorf("failed to insert raw telegram: %w", err)
	}

	obsmetrics.RecordDBQuery("insert_raw", result, elapsed)

	span.SetAttributes(attribute.String("telegram.uuid", msg.Uuid), attribute.String("telegram.status", string(msg.Status)))
	r.logger.Debug("Persisted raw telegram",
		zap.String("uuid", msg.Uuid),
		zap.String("status", string(msg.Status)),
	)

	return nil
}

// messageExists performs a lightweight existence check for a business message,
// using (message_id, date_time) as the idempotency key. This avoids requiring a
// unique constraint at the database level while still preventing duplicate effects.
func (r *Repository) messageExists(ctx context.Context, messageID, dateTime string) (bool, error) {
	if messageID == "" || dateTime == "" {
		return false, nil
	}

	const query = `
		SELECT 1
		FROM aviation.telegrams
		WHERE message_id = $1 AND date_time = $2
		LIMIT 1
	`

	var one int
	if err := r.pool.QueryRow(ctx, query, messageID, dateTime).Scan(&one); err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
