package db

import (
	"context"
	"fmt"
	"time"

	"caatsm/internal/config"
	"caatsm/pkg/utils"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewConnectionPool creates a new PostgreSQL connection pool
func NewConnectionPool(cfg *config.DatabaseConfig) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Configure connection pool
	if cfg.MaxConns > 0 {
		poolConfig.MaxConns = int32(cfg.MaxConns)
	}
	if cfg.MinConns > 0 {
		poolConfig.MinConns = int32(cfg.MinConns)
	}
	if cfg.MaxConnLifetime > 0 {
		poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	}

	// Set connection health check
	poolConfig.HealthCheckPeriod = 1 * time.Minute
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log := utils.GetSugaredLogger()
	log.Infof("Database connection pool created: %s@%s:%d/%s", cfg.User, cfg.Host, cfg.Port, cfg.Database)

	return pool, nil
}

// HealthCheck verifies the database connection is healthy
func HealthCheck(ctx context.Context, pool *pgxpool.Pool) error {
	return pool.Ping(ctx)
}

