package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"caatsm/internal/config"
	"caatsm/internal/infrastructure/db"
	natsinfra "caatsm/internal/infrastructure/nats"
	"caatsm/internal/repository/postgres"
	"caatsm/pkg/utils"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	if err := config.ValidateConfig(cfg); err != nil {
		fmt.Printf("Invalid configuration: %v\n", err)
		os.Exit(1)
	}

	log := utils.GetLogger()
	log.Info("Starting message worker")

	// Initialize database connection pool
	pool, err := db.NewConnectionPool(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to create database connection pool: %v", err)
	}
	defer pool.Close()

	// Initialize repository
	repo := postgres.NewTelegramRepository(pool)

	// Initialize NATS JetStream
	js, err := natsinfra.NewJetStream(&cfg.Nats)
	if err != nil {
		log.Fatalf("Failed to create JetStream context: %v", err)
	}

	// Ensure stream exists
	if cfg.Nats.JetStream.AutoProvision {
		streamConfig := natsinfra.CreateStreamConfig(&cfg.Nats)
		if err := natsinfra.EnsureStream(js, streamConfig); err != nil {
			log.Fatalf("Failed to ensure stream: %v", err)
		}
	}

	// Initialize publisher
	publisher, err := natsinfra.NewPublisher(js, cfg)
	if err != nil {
		log.Fatalf("Failed to create publisher: %v", err)
	}

	// Initialize message handler
	handler := natsinfra.NewHandler(cfg, publisher, repo)

	// Initialize consumer
	consumer, err := natsinfra.NewConsumer(js, cfg, handler)
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Infof("Received signal %v, shutting down gracefully...", sig)
		cancel()
	}()

	// Start consuming messages
	log.Info("Starting message consumption")
	if err := consumer.Subscribe(ctx); err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	// Wait for context cancellation
	<-ctx.Done()
	log.Info("Worker stopped")
}
