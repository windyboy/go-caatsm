// Package main is the entry point for the caatsm application.
// It handles command-line interface (CLI) interactions, configuration loading,
// and initialization of the NATS message listener.
package main

import (
	"caatsm/internal/config"
	"caatsm/internal/nats"
	"caatsm/internal/repository"
	"caatsm/pkg/utils"
	"os"

	"fmt"

	"github.com/urfave/cli/v2"
)

const configKey = "appConfig" // key for storing config in cli.App.Metadata

// main is the primary entry point of the application.
// It sets up the CLI application and executes it.
func main() {
	log := utils.GetSugaredLogger() // Initialize logger early
	app := setupApp()
	if err := app.Run(os.Args); err != nil {
		log.Fatalf("Error running application: %v", err) // Use Fatalf to exit with status 1
	}
}

// setupApp configures and returns the urfave/cli App.
// It defines the application's commands, flags, and actions,
// including the 'listen' command for starting the NATS message listener.
// It also includes a Before action to load initial configuration.
func setupApp() *cli.App {
	log := utils.GetSugaredLogger()
	app := &cli.App{
		Name:  "telegram-message-processor", // More conventional app name
		Usage: "A Civil Aviation Authority Telegram Message Processor",
		// Before action is executed before any command action.
		// Ideal for loading configuration and setting up shared resources.
		Before: func(c *cli.Context) error {
			log.Info("Loading configuration...")
			cfg, err := config.LoadConfig()
			if err != nil {
				log.Errorf("Error loading initial configuration: %v", err)
				return fmt.Errorf("error loading initial configuration: %w", err)
			}

			// Override config with flags from the specific command being run.
			// Note: This means overrideConfig needs to be smart about which command's flags to check,
			// or flags need to be global if they apply to all commands before this 'Before' action.
			// For 'listen' command flags, this override might be better placed in 'listen' command's Before or Action.
			// However, if we want a single config object modified by flags, this is one place to do it.
			// Let's assume overrideConfig will be called from the command's action for now,
			// or flags are made global if truly global overrides are needed here.
			// For this iteration, we'll load base config here and let commands override.
			// Storing the loaded config in App.Metadata to be accessible by commands.
			if c.App.Metadata == nil {
				c.App.Metadata = make(map[string]interface{})
			}
			c.App.Metadata[configKey] = cfg
			log.Info("Initial configuration loaded.")
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:  "listen",
				Usage: "Listen to NATS messages and process them",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "nats-url", // More descriptive flag name
						Aliases: []string{"n"},
						Usage:   "NATS server URL `URL`",
						Value:   "nats://localhost:4222", // Default value
						EnvVars: []string{"NATS_SERVER_URL"}, // Consistent env var naming
					},
					&cli.StringFlag{
						Name:    "nats-topic", // More descriptive flag name
						Aliases: []string{"t"},
						Usage:   "NATS subject/topic to listen to `TOPIC`",
						Value:   "Telegram.Serial",       // Default value
						EnvVars: []string{"NATS_SUBJECT"}, // Consistent env var naming
					},
				},
				Action: executeListen, // Action to execute for the listen command
			},
		},
	}
	return app
}

// overrideConfig applies command-line flag values to the configuration.
// overrideConfig applies command-line flag values from the `listen` command
// to the provided configuration object.
// cfg: The configuration object to be modified.
// c: The CLI context from which to retrieve flag values.
// log: A logger instance for logging override actions.
func overrideConfig(cfg *config.Config, c *cli.Context, log iface.Logger) {
	log.Info("Checking for configuration overrides from command flags...")
	if c.IsSet("nats-url") {
		newNatsURL := c.String("nats-url")
		if cfg.Nats.URL != newNatsURL {
			log.Infof("Overriding NATS URL from '%s' to '%s'", cfg.Nats.URL, newNatsURL)
			cfg.Nats.URL = newNatsURL
		}
	}
	if c.IsSet("nats-topic") {
		newNatsTopic := c.String("nats-topic")
		if cfg.Subscription.Topic != newNatsTopic {
			log.Infof("Overriding NATS topic from '%s' to '%s'", cfg.Subscription.Topic, newNatsTopic)
			cfg.Subscription.Topic = newNatsTopic
		}
	}
}

