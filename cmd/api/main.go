package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"caatsm/internal/api"
	"caatsm/internal/config"
	"caatsm/internal/infrastructure/db"
	"caatsm/internal/repository/postgres"
	"caatsm/internal/service"
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
	log.Info("Starting API server")

	// Initialize database connection pool
	pool, err := db.NewConnectionPool(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to create database connection pool: %v", err)
	}
	defer pool.Close()

	// Initialize repository
	repo := postgres.NewTelegramRepository(pool)

	// Initialize service
	telegramService := service.NewTelegramService(repo)

	// Setup router
	e := api.SetupRouter(telegramService)

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port)

	// Graceful shutdown
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Infof("Received signal %v, shutting down gracefully...", sig)
		cancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := e.Shutdown(shutdownCtx); err != nil {
			log.Errorf("Error shutting down server: %v", err)
		}
	}()

	log.Infof("API server listening on %s", addr)
	if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Info("API server stopped")
}
