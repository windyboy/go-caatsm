package postgres

import (
	"caatsm/internal/adapter/dto"
	"caatsm/internal/adapter/mapper"
	obsmetrics "caatsm/internal/infra/metrics"
	"caatsm/internal/port"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// dbExecer is the subset of *pgxpool.Pool used by Repository. Its method set
// matches *pgxpool.Pool exactly so the concrete pool satisfies it directly, while
// tests can substitute a fake pool without a running database.
type dbExecer interface {
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
}

// uniqueViolationCode is the PostgreSQL SQLSTATE for a unique-constraint
// violation (duplicate key).
const uniqueViolationCode = "23505"

// insertTelegramSQL inserts a parsed telegram. The conflict target
// (uuid, received_at) is a safety net only: uuid is generated per parse, so it
// never prevents duplicate business messages on its own. Business-level
// idempotency is enforced by the aviation.telegram_keys gate table.
const insertTelegramSQL = `
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

// insertDedupKeySQL reserves the business-key slot in aviation.telegram_keys.
// A primary-key violation signals that the same (message_id, date_time) was
// already persisted, i.e. this is a duplicate.
const insertDedupKeySQL = `
	INSERT INTO aviation.telegram_keys (message_id, date_time) VALUES ($1, $2)
`

// Repository implements the port.Repository interface using PostgreSQL
type Repository struct {
	pool   dbExecer
	mapper *mapper.TelegramMapper
	logger *zap.Logger
}

// ProvideRepository creates a PostgreSQL repository
func ProvideRepository(pool *pgxpool.Pool, logger *zap.Logger) (port.Repository, error) {
	return &Repository{
		pool:   pool,
		mapper: mapper.NewTelegramMapper(),
		logger: logger,
	}, nil
}

// InsertOne inserts a single telegram message.
//
// Business-message identity is (message_id, date_time). When both are known we
// enforce idempotency at the database level via the aviation.telegram_keys gate
// table (see insertDedup). When either is missing (legacy/partial telegrams) we
// fall back to a direct insert without the gate, preserving the previous
// behaviour for those records.
func (r *Repository) InsertOne(ctx context.Context, msg *dto.ParsedTelegram) error {
	if msg == nil {
		return fmt.Errorf("message cannot be nil")
	}

	ctx, span := otel.Tracer("caatsm/postgres").Start(ctx, "Repository.InsertOne")
	defer span.End()

	// Set semantic database attributes
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "insert"),
		attribute.String("db.name", "aviation"),
		attribute.String("db.table", "telegrams"),
		attribute.String("caatsm.message.id", msg.MessageID),
		attribute.String("caatsm.message.category", msg.Category),
	)

	if msg.MessageID != "" && msg.DateTime != "" {
		return r.insertDedup(ctx, span, msg)
	}
	return r.insertDirect(ctx, span, msg)
}

// insertDirect inserts a telegram without the business-key gate. Used when the
// business identity is incomplete, matching the previous behaviour for partial
// records.
func (r *Repository) insertDirect(ctx context.Context, span trace.Span, msg *dto.ParsedTelegram) error {
	row, err := r.mapper.ToDBRow(msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		r.logger.Error("failed to map message to DB row", zap.Error(err))
		return fmt.Errorf("failed to map message to DB row: %w", err)
	}

	start := time.Now()
	tag, err := r.pool.Exec(ctx, insertTelegramSQL,
		row[0], row[1], row[2], row[3], row[4], row[5], row[6], row[7], row[8],
		row[9], row[10], row[11], row[12], row[13], row[14], row[15], row[16],
	)
	elapsed := time.Since(start)

	if err != nil {
		obsmetrics.RecordDBQuery("insert_one", obsmetrics.DBResultError, elapsed)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		r.logger.Error("failed to insert message",
			zap.String("uuid", msg.Uuid),
			zap.String("message_id", msg.MessageID),
			zap.Duration("elapsed", elapsed),
			zap.Error(err),
		)
		return fmt.Errorf("failed to insert message: %w", err)
	}

	obsmetrics.RecordDBQuery("insert_one", obsmetrics.DBResultOK, elapsed)

	if tag.RowsAffected() == 0 {
		r.logger.Info("Duplicate message skipped",
			zap.String("uuid", msg.Uuid),
			zap.String("message_id", msg.MessageID),
		)
	}

	r.logger.Debug("Inserted message",
		zap.String("uuid", msg.Uuid),
		zap.String("message_id", msg.MessageID),
	)

	return nil
}

// insertDedup enforces business-message idempotency atomically. It reserves the
// (message_id, date_time) slot in aviation.telegram_keys inside a transaction;
// a primary-key violation means the business message was already persisted (by
// this or a concurrent writer), so the telegram insert is skipped. Because the
// reservation and the telegram insert share the transaction, a failure rolls
// both back and cannot leave a dangling gate row.
func (r *Repository) insertDedup(ctx context.Context, span trace.Span, msg *dto.ParsedTelegram) error {
	txn, err := r.pool.Begin(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		r.logger.Error("failed to begin transaction", zap.Error(err))
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer txn.Rollback(ctx) //nolint:errcheck // Rollback after Commit is expected to fail; error handled by Commit path

	if _, err = txn.Exec(ctx, insertDedupKeySQL, msg.MessageID, msg.DateTime); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			r.logger.Info("Duplicate message detected by business key; skipping insert",
				zap.String("message_id", msg.MessageID),
				zap.String("date_time", msg.DateTime),
			)
			return port.ErrDuplicate
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		r.logger.Error("failed to reserve dedup key",
			zap.String("message_id", msg.MessageID),
			zap.String("date_time", msg.DateTime),
			zap.Error(err),
		)
		return fmt.Errorf("failed to reserve dedup key: %w", err)
	}

	row, err := r.mapper.ToDBRow(msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		r.logger.Error("failed to map message to DB row", zap.Error(err))
		return fmt.Errorf("failed to map message to DB row: %w", err)
	}

	start := time.Now()
	_, err = txn.Exec(ctx, insertTelegramSQL,
		row[0], row[1], row[2], row[3], row[4], row[5], row[6], row[7], row[8],
		row[9], row[10], row[11], row[12], row[13], row[14], row[15], row[16],
	)
	elapsed := time.Since(start)

	if err != nil {
		obsmetrics.RecordDBQuery("insert_one", obsmetrics.DBResultError, elapsed)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		r.logger.Error("failed to insert message",
			zap.String("uuid", msg.Uuid),
			zap.String("message_id", msg.MessageID),
			zap.Duration("elapsed", elapsed),
			zap.Error(err),
		)
		return fmt.Errorf("failed to insert message: %w", err)
	}

	obsmetrics.RecordDBQuery("insert_one", obsmetrics.DBResultOK, elapsed)

	if err = txn.Commit(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		r.logger.Error("failed to commit transaction", zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.logger.Debug("Inserted message",
		zap.String("uuid", msg.Uuid),
		zap.String("message_id", msg.MessageID),
	)

	return nil
}

// InsertBatch inserts multiple telegram messages in a batch using CopyFrom
func (r *Repository) InsertBatch(ctx context.Context, msgs []*dto.ParsedTelegram) error {
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
			r.logger.Error("failed to map message to DB row in batch",
				zap.Int("index", i),
				zap.Int("total", len(msgs)),
				zap.String("message_id", msg.MessageID),
				zap.Error(err),
			)
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
		r.logger.Error("failed to batch insert messages",
			zap.Int("attempted", len(msgs)),
			zap.Duration("elapsed", elapsed),
			zap.Error(err),
		)
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
func (r *Repository) InsertRaw(ctx context.Context, msg *dto.ParsedTelegram) error {
	ctx, span := otel.Tracer("caatsm/postgres").Start(ctx, "Repository.InsertRaw")
	defer span.End()
	span.SetAttributes(attribute.String("db.table", "aviation.telegrams_raw"))

	if msg == nil {
		err := fmt.Errorf("message is nil")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		r.logger.Error("message is nil in InsertRaw")
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
		r.logger.Error("failed to marshal metadata",
			zap.String("uuid", msg.Uuid),
			zap.Error(err),
		)
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
		r.logger.Error("failed to insert raw telegram",
			zap.String("uuid", msg.Uuid),
			zap.String("status", string(msg.Status)),
			zap.Duration("elapsed", elapsed),
			zap.Error(err),
		)
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
