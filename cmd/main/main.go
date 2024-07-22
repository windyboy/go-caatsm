package main

import (
	"caatsm/internal/config"
	"caatsm/internal/nats"

	"fmt"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		return
	}

	if err := config.ValidateConfig(cfg); err != nil {
		fmt.Printf("Invalid configuration: %v\n", err)
		return
	}

	fmt.Println("Loaded configuration successfully")

	subscribe := nats.NewNatsHandler(cfg)
	subscribe.Subscribe()
}
