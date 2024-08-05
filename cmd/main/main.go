package main

import (
	"caatsm/internal/config"
	"caatsm/internal/handlers"
	"caatsm/internal/nats"
	"caatsm/pkg/utils"
	"os"

	"fmt"

	"github.com/urfave/cli/v2"
)

var (
	cfg *config.Config
)

func main() {
	app := setupApp()
	if err := app.Run(os.Args); err != nil {
		fmt.Printf("Error running application: %v\n", err)
	}
}

func setupApp() *cli.App {
	app := &cli.App{
		Name:  "telegram message process",
		Usage: "A Civial Aviation Authority Telegram message processor",
		Before: func(c *cli.Context) error {
			cfg, err := config.LoadConfig()
			if err != nil {
				fmt.Printf("Error loading configuration: %v\n", err)
				return err
			}

			if err := config.ValidateConfig(cfg); err != nil {
				fmt.Printf("Invalid configuration: %v\n", err)
				return err
			}
			overrideConfig(c)
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:  "listen",
				Usage: "Listen to nats messages",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "nats",
						Usage:   "Nats server address",
						Value:   "nats://localhost:4222",
						EnvVars: []string{"NATS_SERVER"},
					},
					&cli.StringFlag{
						Name:    "topic",
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

func overrideConfig(c *cli.Context) {
	if c.IsSet("nats") {
		cfg.Nats.URL = c.String("nats")
	}
	if c.IsSet("topic") {
		cfg.Subscription.Topic = c.String("topic")
	}
}

func executeListen(c *cli.Context) error {
	// cfg, err := config.LoadConfig()
	log := utils.GetLogger()
	fmt.Println("Loaded configuration successfully")
	log.Info("Starting nats subscriber")
	handler := handlers.NewNatsHandler(cfg)

	nats.Subscribe(cfg, &handlers.PlainTextMarshaler{}, handler)
	return nil
}
