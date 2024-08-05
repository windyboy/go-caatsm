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

func main() {
	app := setupApp()
	if err := app.Run(os.Args); err != nil {
		fmt.Printf("Error running application: %v\n", err)
	}
}

func setupApp() *cli.App {
	app := &cli.App{
		Name:  "serial-read",
		Usage: "A serial port reading CLI application",
		Before: func(c *cli.Context) error {
			// parameter = config.GetParameter()
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:   "listen",
				Usage:  "Listen to nats messages",
				Action: executeListen,
			},
		},
	}
	return app
}

func executeListen(c *cli.Context) error {
	cfg, err := config.LoadConfig()
	log := utils.GetLogger()
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		return err
	}

	if err := config.ValidateConfig(cfg); err != nil {
		fmt.Printf("Invalid configuration: %v\n", err)
		return err
	}

	fmt.Println("Loaded configuration successfully")

	log.Info("Starting nats subscriber")
	handler := handlers.NewNatsHandler(cfg)

	nats.Subscribe(cfg, &handlers.PlainTextMarshaler{}, handler)
	return nil
}
