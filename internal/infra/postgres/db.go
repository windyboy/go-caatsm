package postgres

import (
	"caatsm/internal/infra/config"
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// ProvideDB creates a PostgreSQL connection pool
func ProvideDB(cfg *config.Config, logger *zap.Logger) (*pgxpool.Pool, error) {
	ctx := context.Background()

	poolConfig, err := pgxpool.ParseConfig(cfg.Postgres.URL)
	if err != nil {
		logger.Error("failed to parse postgres URL",
			zap.String("url", cfg.Postgres.URL),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to parse postgres URL: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.Postgres.MaxConns)
	poolConfig.MinConns = int32(cfg.Postgres.MinConns)
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = time.Minute * 30

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		logger.Error("failed to create connection pool",
			zap.String("url", cfg.Postgres.URL),
			zap.Int32("max_conns", cfg.Postgres.MaxConns),
			zap.Int32("min_conns", cfg.Postgres.MinConns),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		logger.Error("failed to ping database",
			zap.String("url", cfg.Postgres.URL),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Log the connection pool
	logger.Info("Connected to PostgreSQL", zap.Int32("max_conns", cfg.Postgres.MaxConns), zap.Int32("min_conns", cfg.Postgres.MinConns), zap.String("url", cfg.Postgres.URL))
	return pool, nil
}
