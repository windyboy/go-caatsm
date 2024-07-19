package config

import (
	"caatsm/pkg/utils"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/viper"
	// Adjust this import based on your project structure
)

var MyConfig *Config

type Config struct {
	Nats         NatsConfig
	Subscription SubscriptionConfig
	Timeouts     TimeoutsConfig
	Body         []BodyConfig
}

type NatsConfig struct {
	Client  string
	URL     string
	Cluster string
}

type SubscriptionConfig struct {
	Topic      string
	QueueGroup string `mapstructure:"queue_group"`
}

type TimeoutsConfig struct {
	ServerTimeout  string `mapstructure:"server_timeout"`
	ReconnectWait  string `mapstructure:"reconnect_wait"`
	CloseTimeout   string `mapstructure:"close_timeout"`
	AckWaitTimeout string `mapstructure:"ack_wait_timeout"`
}

type BodyConfig struct {
	Name     string
	Patterns []PatternConfig
}

type PatternConfig struct {
	Pattern    string
	Comments   string
	Expression *regexp.Regexp
}

func SetMyConfig(cfg *Config) {
	MyConfig = cfg
}

func GetMyConfig() *Config {
	if MyConfig == nil {
		cfg, err := LoadConfig()
		if err != nil {
			utils.Logger.Fatalf("error loading config: %v", err)
		}
		MyConfig = cfg
	}
	return MyConfig
}

// LoadConfig loads the configuration from a file
func LoadConfig() (*Config, error) {
	log := utils.Logger
	env := os.Getenv("GO_ENV")
	if env == "" {
		env = "dev"
	}
	log.Infof("Environment: %s", env)

	viper.SetConfigType("toml")
	viper.SetConfigName("config." + env)
	viper.AddConfigPath("configs")
	viper.SetEnvPrefix("tele")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		errMsg := fmt.Sprintf("error reading config file for environment '%s': %v", env, err)
		log.Error(errMsg)
		return nil, fmt.Errorf(errMsg)
	}

	log.Debug("Config file read successfully")
	log.Debugf("Config keys: %v", viper.AllKeys())

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		errMsg := fmt.Sprintf("unable to decode config into struct for environment '%s': %v", env, err)
		log.Error(errMsg)
		return nil, fmt.Errorf(errMsg)
	}

	log.Debugf("Config loaded before regex compilation: %+v", config)

	// Compile regex patterns
	for i := range config.Body {
		for j := range config.Body[i].Patterns {
			name := config.Body[i].Name
			pattern := config.Body[i].Patterns[j].Pattern
			expr, err := regexp.Compile(pattern)
			if err != nil {
				errMsg := fmt.Sprintf("error compiling regex for body '%s', pattern '%s': %v", name, pattern, err)
				log.Error(errMsg)
				return nil, fmt.Errorf(errMsg)
			}
			config.Body[i].Patterns[j].Expression = expr
		}
	}

	log.Debugf("Final config after regex compilation: %+v", config)
	return &config, nil
}

// ValidateConfig validates the loaded configuration
func ValidateConfig(cfg *Config) error {
	log := utils.Logger

	if cfg.Nats.Client == "" {
		err := "nats client is required"
		log.Error(err)
		return fmt.Errorf(err)
	}
	if cfg.Nats.URL == "" {
		err := "nats URL is required"
		log.Error(err)
		return fmt.Errorf(err)
	}
	if cfg.Subscription.Topic == "" {
		err := "subscription topic is required"
		log.Error(err)
		return fmt.Errorf(err)
	}
	if len(cfg.Body) == 0 {
		err := "at least one body configuration is required"
		log.Error(err)
		return fmt.Errorf(err)
	}
	for _, body := range cfg.Body {
		if body.Name == "" {
			err := "body name is required"
			log.Error(err)
			return fmt.Errorf(err)
		}
		for _, pattern := range body.Patterns {
			if pattern.Pattern == "" {
				err := fmt.Sprintf("pattern is required for body '%s'", body.Name)
				log.Error(err)
				return fmt.Errorf(err)
			}
			if pattern.Expression == nil {
				err := fmt.Sprintf("compiled expression is missing for pattern '%s' in body '%s'", pattern.Pattern, body.Name)
				log.Error(err)
				return fmt.Errorf(err)
			}
		}
	}
	log.Info("config validation passed")
	return nil
}
