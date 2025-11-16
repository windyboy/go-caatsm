package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/toml"
	envprovider "github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Config holds all application configuration
type Config struct {
	NATS       NATSConfig       `koanf:"nats"`
	Postgres   PostgresConfig   `koanf:"postgres"`
	App        AppConfig        `koanf:"app"`
	Log        LogConfig        `koanf:"log"`
	Publisher  PublisherConfig  `koanf:"publisher"`
	Telemetry  TelemetryConfig  `koanf:"telemetry"`
	Monitoring MonitoringConfig `koanf:"monitoring"`
	DLQ        DLQConfig        `koanf:"dlq"`
	// Legacy fields for backward compatibility during migration
	Subscription SubscriptionConfig `koanf:"subscription"`
	Timeouts     TimeoutsConfig     `koanf:"timeouts"`
}

// NATSConfig holds NATS/JetStream configuration
type NATSConfig struct {
	URL           string              `koanf:"url"`
	Mode          string              `koanf:"mode"`
	Stream        string              `koanf:"stream"`
	Consumer      string              `koanf:"consumer"`
	StreamLimits  StreamLimitsConfig  `koanf:"stream_limits"`
	ConsumerRules ConsumerRulesConfig `koanf:"consumer_rules"`
	// Legacy fields
	Client  string `koanf:"client"`
	Cluster string `koanf:"cluster"`
}

// StreamLimitsConfig defines JetStream retention controls.
type StreamLimitsConfig struct {
	MaxMsgs  int64         `koanf:"max_msgs"`
	MaxBytes int64         `koanf:"max_bytes"`
	MaxAge   time.Duration `koanf:"max_age"`
	Discard  string        `koanf:"discard"`
	Storage  string        `koanf:"storage"`
	Replicas int           `koanf:"replicas"`
}

// ConsumerRulesConfig captures consumer-level options.
type ConsumerRulesConfig struct {
	MaxDeliver    int             `koanf:"max_deliver"`
	AckWait       time.Duration   `koanf:"ack_wait"`
	MaxAckPending int             `koanf:"max_ack_pending"`
	DeliverPolicy string          `koanf:"deliver_policy"`
	ReplayPolicy  string          `koanf:"replay_policy"`
	Backoff       []time.Duration `koanf:"backoff"`
	StartSequence uint64          `koanf:"start_sequence"`
	StartTime     string          `koanf:"start_time"`
}

// PostgresConfig holds PostgreSQL configuration
type PostgresConfig struct {
	URL      string `koanf:"url"`
	MaxConns int32  `koanf:"max_conns"`
	MinConns int32  `koanf:"min_conns"`
}

