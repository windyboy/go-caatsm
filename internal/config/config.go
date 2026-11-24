package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/toml"
	envprovider "github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Config holds all application configuration.
type Config struct {
	NATS       NATSConfig       `koanf:"nats"`
	Postgres   PostgresConfig   `koanf:"postgres"`
	App        AppConfig        `koanf:"app"`
	Log        LogConfig        `koanf:"log"`
	Publisher  PublisherConfig  `koanf:"publisher"`
	Telemetry  TelemetryConfig  `koanf:"telemetry"`
	Monitoring MonitoringConfig `koanf:"monitoring"`
	DLQ        DLQConfig        `koanf:"dlq"`
	AFTN       AFTNConfig       `koanf:"aftn"`
	// Legacy fields for backward compatibility during migration
	Subscription SubscriptionConfig `koanf:"subscription"`
	Timeouts     TimeoutsConfig     `koanf:"timeouts"`
}

// NATSConfig holds NATS/JetStream configuration.
type NATSConfig struct {
	URL           string              `koanf:"url"`
	Mode          string              `koanf:"mode"`
	Stream        string              `koanf:"stream"`
	PublishStream string              `koanf:"publish_stream"`
	DLQStream     string              `koanf:"dlq_stream"`
	Consumer      string              `koanf:"consumer"`
	AutoProvision *bool               `koanf:"auto_provision"`
	StreamLimits  StreamLimitsConfig  `koanf:"stream_limits"`
	ConsumerRules ConsumerRulesConfig `koanf:"consumer_rules"`
	Auth          NATSAuthConfig      `koanf:"auth"`
	// Legacy fields
	Client  string `koanf:"client"`
	Cluster string `koanf:"cluster"`
}

