package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/knadh/koanf/v2"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/file"
	envprovider "github.com/knadh/koanf/providers/env"
)

// Config holds all application configuration
type Config struct {
	NATS     NATSConfig     `koanf:"nats"`
	Postgres PostgresConfig `koanf:"postgres"`
	App      AppConfig      `koanf:"app"`
	Log      LogConfig      `koanf:"log"`
	Publisher PublisherConfig `koanf:"publisher"`
	// Legacy fields for backward compatibility during migration
	Subscription SubscriptionConfig `koanf:"subscription"`
	Timeouts     TimeoutsConfig     `koanf:"timeouts"`
}

// NATSConfig holds NATS/JetStream configuration
type NATSConfig struct {
	URL      string `koanf:"url"`
	Stream   string `koanf:"stream"`
	Consumer string `koanf:"consumer"`
	// Legacy fields
	Client  string `koanf:"client"`
	Cluster string `koanf:"cluster"`
}

// PostgresConfig holds PostgreSQL configuration
type PostgresConfig struct {
	URL      string `koanf:"url"`
	MaxConns int32  `koanf:"max_conns"`
	MinConns int32  `koanf:"min_conns"`
}

// AppConfig holds application-level configuration
type AppConfig struct {
	BatchSize    int           `koanf:"batch_size"`
	BatchTimeout time.Duration `koanf:"batch_timeout"`
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level  string `koanf:"level"`
	Format string `koanf:"format"` // json or console
}

// PublisherConfig holds publisher configuration
type PublisherConfig struct {
	Topic string `koanf:"topic"`
}

// SubscriptionConfig holds subscription configuration (legacy)
type SubscriptionConfig struct {
	Topic      string `koanf:"topic"`
	QueueGroup string `koanf:"queue_group"`
}

// TimeoutsConfig holds timeout configuration (legacy)
type TimeoutsConfig struct {
	Server        time.Duration `koanf:"server"`
	ReconnectWait time.Duration `koanf:"reconnect_wait"`
	Close         time.Duration `koanf:"close"`
	AckWait       time.Duration `koanf:"ack_wait"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig() (*Config, error) {
	k := koanf.New(".")

	// Determine environment
	env := os.Getenv("GO_ENV")
	if env == "" {
		env = "dev"
	}

	// Load from TOML file
	configFile := fmt.Sprintf("configs/config.%s.toml", env)
	if err := k.Load(file.Provider(configFile), toml.Parser()); err != nil {
		return nil, fmt.Errorf("error loading config file '%s': %w", configFile, err)
	}

	// Load from environment variables with CAATSM_ prefix
	envProvider := envprovider.Provider("CAATSM_", ".", func(s string) string {
		// Convert CAATSM_NATS_URL to nats.url
		s = strings.TrimPrefix(s, "CAATSM_")
		return strings.ToLower(strings.ReplaceAll(s, "_", "."))
	})
	if err := k.Load(envProvider, nil); err != nil {
		// Environment variables are optional, so we don't fail if they're not present
		// This allows the config to work with just the file
	}

	// Unmarshal into Config struct
	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Set defaults
	if cfg.App.BatchSize == 0 {
		cfg.App.BatchSize = 50
	}
	if cfg.App.BatchTimeout == 0 {
		cfg.App.BatchTimeout = 2 * time.Second
	}
	if cfg.Postgres.MaxConns == 0 {
		cfg.Postgres.MaxConns = 10
	}
	if cfg.Postgres.MinConns == 0 {
		cfg.Postgres.MinConns = 2
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	if cfg.Log.Format == "" {
		cfg.Log.Format = "json"
	}
	if cfg.NATS.Stream == "" {
		cfg.NATS.Stream = "TELEGRAM"
	}
	if cfg.NATS.Consumer == "" {
		cfg.NATS.Consumer = "telegram-consumer"
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.NATS.URL == "" {
		return fmt.Errorf("nats.url is required")
	}
	if c.Subscription.Topic == "" && c.NATS.Stream == "" {
		return fmt.Errorf("subscription.topic or nats.stream is required")
	}
	if c.Publisher.Topic == "" {
		return fmt.Errorf("publisher.topic is required")
	}
	if c.Postgres.URL == "" {
		return fmt.Errorf("postgres.url is required")
	}
	if c.App.BatchSize <= 0 {
		return fmt.Errorf("app.batch_size must be greater than 0")
	}
	if c.App.BatchTimeout <= 0 {
		return fmt.Errorf("app.batch_timeout must be greater than 0")
	}
	return nil
}

// ProvideConfig is a Wire provider function
func ProvideConfig() (*Config, error) {
	return LoadConfig()
}

