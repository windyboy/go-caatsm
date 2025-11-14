package di

import (
	"context"
	"fmt"

	"caatsm/internal/config"
	applog "caatsm/internal/infra/log"
	"caatsm/internal/infrastructure/db"
	natsinfra "caatsm/internal/infrastructure/nats"
	"caatsm/internal/repository/postgres"
)

// WorkerApp bundles all dependencies required to run the message worker.
type WorkerApp struct {
	Config   *config.Config
	consumer *natsinfra.Consumer
	cleanup  []func() error
}

// InitializeApp wires together configuration, infrastructure clients, and handlers.
func InitializeApp() (*WorkerApp, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("error loading configuration: %w", err)
	}
	if err := config.ValidateConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	pool, err := db.NewConnectionPool(&cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to create database connection pool: %w", err)
	}

	js, err := natsinfra.NewJetStream(&cfg.Nats)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	if cfg.Nats.JetStream.AutoProvision {
		streamConfig := natsinfra.CreateStreamConfig(&cfg.Nats)
		if err := natsinfra.EnsureStream(js, streamConfig); err != nil {
			pool.Close()
			return nil, fmt.Errorf("failed to ensure stream: %w", err)
		}
	}

	publisher, err := natsinfra.NewPublisher(js, cfg)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to create publisher: %w", err)
	}

	repository := postgres.NewTelegramRepository(pool)
	handler := natsinfra.NewHandler(cfg, publisher, repository)

	consumer, err := natsinfra.NewConsumer(js, cfg, handler)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	app := &WorkerApp{
		Config:   cfg,
		consumer: consumer,
		cleanup: []func() error{
			consumer.Close,
			func() error {
				pool.Close()
				return nil
			},
		},
	}

	return app, nil
}

// Run starts the consumer loop.
func (a *WorkerApp) Run(ctx context.Context) error {
	return a.consumer.Subscribe(ctx)
}

// Close releases all managed resources.
func (a *WorkerApp) Close() {
	for _, fn := range a.cleanup {
		if err := fn(); err != nil {
			applog.Sugared().Warnf("cleanup error: %v", err)
		}
	}
}
