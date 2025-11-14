package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/toml"
	envprovider "github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Deprecated: Use explicit config passing instead. Will be removed in v2.0
var MyConfig *Config

type Config struct {
	Nats         NatsConfig
	Subscription SubscriptionConfig
	Publisher    PublisherConfig
	Database     DatabaseConfig
	Redis        RedisConfig
	API          APIConfig
	Timeouts     TimeoutsConfig
}

type NatsConfig struct {
	Client    string
	URL       string
	Cluster   string
	JetStream JetStreamConfig
}

type JetStreamConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	StreamName    string `mapstructure:"stream_name"`
	Subject       string `mapstructure:"subject"`
	MaxPending    int    `mapstructure:"max_pending"`
	AutoProvision bool   `mapstructure:"auto_provision"`
}

type SubscriptionConfig struct {
	Topic      string `mapstructure:"topic"`
	QueueGroup string `mapstructure:"queue_group"`
}

type PublisherConfig struct {
	Topic string `mapstructure:"topic"`
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxConns        int           `mapstructure:"max_conns"`
	MinConns        int           `mapstructure:"min_conns"`
	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	Enabled  bool   `mapstructure:"enabled"`
}

type APIConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"` // dev, prod
}

type TimeoutsConfig struct {
	Server        time.Duration `mapstructure:"server"`
	ReconnectWait time.Duration `mapstructure:"reconnect_wait"`
	Close         time.Duration `mapstructure:"close"`
	AckWait       time.Duration `mapstructure:"ack_wait"`
}

type BodyConfig struct {
	Patterns []PatternConfig
}

type PatternConfig struct {
	Pattern    string
	Comments   string
	Expression *regexp.Regexp
}

const (
	EnvProd = "prod"
	EnvDev  = "dev"
	EnvTest = "test"
)

var (
	envPrefixes           = []string{"TELE_", "CAATSM_"}
	underscoreLiteralKeys = []string{
		"stream_name",
		"max_pending",
		"auto_provision",
		"queue_group",
		"ssl_mode",
		"max_conns",
		"min_conns",
		"max_conn_lifetime",
		"reconnect_wait",
		"ack_wait",
	}
	underscorePlaceholder = "-koanf-underscore-"
)

// Deprecated: Use explicit config passing instead. Will be removed in v2.0
func SetMyConfig(cfg *Config) {
	MyConfig = cfg
}

// Deprecated: Use explicit config passing instead. Will be removed in v2.0
func GetMyConfig() *Config {
	if MyConfig == nil {
		cfg, err := LoadConfig()
		if err != nil {
			fmt.Printf("error loading config: %v", err)
		}
		MyConfig = cfg
	}
	return MyConfig
}

// LoadConfig loads the configuration from a file
func LoadConfig() (*Config, error) {
	env := os.Getenv("GO_ENV")
	if env == "" {
		env = EnvDev
	}

	k := koanf.New(".")
	configFile := os.Getenv("CAATSM_CONFIG_FILE")
	if configFile == "" {
		configDir := os.Getenv("CAATSM_CONFIG_DIR")
		if configDir == "" {
			configDir = "configs"
		}
		configFile = filepath.Join(configDir, fmt.Sprintf("config.%s.toml", env))
	}

	if err := k.Load(file.Provider(configFile), toml.Parser()); err != nil {
		return nil, fmt.Errorf("error reading config file for environment '%s': %w", env, err)
	}

	for _, prefix := range envPrefixes {
		if err := k.Load(envprovider.Provider(prefix, ".", envKeyMap(prefix)), nil); err != nil {
			return nil, fmt.Errorf("error loading environment overrides for prefix '%s': %w", prefix, err)
		}
	}

	var config Config
	if err := k.UnmarshalWithConf("", &config, koanf.UnmarshalConf{Tag: "mapstructure"}); err != nil {
		return nil, fmt.Errorf("unable to decode config into struct for environment '%s': %w", env, err)
	}

	applyDefaults(&config)
	return &config, nil
}

// ValidateConfig validates the loaded configuration
func ValidateConfig(cfg *Config) error {
	if cfg.Nats.Client == "" {
		return fmt.Errorf("nats client is required")
	}
	if cfg.Nats.URL == "" {
		return fmt.Errorf("nats URL is required")
	}
	if cfg.Subscription.Topic == "" {
		return fmt.Errorf("subscription topic is required")
	}
	if cfg.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if cfg.Database.Database == "" {
		return fmt.Errorf("database name is required")
	}
	if cfg.API.Port == 0 {
		return fmt.Errorf("api port is required")
	}
	if cfg.Timeouts.Server <= 0 {
		return fmt.Errorf("timeouts server must be greater than 0")
	}
	if cfg.Timeouts.ReconnectWait <= 0 {
		return fmt.Errorf("timeouts reconnect_wait must be greater than 0")
	}
	if cfg.Timeouts.Close <= 0 {
		return fmt.Errorf("timeouts close must be greater than 0")
	}
	if cfg.Timeouts.AckWait <= 0 {
		return fmt.Errorf("timeouts ack_wait must be greater than 0")
	}
	return nil
}

func applyDefaults(cfg *Config) {
	if cfg.Timeouts.Server <= 0 {
		cfg.Timeouts.Server = 5 * time.Second
	}
	if cfg.Timeouts.ReconnectWait <= 0 {
		cfg.Timeouts.ReconnectWait = 2 * time.Second
	}
	if cfg.Timeouts.Close <= 0 {
		cfg.Timeouts.Close = 5 * time.Second
	}
	if cfg.Timeouts.AckWait <= 0 {
		cfg.Timeouts.AckWait = 30 * time.Second
	}
}

func envKeyMap(prefix string) func(string) string {
	return func(s string) string {
		key := strings.TrimPrefix(s, prefix)
		key = strings.ToLower(key)
		if key == "" {
			return ""
		}

		placeholderKey := key
		for _, literal := range underscoreLiteralKeys {
			placeholder := strings.ReplaceAll(literal, "_", underscorePlaceholder)
			placeholderKey = strings.ReplaceAll(placeholderKey, literal, placeholder)
		}

		placeholderKey = strings.ReplaceAll(placeholderKey, "_", ".")
		placeholderKey = strings.ReplaceAll(placeholderKey, underscorePlaceholder, "_")

		return placeholderKey
	}
}