// AppConfig holds application-level configuration
type AppConfig struct {
	BatchSize       int           `koanf:"batch_size"`
	BatchTimeout    time.Duration `koanf:"batch_timeout"`
	MonitorInterval time.Duration `koanf:"monitor_interval"`
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

// TelemetryConfig controls tracing/metrics exporters.
type TelemetryConfig struct {
	Enabled  bool   `koanf:"enabled"`
	Endpoint string `koanf:"endpoint"`
	Insecure bool   `koanf:"insecure"`
}

// DLQConfig defines the dead-letter queue routing for poison/permanent messages.
// If Enabled is true and Subject is non-empty, permanent failures will be published
// to the configured subject for offline processing.
type DLQConfig struct {
	Enabled bool   `koanf:"enabled"`
	Subject string `koanf:"subject"`
}

// MonitoringConfig controls the lightweight HTTP server that exposes health and metrics endpoints.
type MonitoringConfig struct {
	Disabled      bool          `koanf:"disabled"`
	Addr          string        `koanf:"addr"`
	EnableMetrics bool          `koanf:"enable_metrics"`
	EnableHealth  bool          `koanf:"enable_health"`
	ReadTimeout   time.Duration `koanf:"read_timeout"`
	WriteTimeout  time.Duration `koanf:"write_timeout"`
	HealthTimeout time.Duration `koanf:"health_timeout"`
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
	if cfg.App.MonitorInterval == 0 {
		cfg.App.MonitorInterval = 30 * time.Second
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
	if cfg.NATS.Mode == "" {
		cfg.NATS.Mode = "jetstream"
	} else {
		cfg.NATS.Mode = strings.ToLower(cfg.NATS.Mode)
	}
	if cfg.NATS.Stream == "" {
		cfg.NATS.Stream = "TELEGRAM"
	}
	if cfg.NATS.Consumer == "" {
		cfg.NATS.Consumer = "telegram-consumer"
	}
	if cfg.NATS.StreamLimits.MaxMsgs == 0 {
		cfg.NATS.StreamLimits.MaxMsgs = 100000
	}
	if cfg.NATS.StreamLimits.MaxBytes == 0 {
		cfg.NATS.StreamLimits.MaxBytes = 64 * 1024 * 1024
	}
	if cfg.NATS.StreamLimits.MaxAge == 0 {
		cfg.NATS.StreamLimits.MaxAge = 24 * time.Hour
	}
	if cfg.NATS.StreamLimits.Discard == "" {
		cfg.NATS.StreamLimits.Discard = "old"
	}
	if cfg.NATS.StreamLimits.Storage == "" {
		cfg.NATS.StreamLimits.Storage = "file"
	}
	if cfg.NATS.StreamLimits.Replicas == 0 {
		cfg.NATS.StreamLimits.Replicas = 1
	}
	if cfg.NATS.ConsumerRules.MaxDeliver == 0 {
		cfg.NATS.ConsumerRules.MaxDeliver = 5
	}
	if cfg.Timeouts.AckWait == 0 {
		cfg.Timeouts.AckWait = 30 * time.Second
	}
	if cfg.NATS.ConsumerRules.AckWait == 0 {
		cfg.NATS.ConsumerRules.AckWait = cfg.Timeouts.AckWait
	}
	if cfg.NATS.ConsumerRules.MaxAckPending == 0 {
		cfg.NATS.ConsumerRules.MaxAckPending = 1024
	}
	if cfg.NATS.ConsumerRules.DeliverPolicy == "" {
		cfg.NATS.ConsumerRules.DeliverPolicy = "all"
	}
	if cfg.NATS.ConsumerRules.ReplayPolicy == "" {
		cfg.NATS.ConsumerRules.ReplayPolicy = "instant"
	}
	if cfg.Telemetry.Endpoint == "" {
		cfg.Telemetry.Endpoint = ""
	}

	if !cfg.Monitoring.Disabled && cfg.Monitoring.Addr == "" && !cfg.Monitoring.EnableHealth && !cfg.Monitoring.EnableMetrics {
		cfg.Monitoring.Addr = ":2112"
		cfg.Monitoring.EnableHealth = true
		cfg.Monitoring.EnableMetrics = true
	}
	if cfg.Monitoring.ReadTimeout == 0 {
		cfg.Monitoring.ReadTimeout = 5 * time.Second
	}
	if cfg.Monitoring.WriteTimeout == 0 {
		cfg.Monitoring.WriteTimeout = 5 * time.Second
	}
	if cfg.Monitoring.HealthTimeout == 0 {
		cfg.Monitoring.HealthTimeout = 2 * time.Second
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
	switch strings.ToLower(c.NATS.Mode) {
	case "", "jetstream", "core":
	default:
		return fmt.Errorf("nats.mode must be 'jetstream' or 'core'")
	}
	if c.NATS.Stream == "" {
		return fmt.Errorf("nats.stream is required")
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
	if c.App.MonitorInterval <= 0 {
		return fmt.Errorf("app.monitor_interval must be greater than 0")
	}
	if c.NATS.StreamLimits.MaxMsgs < 0 {
		return fmt.Errorf("nats.stream_limits.max_msgs must be >= 0")
	}
	if c.NATS.StreamLimits.MaxBytes < 0 {
		return fmt.Errorf("nats.stream_limits.max_bytes must be >= 0")
	}
	if c.NATS.StreamLimits.MaxAge < 0 {
		return fmt.Errorf("nats.stream_limits.max_age must be >= 0")
	}
	if c.NATS.StreamLimits.Replicas <= 0 {
		return fmt.Errorf("nats.stream_limits.replicas must be greater than 0")
	}
	if c.NATS.ConsumerRules.MaxDeliver <= 0 {
		return fmt.Errorf("nats.consumer.max_deliver must be greater than 0")
	}
	if c.NATS.ConsumerRules.AckWait <= 0 {
		return fmt.Errorf("nats.consumer.ack_wait must be greater than 0")
	}
	if c.NATS.ConsumerRules.MaxAckPending < 0 {
		return fmt.Errorf("nats.consumer.max_ack_pending must be >= 0")
	}
	switch strings.ToLower(c.NATS.ConsumerRules.DeliverPolicy) {
	case "", "all", "new", "last", "last_per_subject", "sequence", "time":
	default:
		return fmt.Errorf("nats.consumer.deliver_policy must be one of all,new,last,last_per_subject,sequence,time")
	}
	switch strings.ToLower(c.NATS.ConsumerRules.ReplayPolicy) {
	case "", "instant", "original":
	default:
		return fmt.Errorf("nats.consumer.replay_policy must be instant or original")
	}
	if c.NATS.ConsumerRules.StartTime != "" {
		if _, err := time.Parse(time.RFC3339, c.NATS.ConsumerRules.StartTime); err != nil {
			return fmt.Errorf("nats.consumer.start_time must be RFC3339: %w", err)
		}
	}
	for _, d := range c.NATS.ConsumerRules.Backoff {
		if d < 0 {
			return fmt.Errorf("nats.consumer.backoff durations must be >= 0")
		}
	}
	if c.Telemetry.Endpoint == "" && c.Telemetry.Enabled {
		return fmt.Errorf("telemetry.endpoint is required when telemetry.enabled=true")
	}
	if c.Monitoring.ReadTimeout < 0 {
		return fmt.Errorf("monitoring.read_timeout must be >= 0")
	}
	if c.Monitoring.WriteTimeout < 0 {
		return fmt.Errorf("monitoring.write_timeout must be >= 0")
	}
	if c.Monitoring.HealthTimeout < 0 {
		return fmt.Errorf("monitoring.health_timeout must be >= 0")
	}
	return nil
}

// ProvideConfig is a Wire provider function
func ProvideConfig() (*Config, error) {
	return LoadConfig()
}

// EffectiveSubscriptionTopic returns the active subscription subject.
// Retains support for legacy config.Subscription fields while allowing
// future consolidation.
func (c *Config) EffectiveSubscriptionTopic() string {
	if c == nil {
		return ""
	}
	if topic := c.Subscription.Topic; topic != "" {
		return topic
	}
	return "telegram.>"
}
