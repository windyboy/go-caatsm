// Package config provides structures and functions for loading, validating,
// and managing application configuration. It uses Viper for reading
// configuration files and environment variables.
package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// MyConfig is a global variable holding the loaded application configuration.
// Deprecated: Using global config variables is generally discouraged. Prefer passing config explicitly.
var MyConfig *Config

// Config is the main structure holding all application configuration.
type Config struct {
	Nats         NatsConfig         // NatsConfig holds NATS specific settings.
	Subscription SubscriptionConfig // SubscriptionConfig holds NATS subscription settings.
	Publisher    PublisherConfig    // PublisherConfig holds NATS publisher settings.
	Timeouts     TimeoutsConfig     // TimeoutsConfig holds various timeout settings.
	Hasura       HasuraConfig       // HasuraConfig holds Hasura GraphQL endpoint settings.
}

// NatsConfig holds configuration related to NATS server connection.
type NatsConfig struct {
	Client  string // Client is the NATS client ID.
	URL     string // URL is the NATS server URL (e.g., "nats://localhost:4222").
	Cluster string // Cluster is the NATS cluster ID (if applicable).
}

// SubscriptionConfig holds configuration for NATS subscriptions.
type SubscriptionConfig struct {
	Topic      string `mapstructure:"topic"`        // Topic is the NATS subject/topic to subscribe to.
	QueueGroup string `mapstructure:"queue_group"`  // QueueGroup is the NATS queue group name for durable subscriptions.
}

// PublisherConfig holds configuration for NATS publishing.
type PublisherConfig struct {
	Topic string `mapstructure:"topic"` // Topic is the NATS subject/topic to publish messages to.
}

// TimeoutsConfig holds various timeout settings used across the application.
type TimeoutsConfig struct {
	Server        time.Duration `mapstructure:"server"`         // Server general server timeout.
	ReconnectWait time.Duration `mapstructure:"reconnect_wait"` // ReconnectWait is the duration to wait before attempting to reconnect to NATS.
	Close         time.Duration `mapstructure:"close"`          // Close is the timeout for closing NATS connections gracefully.
	AckWait       time.Duration `mapstructure:"ack_wait"`       // AckWait is the duration to wait for an ACK from NATS JetStream.
}

// BodyConfig holds configuration for message body parsing patterns.
// Note: This struct is defined but not currently used within the Config struct.
type BodyConfig struct {
	Patterns []PatternConfig // Patterns is a slice of pattern configurations.
}

// PatternConfig defines a single regex pattern for parsing.
// Note: This struct is defined but not currently used within the Config struct.
type PatternConfig struct {
	Pattern    string         // Pattern is the regex string.
	Comments   string         // Comments provides a description for the pattern.
	Expression *regexp.Regexp // Expression is the compiled regex.
}

// HasuraConfig holds configuration for connecting to a Hasura GraphQL endpoint.
type HasuraConfig struct {
	Endpoint string // Endpoint is the URL of the Hasura GraphQL API.
	Secret   string // Secret is the admin secret key for Hasura (if required).
}

const (
	// EnvProd represents the production environment.
	EnvProd = "prod"
	// EnvDev represents the development environment.
	EnvDev = "dev"
	// EnvTest represents the testing environment.
	EnvTest = "test"
)

// SetMyConfig sets the global MyConfig variable.
// Deprecated: Using global config variables is generally discouraged. Prefer passing config explicitly.
func SetMyConfig(cfg *Config) {
	MyConfig = cfg
}

// GetMyConfig retrieves the global MyConfig variable.
// If MyConfig is nil, it attempts to load it using LoadConfig.
// Deprecated: Using global config variables is generally discouraged. Prefer passing config explicitly.
// This function also has basic error handling (prints to stdout) which is not ideal for library code.
func GetMyConfig() *Config {
	if MyConfig == nil {
		cfg, err := LoadConfig()
		if err != nil {
			// Using fmt.Printf here is not ideal for a library function.
			// Consider returning the error or using a proper logger.
			fmt.Printf("error loading config in GetMyConfig: %v\n", err)
		}
		MyConfig = cfg // MyConfig might be nil if LoadConfig failed
	}
	return MyConfig
}

// LoadConfig loads the application configuration from a TOML file based on the GO_ENV environment variable.
// It looks for "config.<GO_ENV>.toml" in the "./configs" directory.
// If GO_ENV is not set, it defaults to "dev".
// Environment variables with prefix "TELE_" can override config file values (e.g., TELE_NATS_URL overrides Nats.URL).
// Returns the loaded Config struct or an error if loading or unmarshalling fails.
func LoadConfig() (*Config, error) {
	// log := utils.Logger // Assuming a logger would be used here in a real scenario.
	env := os.Getenv("GO_ENV")
	if env == "" {
		env = "dev"
	}
	// log.Infof("Environment: %s", env)

	viper.SetConfigType("toml")
	viper.SetConfigName("config." + env)
	viper.AddConfigPath("configs")
	viper.SetEnvPrefix("tele")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		errMsg := fmt.Sprintf("error reading config file for environment '%s': %v", env, err)
		// log.Error(errMsg)
		return nil, fmt.Errorf(errMsg)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		errMsg := fmt.Sprintf("unable to decode config into struct for environment '%s': %v", env, err)
		// log.Error(errMsg)
		return nil, fmt.Errorf(errMsg)
	}
	return &config, nil
}

// ValidateConfig checks if essential fields in the Config struct are populated.
// cfg: The Config object to validate.
// Returns an error if any mandatory configuration is missing, otherwise nil.
func ValidateConfig(cfg *Config) error {
	// log := utils.Logger // Assuming a logger would be used here.

	if cfg.Nats.Client == "" {
		return fmt.Errorf("nats client is required")
	}
	if cfg.Nats.URL == "" {
		return fmt.Errorf("nats URL is required")
	}
	if cfg.Subscription.Topic == "" {
		return fmt.Errorf("subscription topic is required")
	}
	if cfg.Publisher.Topic == "" {
		return fmt.Errorf("publisher topic is required")
	}
	if cfg.Hasura.Endpoint == "" {
		return fmt.Errorf("hasura endpoint is required")
	}
	// Hasura secret might be optional if Hasura is not configured for auth,
	// but if an endpoint is given, a secret is usually expected.
	// For now, let's make it required if the endpoint is present.
	if cfg.Hasura.Endpoint != "" && cfg.Hasura.Secret == "" {
		return fmt.Errorf("hasura secret is required when hasura endpoint is provided")
	}

	// fmt.Println("config validation passed")
	return nil
}
