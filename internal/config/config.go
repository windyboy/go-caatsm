package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/viper"
)

var MyConfig *Config

type Config struct {
	Nats         NatsConfig
	Subscription SubscriptionConfig
	Timeouts     TimeoutsConfig
}

type NatsConfig struct {
	Client  string
	URL     string
	Cluster string
}

type SubscriptionConfig struct {
	Topic      string `mapstructure:"topic"`
	QueueGroup string `mapstructure:"queue_group"`
}

type TimeoutsConfig struct {
	ServerTimeout  string `mapstructure:"server_timeout"`
	ReconnectWait  string `mapstructure:"reconnect_wait"`
	CloseTimeout   string `mapstructure:"close_timeout"`
	AckWaitTimeout string `mapstructure:"ack_wait_timeout"`
}

type BodyConfig struct {
	Patterns []PatternConfig
}

type PatternConfig struct {
	Pattern    string
	Comments   string
	Expression *regexp.Regexp
}

// Define the regex patterns as constants
const (
	arrPatternString = `^\((?P<type>[A-Z]{3})-(?P<number>[A-Z0-9]+)-(?P<ssr>[A-Z0-9]+)-(?P<departure>[A-Z]{4})-(?P<arrival>[A-Z]{4})\)$`
	depPatternString = `^\((?P<type>[A-Z]{3})-(?P<number>[A-Z0-9]+)-(?P<ssr>[A-Z0-9]+)-(?P<departure>[A-Z]{4})-(?P<departure_time>\d{4})-(?P<arrival>[A-Z]{4})\)$`
	fplPatternString = `^\((?P<type>[A-Z]{3})\-(?P<number>[A-Z]+\d+)\-(?P<indicator>[A-Z]{2})(?:.*\s*)?\-(?P<aircraft>[A-Z]+\d+\/?[A-Z]?)\s*\-(?P<surve>.*)\s*\-(?P<departure>[A-Z]{4})(?P<departure_time>\d{4})\s*\-(?P<speed>[A-Z]+\d+)(?P<level>[A-Z0-9]+)\s(?P<route>.*)\s*\-(?P<destination>[A-Z]{4})(?P<estt>\d{4})\s(?P<alter>[A-Z]{4})\s*\-(?P<pbn>PBN\/[A-Z0-9]+)\s(?P<nav>NAV\/\w+)\sREG\/(?P<reg>[A-Z0-9]+)\sEET\/(?P<eet>\w{4}\d{4})\sSEL\/(?P<sel>\w+)\sPER\/(?P<performance>\w)\sRIF\/(?P<rif>\w+\s[A-Z0-9]+\s[A-Z]+)\s*RMK\/(?P<remark>.*)\)$`
)

// Initialize the bodyPatterns map
var bodyPatterns = map[string]BodyConfig{
	"ARR": {
		Patterns: []PatternConfig{
			{
				Pattern:    arrPatternString,
				Comments:   "Pattern for ARR message",
				Expression: regexp.MustCompile(arrPatternString),
			},
		},
	},
	"DEP": {
		Patterns: []PatternConfig{
			{
				Pattern:    depPatternString,
				Comments:   "Pattern for DEP message",
				Expression: regexp.MustCompile(depPatternString),
			},
		},
	},
	"FPL": {
		Patterns: []PatternConfig{
			{
				Pattern:    fplPatternString,
				Comments:   "Pattern for FPL message",
				Expression: regexp.MustCompile(fplPatternString),
			},
		},
	},
}

func GetBodyPatterns() map[string]BodyConfig {
	return bodyPatterns
}

func SetMyConfig(cfg *Config) {
	MyConfig = cfg
}

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
		return nil, fmt.Errorf(errMsg)
	}

	// log.Debug("Config file read successfully")
	// log.Debugf("Config keys: %v", viper.AllKeys())

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		errMsg := fmt.Sprintf("unable to decode config into struct for environment '%s': %v", env, err)
		// log.Error(errMsg)
		return nil, fmt.Errorf(errMsg)
	}

	// log.Debugf("Config loaded: %+v", config)
	return &config, nil
}

// ValidateConfig validates the loaded configuration
func ValidateConfig(cfg *Config) error {
	// log := utils.Logger

	if cfg.Nats.Client == "" {
		return fmt.Errorf("nats client is required")
	}
	if cfg.Nats.URL == "" {
		return fmt.Errorf("nats URL is required")
	}
	if cfg.Subscription.Topic == "" {
		return fmt.Errorf("subscription topic is required")
	}
	fmt.Println("config validation passed")
	return nil
}

// func logAndReturnError(log *logrus.Logger, msg string) error {
// 	log.Error(msg)
// 	return fmt.Errorf(msg)
// }
