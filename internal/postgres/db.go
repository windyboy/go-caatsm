package postgres

import (
	"caatsm/internal/config"
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func sanitizePostgresURLForLogging(poolConfig *pgxpool.Config) string {
	if poolConfig == nil || poolConfig.ConnConfig == nil {
		return "postgres://***@***/***"
	}

	host := poolConfig.ConnConfig.Host
	port := poolConfig.ConnConfig.Port
	database := poolConfig.ConnConfig.Database

	if host == "" {
		host = "***"
	}
	if port == 0 {
		port = 5432
	}
	if database == "" {
		database = "***"
	}

	return fmt.Sprintf("postgres://***@%s:%d/%s", host, port, database)
}

// OpenPool creates and verifies a PostgreSQL connection pool.
func OpenPool(ctx context.Context, cfg *config.Config, logger *zap.Logger) (*pgxpool.Pool, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.Postgres.URL)
	if err != nil {
		logger.Error("failed to parse postgres URL", zap.Error(err))
		return nil, fmt.Errorf("failed to parse postgres URL: %w", err)
	}

	safeURL := sanitizePostgresURLForLogging(poolConfig)

	poolConfig.MaxConns = cfg.Postgres.MaxConns
	poolConfig.MinConns = cfg.Postgres.MinConns
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		logger.Error("failed to create connection pool",
			zap.String("url", safeURL),
			zap.Int32("max_conns", cfg.Postgres.MaxConns),
			zap.Int32("min_conns", cfg.Postgres.MinConns),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		logger.Error("failed to ping database", zap.String("url", safeURL), zap.Error(err))
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Connected to PostgreSQL",
		zap.String("url", safeURL),
		zap.Int32("max_conns", cfg.Postgres.MaxConns),
		zap.Int32("min_conns", cfg.Postgres.MinConns),
	)
	return pool, nil
}