// NATSAuthConfig holds NATS authentication configuration.
type NATSAuthConfig struct {
	// Token authentication (mutually exclusive with User/Password and CredentialsFile)
	Token string `koanf:"token"`

	// Credentials file authentication (mutually exclusive with Token and User/Password)
	CredentialsFile string `koanf:"credentials_file"`

	// User/Password authentication (mutually exclusive with Token and CredentialsFile)
	User     string `koanf:"user"`
	Password string `koanf:"password"`

	// TLS configuration
	TLSEnabled  bool   `koanf:"tls_enabled"`
	TLSCertFile string `koanf:"tls_cert_file"`
	TLSKeyFile  string `koanf:"tls_key_file"`
	TLSCAFile   string `koanf:"tls_ca_file"`
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

// PostgresConfig holds PostgreSQL configuration.
type PostgresConfig struct {
	URL      string `koanf:"url"`
	MaxConns int32  `koanf:"max_conns"`
	MinConns int32  `koanf:"min_conns"`
}

// AppConfig holds application-level configuration.
type AppConfig struct {
	BatchSize       int           `koanf:"batch_size"`
	BatchTimeout    time.Duration `koanf:"batch_timeout"`
	MonitorInterval time.Duration `koanf:"monitor_interval"`
}

// LogConfig holds logging configuration.
type LogConfig struct {
	Level  string `koanf:"level"`
	Format string `koanf:"format"` // json or console

	Output []string `koanf:"output"` // stdout, stderr, or file path
	File   string   `koanf:"file"`   // Log file path (if output includes file)

	MaxSize    int  `koanf:"max_size"`    // Max size in MB before rotation
	MaxBackups int  `koanf:"max_backups"` // Max number of old log files to keep
	MaxAge     int  `koanf:"max_age"`     // Max days to retain old log files
	Compress   bool `koanf:"compress"`    // Compress rotated log files

	DisableCaller     bool               `koanf:"disable_caller"`
	DisableStacktrace bool               `koanf:"disable_stacktrace"`
	Development       bool               `koanf:"development"`
	Sampling          *LogSamplingConfig `koanf:"sampling"`
}

// LogSamplingConfig configures log sampling to reduce high-volume logs.
type LogSamplingConfig struct {
	Initial    int           `koanf:"initial"`
	Thereafter int           `koanf:"thereafter"`
	Tick       time.Duration `koanf:"tick"`
}

// PublisherConfig holds publisher configuration.
type PublisherConfig struct {
	Topic string `koanf:"topic"`
}

// TelemetryConfig controls tracing/metrics exporters.
type TelemetryConfig struct {
	Enabled  bool   `koanf:"enabled"`
	Endpoint string `koanf:"endpoint"`
	Insecure bool   `koanf:"insecure"`
}

// DLQConfig defines dead-letter queue routing for permanent failures.
type DLQConfig struct {
	Enabled bool   `koanf:"enabled"`
	Subject string `koanf:"subject"`
}

// AFTNConfig defines AFTN protocol validation and monitoring settings.
type AFTNConfig struct {
	ValidationEnabled          bool          `koanf:"validation_enabled"`
	MessageGapThreshold        time.Duration `koanf:"message_gap_threshold"`
	EnableSequenceGapDetection bool          `koanf:"enable_sequence_gap_detection"`
}

// MonitoringConfig controls the lightweight HTTP server for health and metrics.
type MonitoringConfig struct {
	Disabled      bool          `koanf:"disabled"`
	Addr          string        `koanf:"addr"`
	EnableMetrics bool          `koanf:"enable_metrics"`
	EnableHealth  bool          `koanf:"enable_health"`
	ReadTimeout   time.Duration `koanf:"read_timeout"`
	WriteTimeout  time.Duration `koanf:"write_timeout"`
	HealthTimeout time.Duration `koanf:"health_timeout"`
}

// SubscriptionConfig holds subscription configuration.
type SubscriptionConfig struct {
	Topic      string `koanf:"topic"`
	QueueGroup string `koanf:"queue_group"`
}

// TimeoutsConfig holds timeout configuration.
type TimeoutsConfig struct {
	Server        time.Duration `koanf:"server"`
	ReconnectWait time.Duration `koanf:"reconnect_wait"`
	Close         time.Duration `koanf:"close"`
	AckWait       time.Duration `koanf:"ack_wait"`
}

var (
	nestedEnvPrefixes = []struct {
		envPrefix string
		koanfPath string
	}{
		{"nats_stream_limits_", "nats.stream_limits."},
		{"nats_consumer_rules_", "nats.consumer_rules."},
		{"nats_auth_", "nats.auth."},
		{"log_sampling_", "log.sampling."},
	}

	topEnvPrefixes = []string{
		"nats_",
		"postgres_",
		"app_",
		"log_",
		"publisher_",
		"telemetry_",
		"monitoring_",
		"dlq_",
		"aftn_",
		"subscription_",
		"timeouts_",
	}
)

// mapEnvKey converts CAATSM_* environment variable names to koanf dotted paths
// while preserving underscores within snake_case field names.
//
// Supported forms:
//   - Double underscore nesting: CAATSM_MONITORING__ENABLE_METRICS -> monitoring.enable_metrics
//   - Known section prefix:    CAATSM_MONITORING_ENABLE_METRICS  -> monitoring.enable_metrics
func mapEnvKey(s string) string {
	s = strings.ToLower(strings.TrimPrefix(s, "CAATSM_"))
	if strings.Contains(s, "__") {
		return strings.ReplaceAll(s, "__", ".")
	}
	for _, p := range nestedEnvPrefixes {
		if strings.HasPrefix(s, p.envPrefix) {
			return p.koanfPath + strings.TrimPrefix(s, p.envPrefix)
		}
	}
	for _, prefix := range topEnvPrefixes {
		if strings.HasPrefix(s, prefix) {
			section := strings.TrimSuffix(prefix, "_")
			return section + "." + strings.TrimPrefix(s, prefix)
		}
	}
	return strings.ReplaceAll(s, "_", ".")
}

func hasCAATSMEnvVars() bool {
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "CAATSM_") {
			return true
		}
	}
	return false
}

