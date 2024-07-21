package main

import (
	"caatsm/internal/config"
	"caatsm/internal/nats"
	"caatsm/pkg/utils"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		utils.Logger.WithError(err).Fatal("Error loading config")
	}

	if err := config.ValidateConfig(cfg); err != nil {
		utils.Logger.WithError(err).Fatal("Config validation error")
	}

	utils.Logger.Info("Loaded configuration successfully")

	subscribe := nats.NewNatsHandler(cfg)
	subscribe.Subscribe()
}
