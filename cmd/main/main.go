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
	app := &cli.App{
		Name:  "telegram message process",
		Usage: "A Civial Aviation Authority Telegram Message Processor",
		Before: func(c *cli.Context) error {
			return nil
		},
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
	return app
}

func overrideConfig(c *cli.Context, cfg *config.Config) {
	if c.IsSet("nats") {
		cfg.Nats.URL = c.String("nats")
		fmt.Printf("Overriding nats url to %s\n", cfg.Nats.URL)
	}
	if c.IsSet("topic") {
		cfg.Subscription.Topic = c.String("topic")
		fmt.Printf("Overriding nats topic to %s\n", cfg.Subscription.Topic)
	}
}

func executeListen(c *cli.Context) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("error loading configuration: %w", err)
	}

	if err := config.ValidateConfig(cfg); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	overrideConfig(c, cfg)
	fmt.Println("Loaded configuration successfully")

	log := utils.GetLogger()
	log.Info("Starting nats subscriber")

	js, err := natsinfra.NewJetStream(&cfg.Nats)
	if err != nil {
		return fmt.Errorf("failed to create JetStream context: %w", err)
	}

	if cfg.Nats.JetStream.AutoProvision {
		streamConfig := natsinfra.CreateStreamConfig(&cfg.Nats)
		if err := natsinfra.EnsureStream(js, streamConfig); err != nil {
			return fmt.Errorf("failed to ensure stream: %w", err)
		}
	}

	publisher, err := natsinfra.NewPublisher(js, cfg)
	if err != nil {
		return fmt.Errorf("failed to create publisher: %w", err)
	}

	pool, err := db.NewConnectionPool(&cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to create database connection pool: %w", err)
	}
	defer pool.Close()

	repository := postgres.NewTelegramRepository(pool)

	handler := natsinfra.NewHandler(cfg, publisher, repository)

	consumer, err := natsinfra.NewConsumer(js, cfg, handler)
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}
	defer consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Infof("Received signal %v, shutting down gracefully...", sig)
		cancel()
	}()

	return consumer.Subscribe(ctx)
}
