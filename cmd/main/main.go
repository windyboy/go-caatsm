package main

import (
	"caatsm/internal/infra/config"
	"caatsm/pkg/di"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"
)

func main() {
	app := setupApp()
	if err := app.Run(os.Args); err != nil {
		fmt.Printf("Error running application: %v\n", err)
		os.Exit(1)
	}
}

func setupApp() *cli.App {
	return &cli.App{
		Name:  "telegram message process",
		Usage: "A Civil Aviation Authority Telegram Message Processor",
		Commands: []*cli.Command{
			{
				Name:  "listen",
				Usage: "Listen to nats messages",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "nats",
						Aliases: []string{"n"},
						Usage:   "Nats server address",
						Value:   "nats://localhost:4222",
						EnvVars: []string{"NATS_SERVER"},
					},
					&cli.StringFlag{
						Name:    "topic",
						Aliases: []string{"t"},
						Usage:   "Nats topic to listen to",
						Value:   "telegram.serial",
						EnvVars: []string{"NATS_SUBJECT"},
					},
				},
				Action: executeListen,
			},
		},
	}
}

func executeListen(c *cli.Context) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if flagURL := c.String("nats"); flagURL != "" {
		cfg.NATS.URL = flagURL
	}

	if flagTopic := c.String("topic"); flagTopic != "" {
		cfg.Subscription.Topic = flagTopic
	}

	// Initialize dependencies using Wire
	processor, consumer, err := di.InitializeAppWithConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize app: %w", err)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start consumer in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := consumer.Start(ctx); err != nil {
			errChan <- fmt.Errorf("consumer error: %w", err)
		}
	}()

	var runErr error

	// Wait for signal or error
	select {
	case sig := <-sigChan:
		fmt.Printf("Received signal: %v, shutting down...\n", sig)
		cancel()
	case err := <-errChan:
		cancel()
		if err != nil && !errors.Is(err, context.Canceled) {
			runErr = err
		}
	}

	waitTimeout := 5 * time.Second
	select {
	case err := <-errChan:
		if err != nil && !errors.Is(err, context.Canceled) {
			runErr = err
		}
	case <-time.After(waitTimeout):
		fmt.Printf("Timed out waiting for consumer shutdown after %s\n", waitTimeout)
	}

	if err := consumer.Shutdown(context.Background()); err != nil {
		if runErr == nil {
			runErr = fmt.Errorf("failed to drain NATS connection: %w", err)
		}
	}

	// Note: processor is initialized but not directly used here
	_ = processor

	return runErr
}