// LoadConfig loads configuration from file (if present) and environment variables.
func LoadConfig() (*Config, error) {
	k := koanf.New(".")

	env := os.Getenv("GO_ENV")
	if env == "" {
		env = "dev"
	}

	explicitFile := os.Getenv("CAATSM_CONFIG_FILE")
	configFile := explicitFile
	if configFile == "" {
		configFile = fmt.Sprintf("configs/config.%s.toml", env)
	}

	if err := k.Load(file.Provider(configFile), toml.Parser()); err != nil {
		// Allow 12-factor pure-environment startup when default TOML file is absent and CAATSM_* vars exist.
		if explicitFile != "" || !errors.Is(err, os.ErrNotExist) || !hasCAATSMEnvVars() {
			return nil, fmt.Errorf("error loading config file '%s': %w", configFile, err)
		}
	}

	envProvider := envprovider.Provider("CAATSM_", ".", mapEnvKey)
	if err := k.Load(envProvider, nil); err != nil {
		return nil, fmt.Errorf("error loading environment variables: %w", err)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	applyDefaults(&cfg)

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

func applyDefaults(cfg *Config) {
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
	if len(cfg.Log.Output) == 0 {
		cfg.Log.Output = []string{"stdout"}
	} else {
		hasFile := false
		for _, out := range cfg.Log.Output {
			if out == "file" {
				hasFile = true
				break
			}
		}
		if hasFile && cfg.Log.File == "" {
			cfg.Log.File = "logs/caatsm.log"
		}
	}
	if cfg.Log.MaxSize == 0 {
		cfg.Log.MaxSize = 100
	}
	if cfg.Log.MaxBackups == 0 {
		cfg.Log.MaxBackups = 7
	}
	if cfg.Log.MaxAge == 0 {
		cfg.Log.MaxAge = 30
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
	if cfg.Timeouts.ReconnectWait == 0 {
		cfg.Timeouts.ReconnectWait = 5 * time.Second
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
	if cfg.AFTN.MessageGapThreshold == 0 {
		cfg.AFTN.MessageGapThreshold = 2 * time.Minute
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.NATS.URL == "" {
		return fmt.Errorf("nats.url is required")
	}
	mode := strings.ToLower(c.NATS.Mode)
	if mode != "" && mode != "jetstream" {
		return fmt.Errorf("nats.mode must be 'jetstream' or empty (defaults to 'jetstream'), got: %s", c.NATS.Mode)
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
	if c.AFTN.MessageGapThreshold < 0 {
		return fmt.Errorf("aftn.message_gap_threshold must be >= 0")
	}
	return nil
}

// EffectiveSubscriptionTopic returns the active subscription subject.
func (c *Config) EffectiveSubscriptionTopic() string {
	if c == nil {
		return ""
	}
	if topic := c.Subscription.Topic; topic != "" {
		return topic
	}
	return "telegram.>"
}

// EffectivePublishStream returns the dedicated stream name for outbound parsed telegrams.
func (c *Config) EffectivePublishStream() string {
	if c == nil {
		return "TELEGRAM_OUT"
	}
	if c.NATS.PublishStream != "" {
		return c.NATS.PublishStream
	}
	base := c.NATS.Stream
	if base == "" {
		base = "TELEGRAM"
	}
	return base + "_OUT"
}

// EffectiveDLQStream returns the dedicated stream name for DLQ messages.
func (c *Config) EffectiveDLQStream() string {
	if c == nil {
		return "TELEGRAM_DLQ"
	}
	if c.NATS.DLQStream != "" {
		return c.NATS.DLQStream
	}
	base := c.NATS.Stream
	if base == "" {
		base = "TELEGRAM"
	}
	return base + "_DLQ"
}

// ShouldAutoProvision reports whether JetStream streams and consumers should be
// auto-created on startup. Defaults to false in production (GO_ENV=prod/production)
// and true in other environments unless overridden by nats.auto_provision.
func (c *Config) ShouldAutoProvision() bool {
	if c != nil && c.NATS.AutoProvision != nil {
		return *c.NATS.AutoProvision
	}
	env := strings.ToLower(os.Getenv("GO_ENV"))
	return env != "prod" && env != "production"
}
