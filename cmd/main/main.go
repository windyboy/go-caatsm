package main

import (
	"caatsm/pkg/di"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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
						Value:   "Telegram.Serial",
						EnvVars: []string{"NATS_SUBJECT"},
					},
				},
				Action: executeListen,
			},
		},
	}
}

func executeListen(c *cli.Context) error {
	// Initialize dependencies using Wire
	processor, consumer, err := di.InitializeApp()
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

	// Wait for signal or error
	select {
	case sig := <-sigChan:
		fmt.Printf("Received signal: %v, shutting down...\n", sig)
		cancel()
	case err := <-errChan:
		return err
	}

	// Note: processor is initialized but not directly used here
	// It's used by the consumer internally
	_ = processor

	return nil
}
