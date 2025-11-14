package postgres

import (
	"caatsm/internal/infra/config"
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ProvideDB creates a PostgreSQL connection pool
func ProvideDB(cfg *config.Config) (*pgxpool.Pool, error) {
	ctx := context.Background()
	
	poolConfig, err := pgxpool.ParseConfig(cfg.Postgres.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres URL: %w", err)
	}
	
	poolConfig.MaxConns = int32(cfg.Postgres.MaxConns)
	poolConfig.MinConns = int32(cfg.Postgres.MinConns)
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = time.Minute * 30
	
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}
	
	// Test connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	
	return pool, nil
}

