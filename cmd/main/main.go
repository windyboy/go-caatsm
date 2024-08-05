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

func overrideConfig(c *cli.Context) {
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
		fmt.Printf("Error loading configuration: %v\n", err)
		return err
	}
	if err := config.ValidateConfig(cfg); err != nil {
		fmt.Printf("Invalid configuration: %v\n", err)
		return err
	}
	overrideConfig(c)
	fmt.Println("Loaded configuration successfully")
	log := utils.GetLogger()
	log.Info("Starting nats subscriber")
	handler := handlers.New(cfg)
	nats.Subscribe(cfg, &handlers.PlainTextMarshaler{}, handler)
	return nil
}
