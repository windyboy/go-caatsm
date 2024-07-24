package main

import (
	"caatsm/internal/config"
	"caatsm/internal/nats"
	"caatsm/pkg/utils"

	"fmt"
)

func main() {
	cfg, err := config.LoadConfig()
	log := utils.GetLogger()
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		return
	}

	if err := config.ValidateConfig(cfg); err != nil {
		fmt.Printf("Invalid configuration: %v\n", err)
		return
	}

	fmt.Println("Loaded configuration successfully")

	log.Info("Starting nats subscriber")
	subscribe := nats.NewNatsHandler(cfg)
	subscribe.Subscribe()
}
