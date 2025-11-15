package postgres

import (
	"caatsm/internal/adapter"
	"caatsm/internal/adapter/mapper"
	"caatsm/internal/domain"
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

	tag, err := r.pool.Exec(ctx, query,
		row[0], row[1], row[2], row[3], row[4], row[5], row[6], row[7], row[8],
		row[9], row[10], row[11], row[12], row[13], row[14], row[15], row[16],
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("failed to insert message: %w", err)
	}

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
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("failed to batch insert messages: %w", err)
	}

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

	_, err = r.pool.Exec(ctx, query,
		msg.Uuid,
		string(msg.Status),
		msg.ErrorReason,
		msg.Content,
		msg.ReceivedAt,
		metadataJSON,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("failed to insert raw telegram: %w", err)
	}

	span.SetAttributes(attribute.String("telegram.uuid", msg.Uuid), attribute.String("telegram.status", string(msg.Status)))
	r.logger.Debug("Persisted raw telegram",
		zap.String("uuid", msg.Uuid),
		zap.String("status", string(msg.Status)),
	)

	return nil
}
