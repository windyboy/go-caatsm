package postgres

import (
	"caatsm/internal/observability"
	"caatsm/internal/telegram"
	"context"
	"encoding/json"
	"errors"
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

// Repository persists parsed and raw telegrams to PostgreSQL/TimescaleDB.
type Repository struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewRepository constructs a PostgreSQL Repository.
func NewRepository(pool *pgxpool.Pool, logger *zap.Logger) *Repository {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Repository{
		pool:   pool,
		logger: logger,
	}
}

// InsertOne inserts a single parsed telegram into aviation.telegrams.
func (r *Repository) InsertOne(ctx context.Context, msg *telegram.Message) error {
	if msg == nil {
		return fmt.Errorf("message cannot be nil")
	}
	if r.pool == nil {
		return fmt.Errorf("postgres pool is nil")
	}

	ctx, span := otel.Tracer("caatsm/postgres").Start(ctx, "Repository.InsertOne")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "insert"),
		attribute.String("db.name", "aviation"),
		attribute.String("db.table", "telegrams"),
		attribute.String("caatsm.message.id", msg.MessageID),
		attribute.String("caatsm.message.category", msg.Category),
	)

	if msg.MessageID != "" && msg.DateTime != "" {
		exists, err := r.messageExists(ctx, msg.MessageID, msg.DateTime)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			r.logger.Error("failed to check existing message",
				zap.String("message_id", msg.MessageID),
				zap.String("date_time", msg.DateTime),
				zap.Error(err),
			)
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

	row, err := toDBRow(msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		r.logger.Error("failed to map message to DB row", zap.Error(err))
		return fmt.Errorf("failed to map message to DB row: %w", err)
	}

	const query = `
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
	tag, err := r.pool.Exec(ctx, query, row...)
	elapsed := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		observability.RecordDBQuery("insert_one", observability.DBResultError, elapsed)
		r.logger.Error("failed to insert message",
			zap.String("uuid", msg.UUID),
			zap.String("message_id", msg.MessageID),
			zap.Duration("elapsed", elapsed),
			zap.Error(err),
		)
		return fmt.Errorf("failed to insert message: %w", err)
	}

	observability.RecordDBQuery("insert_one", observability.DBResultOK, elapsed)

	if tag.RowsAffected() == 0 {
		r.logger.Info("Duplicate message skipped",
			zap.String("uuid", msg.UUID),
			zap.String("message_id", msg.MessageID),
		)
		return nil
	}

	r.logger.Debug("Inserted message",
		zap.String("uuid", msg.UUID),
		zap.String("message_id", msg.MessageID),
	)
	return nil
}

// InsertBatch inserts multiple telegram messages using pgx CopyFrom.
func (r *Repository) InsertBatch(ctx context.Context, msgs []*telegram.Message) error {
	if len(msgs) == 0 {
		return nil
	}
	if r.pool == nil {
		return fmt.Errorf("postgres pool is nil")
	}

	ctx, span := otel.Tracer("caatsm/postgres").Start(ctx, "Repository.InsertBatch")
	defer span.End()
	span.SetAttributes(attribute.String("db.table", "aviation.telegrams"))

	rows := make([][]any, len(msgs))
	for i, msg := range msgs {
		row, err := toDBRow(msg)
		if err != nil {
			r.logger.Error("failed to map message to DB row in batch",
				zap.Int("index", i),
				zap.Int("total", len(msgs)),
				zap.Error(err),
			)
			return fmt.Errorf("failed to map message %d to DB row: %w", i, err)
		}
		rows[i] = row
	}

	start := time.Now()
	copyCount, err := r.pool.CopyFrom(
		ctx,
		pgx.Identifier{"aviation", "telegrams"},
		telegramColumns,
		pgx.CopyFromRows(rows),
	)
	elapsed := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		observability.RecordDBQuery("insert_batch", observability.DBResultError, elapsed)
		r.logger.Error("failed to batch insert messages",
			zap.Int("attempted", len(msgs)),
			zap.Duration("elapsed", elapsed),
			zap.Error(err),
		)
		return fmt.Errorf("failed to batch insert messages: %w", err)
	}

	observability.RecordDBQuery("insert_batch", observability.DBResultOK, elapsed)
	span.SetAttributes(attribute.Int64("db.inserted", copyCount))
	r.logger.Info("Batch inserted messages",
		zap.Int("count", int(copyCount)),
		zap.Int("attempted", len(msgs)),
	)
	return nil
}

// InsertRaw inserts a failed or unparsed telegram into aviation.telegrams_raw for audit.
func (r *Repository) InsertRaw(ctx context.Context, msg *telegram.Message) error {
	ctx, span := otel.Tracer("caatsm/postgres").Start(ctx, "Repository.InsertRaw")
	defer span.End()
	span.SetAttributes(attribute.String("db.table", "aviation.telegrams_raw"))

	if msg == nil {
		err := fmt.Errorf("message is nil")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		r.logger.Error("message is nil in InsertRaw")
		return err
	}
	if r.pool == nil {
		return fmt.Errorf("postgres pool is nil")
	}

	if msg.UUID == "" {
		msg.UUID = uuid.NewString()
	}
	if msg.ReceivedAt.IsZero() {
		msg.ReceivedAt = time.Now()
	}

	metadata := map[string]any{
		"message_id": msg.MessageID,
		"category":   msg.Category,
		"comments":   msg.Comments,
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		r.logger.Error("failed to marshal metadata", zap.String("uuid", msg.UUID), zap.Error(err))
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	const query = `
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
		msg.UUID,
		string(msg.Status),
		msg.ErrorReason,
		msg.Content,
		msg.ReceivedAt,
		metadataJSON,
	)
	elapsed := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		observability.RecordDBQuery("insert_raw", observability.DBResultError, elapsed)
		r.logger.Error("failed to insert raw telegram",
			zap.String("uuid", msg.UUID),
			zap.String("status", string(msg.Status)),
			zap.Duration("elapsed", elapsed),
			zap.Error(err),
		)
		return fmt.Errorf("failed to insert raw telegram: %w", err)
	}

	observability.RecordDBQuery("insert_raw", observability.DBResultOK, elapsed)
	span.SetAttributes(
		attribute.String("telegram.uuid", msg.UUID),
		attribute.String("telegram.status", string(msg.Status)),
	)
	r.logger.Debug("Persisted raw telegram",
		zap.String("uuid", msg.UUID),
		zap.String("status", string(msg.Status)),
	)
	return nil
}

// messageExists performs a time-bounded idempotency check (7-day TimescaleDB chunk window)
// using (message_id, date_time).
func (r *Repository) messageExists(ctx context.Context, messageID, dateTime string) (bool, error) {
	if messageID == "" || dateTime == "" {
		return false, nil
	}

	const query = `
		SELECT 1
		FROM aviation.telegrams
		WHERE message_id = $1
		  AND date_time = $2
		  AND received_at >= NOW() - INTERVAL '7 days'
		LIMIT 1
	`

	var one int
	if err := r.pool.QueryRow(ctx, query, messageID, dateTime).Scan(&one); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		r.logger.Error("failed to check message existence",
			zap.String("message_id", messageID),
			zap.String("date_time", dateTime),
			zap.Error(err),
		)
		return false, err
	}
	return true, nil
}
