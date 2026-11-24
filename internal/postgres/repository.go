package postgres

import (
	"caatsm/internal/telegram"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

const (
	dbResultOK    = "ok"
	dbResultError = "error"
)

// MetricsRecorder captures PostgreSQL query telemetry without importing observability directly.
type MetricsRecorder interface {
	RecordDBQuery(operation, result string, elapsed time.Duration)
}

type noopMetricsRecorder struct{}

func (noopMetricsRecorder) RecordDBQuery(string, string, time.Duration) {}

// Repository persists parsed and raw telegrams to PostgreSQL/TimescaleDB.
type Repository struct {
	pool     *pgxpool.Pool
	recorder MetricsRecorder
	logger   *zap.Logger
}

// NewRepository constructs a PostgreSQL Repository.
func NewRepository(pool *pgxpool.Pool, rec MetricsRecorder, logger *zap.Logger) *Repository {
	if rec == nil {
		rec = noopMetricsRecorder{}
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Repository{
		pool:     pool,
		recorder: rec,
		logger:   logger,
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
		exists, err := r.messageExists(ctx, msg)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			r.logger.Error("failed to check existing message",
				zap.String("message_id", msg.MessageID),
				zap.String("date_time", msg.DateTime),
				zap.String("category", msg.Category),
				zap.Error(err),
			)
			return fmt.Errorf("failed to check existing message: %w", err)
		}
		if exists {
			r.logger.Info("Duplicate message detected; skipping insert",
				zap.String("message_id", msg.MessageID),
				zap.String("date_time", msg.DateTime),
				zap.String("category", msg.Category),
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
		r.recorder.RecordDBQuery("insert_one", dbResultError, elapsed)
		r.logger.Error("failed to insert message",
			zap.String("uuid", msg.UUID),
			zap.String("message_id", msg.MessageID),
			zap.Duration("elapsed", elapsed),
			zap.Error(err),
		)
		return fmt.Errorf("failed to insert message: %w", err)
	}

	r.recorder.RecordDBQuery("insert_one", dbResultOK, elapsed)

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
		r.recorder.RecordDBQuery("insert_batch", dbResultError, elapsed)
		r.logger.Error("failed to batch insert messages",
			zap.Int("attempted", len(msgs)),
			zap.Duration("elapsed", elapsed),
			zap.Error(err),
		)
		return fmt.Errorf("failed to batch insert messages: %w", err)
	}

	r.recorder.RecordDBQuery("insert_batch", dbResultOK, elapsed)
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

	var rawUUID uuid.UUID
	if msg.UUID == "" {
		rawUUID = uuid.New()
		msg.UUID = rawUUID.String()
	} else {
		var err error
		rawUUID, err = uuid.Parse(msg.UUID)
		if err != nil {
			return fmt.Errorf("invalid UUID for raw telegram: %w", err)
		}
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
		rawUUID,
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
		r.recorder.RecordDBQuery("insert_raw", dbResultError, elapsed)
		r.logger.Error("failed to insert raw telegram",
			zap.String("uuid", msg.UUID),
			zap.String("status", string(msg.Status)),
			zap.Duration("elapsed", elapsed),
			zap.Error(err),
		)
		return fmt.Errorf("failed to insert raw telegram: %w", err)
	}

	r.recorder.RecordDBQuery("insert_raw", dbResultOK, elapsed)
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

// messageExists performs a time-bounded idempotency check (7-day TimescaleDB chunk window).
// For weather reports, it checks both the current structured message_id and the legacy
// {station}-{issue_time} format, confirming normalized content equality so distinct reports
// (e.g., original vs COR/AMD or two different COR updates at the same issue time) are preserved.
func (r *Repository) messageExists(ctx context.Context, msg *telegram.Message) (bool, error) {
	if msg == nil || msg.MessageID == "" || msg.DateTime == "" {
		return false, nil
	}

	if isWeatherMessage(msg) {
		return r.weatherMessageExists(ctx, msg)
	}

	const queryWithCategory = `
		SELECT 1
		FROM aviation.telegrams
		WHERE message_id = $1
		  AND date_time = $2
		  AND category = $3
		  AND received_at >= NOW() - INTERVAL '7 days'
		LIMIT 1
	`
	const queryWithoutCategory = `
		SELECT 1
		FROM aviation.telegrams
		WHERE message_id = $1
		  AND date_time = $2
		  AND received_at >= NOW() - INTERVAL '7 days'
		LIMIT 1
	`

	var one int
	start := time.Now()
	var err error
	if msg.Category != "" {
		err = r.pool.QueryRow(ctx, queryWithCategory, msg.MessageID, msg.DateTime, msg.Category).Scan(&one)
	} else {
		err = r.pool.QueryRow(ctx, queryWithoutCategory, msg.MessageID, msg.DateTime).Scan(&one)
	}
	elapsed := time.Since(start)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.recorder.RecordDBQuery("message_exists", dbResultOK, elapsed)
			return false, nil
		}
		r.recorder.RecordDBQuery("message_exists", dbResultError, elapsed)
		r.logger.Error("failed to check message existence",
			zap.String("message_id", msg.MessageID),
			zap.String("date_time", msg.DateTime),
			zap.String("category", msg.Category),
			zap.Error(err),
		)
		return false, err
	}
	r.recorder.RecordDBQuery("message_exists", dbResultOK, elapsed)
	return true, nil
}

func (r *Repository) weatherMessageExists(ctx context.Context, msg *telegram.Message) (bool, error) {
	keys := weatherCandidateMessageIDs(msg)
	const query = `
		SELECT content
		FROM aviation.telegrams
		WHERE message_id = ANY($1)
		  AND date_time = $2
		  AND category = $3
		  AND received_at >= NOW() - INTERVAL '7 days'
	`

	start := time.Now()
	rows, err := r.pool.Query(ctx, query, keys, msg.DateTime, msg.Category)
	elapsed := time.Since(start)
	if err != nil {
		r.recorder.RecordDBQuery("message_exists", dbResultError, elapsed)
		r.logger.Error("failed to query weather message candidates",
			zap.Strings("message_ids", keys),
			zap.String("date_time", msg.DateTime),
			zap.String("category", msg.Category),
			zap.Error(err),
		)
		return false, err
	}
	defer rows.Close()

	var candidates []string
	for rows.Next() {
		var content string
		if scanErr := rows.Scan(&content); scanErr != nil {
			r.recorder.RecordDBQuery("message_exists", dbResultError, time.Since(start))
			return false, scanErr
		}
		candidates = append(candidates, content)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		r.recorder.RecordDBQuery("message_exists", dbResultError, time.Since(start))
		return false, rowsErr
	}
	r.recorder.RecordDBQuery("message_exists", dbResultOK, time.Since(start))
	return matchesWeatherContent(candidates, msg.Content), nil
}

func isWeatherMessage(msg *telegram.Message) bool {
	if msg == nil {
		return false
	}
	if msg.Protocol == telegram.ProtocolWeather {
		return true
	}
	switch strings.ToUpper(strings.TrimSpace(msg.Category)) {
	case "METAR", "SPECI", "TAF":
		return true
	default:
		return false
	}
}

// weatherCandidateMessageIDs returns the current weather message_id and, when applicable,
// the legacy {station}-{issue_time} key used prior to type/modifier-prefixed keys.
func weatherCandidateMessageIDs(msg *telegram.Message) []string {
	if msg == nil || msg.MessageID == "" {
		return nil
	}
	keys := []string{msg.MessageID}
	if legacy := legacyWeatherMessageID(msg.MessageID); legacy != "" && legacy != msg.MessageID {
		keys = append(keys, legacy)
	}
	return keys
}

func legacyWeatherMessageID(messageID string) string {
	parts := strings.Split(messageID, "-")
	// Current format: {TYPE}-{STATION}-{YYYYMMDDHHMMSS} (3 parts)
	// or {TYPE}-{MODIFIER}-{STATION}-{YYYYMMDDHHMMSS} (4 parts).
	if len(parts) < 3 {
		return ""
	}
	switch strings.ToUpper(parts[0]) {
	case "METAR", "SPECI", "TAF":
		return parts[len(parts)-2] + "-" + parts[len(parts)-1]
	default:
		return ""
	}
}

func matchesWeatherContent(existingContents []string, incomingContent string) bool {
	want := normalizeWeatherContent(incomingContent)
	for _, existing := range existingContents {
		if normalizeWeatherContent(existing) == want {
			return true
		}
	}
	return false
}

// normalizeWeatherContent normalizes CRLF/CR line endings and leading/trailing line whitespace
// without altering internal report tokens.
func normalizeWeatherContent(raw string) string {
	s := strings.ReplaceAll(raw, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return strings.Join(out, "\n")
}