func executeListen(c *cli.Context) error {
	log := utils.GetSugaredLogger().With("command", "listen") // Logger specific to this command context
	log.Info("Executing listen command...")

	// Retrieve the base configuration loaded in the App's Before action.
	appCfg, ok := c.App.Metadata[configKey].(*config.Config)
	if !ok || appCfg == nil {
		log.Error("Configuration not found or invalid type in app metadata.")
		return fmt.Errorf("configuration not found in app metadata")
	}

	// Apply overrides from 'listen' command flags.
	overrideConfig(appCfg, c, log)

	// Validate the (potentially overridden) configuration.
	log.Info("Validating final configuration...")
	if err := config.ValidateConfig(appCfg); err != nil {
		log.Errorf("Invalid configuration after applying overrides: %v", err)
		return fmt.Errorf("invalid configuration: %w", err)
	}

	log.Infof("Configuration validated successfully. Final NATS URL: %s, Topic: %s", appCfg.Nats.URL, appCfg.Subscription.Topic)

	log.Info("Initializing NATS subscriber...")
	// The subscribe function will block if successful.
	// Any error during subscriber setup should be returned to stop the command.
	if err := subscribe(appCfg, log); err != nil {
		log.Errorf("NATS subscription failed: %v", err)
		return err // Error will be handled by app.Run, causing non-zero exit
	}

	log.Info("NATS subscriber finished.") // Should ideally not be reached if subscribe blocks indefinitely
	return nil
}

// subscribe initializes the NATS publisher, repository, message handler,
// and NATS subscriber, then starts the subscription process.
// This function is expected to block indefinitely as the subscriber listens for messages.
// appConfig: The application configuration containing NATS, repository, etc., settings.
// log: A logger instance for logging setup and operational messages.
// Returns an error if any part of the setup (publisher, repository, handler, subscriber) fails critically.
func subscribe(appConfig *config.Config, log iface.Logger) error {
	log.Info("Initializing NATS publisher...")
	pub := nats.NewPub(appConfig) // Pass appConfig
	if pub == nil {
		// Assuming NewPub could theoretically return nil if setup failed internally, though not explicit in its current usage.
		// Or it might panic, which would be caught by a defer recover higher up if implemented.
		log.Error("Failed to initialize NATS publisher.")
		return fmt.Errorf("failed to initialize NATS publisher")
	}

	log.Info("Initializing repository...")
	repo := repository.NewHasura(appConfig) // Pass appConfig
	if repo == nil {
		// Similar assumption for NewHasura
		log.Error("Failed to initialize repository.")
		return fmt.Errorf("failed to initialize repository")
	}

	log.Info("Initializing NATS message handler...")
	handler := nats.NewHandler(appConfig, pub, repo) // Pass appConfig
	if handler == nil {
		// Similar assumption for NewHandler
		log.Error("Failed to initialize NATS message handler.")
		return fmt.Errorf("failed to initialize NATS message handler")
	}

	log.Info("Setting up NATS subscription...")
	// Assuming NewSub might return an error or the subscriber object could be nil.
	subscriber := nats.NewSub(appConfig) // Pass appConfig
	if subscriber == nil {
		log.Error("Failed to create NATS subscriber.")
		return fmt.Errorf("failed to create NATS subscriber")
	}

	// The Subscribe method is expected to be blocking and run indefinitely.
	// If Subscribe itself can return an error (e.g., connection issues, invalid topic),
	// that would need to be handled here. The current interface doesn't show it returning an error.
	// If it panics on error, that's another scenario.
	// For now, we assume if we get to this point, it tries to connect and subscribe.
	log.Info("Starting NATS subscriber (this is a blocking call)...")
	subscriber.Subscribe(handler) // This blocks

	// This part will only be reached if Subscribe stops, which might indicate an issue or shutdown.
	log.Info("NATS subscription ended.")
	return nil // Or an error if Subscribe indicated one upon stopping.
}
