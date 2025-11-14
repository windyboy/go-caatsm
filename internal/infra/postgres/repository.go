package postgres

import (
	"caatsm/internal/adapter"
	"caatsm/internal/adapter/mapper"
	"caatsm/internal/domain"
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
	row, err := r.mapper.ToDBRow(msg)
	if err != nil {
		return fmt.Errorf("failed to map message to DB row: %w", err)
	}

	query := `
		INSERT INTO aviation.telegrams (
			uuid, message_id, date_time, priority_indicator, primary_address,
			secondary_addresses, originator, originator_date_time, category,
			content, body_data, received_at, parsed_at, dispatched_at, need_dispatch
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)
		ON CONFLICT (uuid) DO NOTHING
	`

	_, err = r.pool.Exec(ctx, query,
		row[0], row[1], row[2], row[3], row[4], row[5], row[6], row[7], row[8],
		row[9], row[10], row[11], row[12], row[13], row[14],
	)
	if err != nil {
		return fmt.Errorf("failed to insert message: %w", err)
	}

	r.logger.Debug("Inserted message",
		zap.String("uuid", msg.Uuid),
		zap.String("message_id", msg.MessageID),
	)

	return nil
}

// InsertBatch inserts multiple telegram messages in a batch using CopyFrom
func (r *Repository) InsertBatch(ctx context.Context, msgs []*domain.ParsedMessage) error {
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
			"content", "body_data", "received_at", "parsed_at", "dispatched_at", "need_dispatch",
		},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("failed to batch insert messages: %w", err)
	}

	r.logger.Info("Batch inserted messages",
		zap.Int("count", int(copyCount)),
		zap.Int("attempted", len(msgs)),
	)

	return nil
}
