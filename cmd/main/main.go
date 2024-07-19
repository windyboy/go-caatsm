package main

import (
	"caatsm/internal/config"
	"caatsm/pkg/utils"

	"github.com/sirupsen/logrus"
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

	// Example usage
	utils.Logger.WithFields(logrus.Fields{
		"url": cfg.Nats.URL,
	}).Info("NATS configuration")
}
