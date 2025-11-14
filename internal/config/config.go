package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/viper"
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
	// log := utils.Logger
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
		return nil, errors.New(errMsg)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		errMsg := fmt.Sprintf("unable to decode config into struct for environment '%s': %v", env, err)
		// log.Error(errMsg)
		return nil, errors.New(errMsg)
	}
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
	return nil
}
