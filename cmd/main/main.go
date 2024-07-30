package main

import (
	"caatsm/internal/config"
	"caatsm/internal/handlers"
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
	handler := handlers.NewNatsHandler(cfg)

	nats.Subscribe(cfg, &handlers.PlainTextMarshaler{}, handler)
}
