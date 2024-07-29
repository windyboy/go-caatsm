package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/viper"
)

var MyConfig *Config

type Config struct {
	Nats         NatsConfig
	Subscription SubscriptionConfig
	Timeouts     TimeoutsConfig
	Hasura       HasuraConfig
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

type HasuraConfig struct {
	Endpoint string
	Secret   string
}

const (
	EnvProd = "prod"
	// logConfigFile = "configs/log_config.json"
	// arrPatternString represents the regular expression pattern used to match arrival patterns.
	// The pattern matches strings in the format: "(TYPE-NUMBER-SSR-DEPARTURE-ARRIVAL)".
	// The pattern captures the following named groups:
	// - category: the three-letter category code
	// - number: the alphanumeric flight number
	// - ssr: the alphanumeric SSR code
	// - departure: the four-letter departure airport code
	// - arrival: the four-letter arrival airport code
	arrPatternString = `^\((?P<category>[A-Z]{3})` +
		`-(?P<number>[A-Z0-9]+)(\/?(?P<ssr>[A-Z0-9]+))?` +
		`-(?P<dep>[A-Z]{4})` +
		`-(?P<arr>[A-Z]{4})(?P<arr_time>\d{4})\)$`

	// depPatternString represents the regular expression pattern used to match departure patterns.
	// The pattern matches strings in the format: "(TYPE-NUMBER-SSR-DEPARTURE-DEPARTURE_TIME-ARRIVAL)".
	// The pattern captures the following named groups:
	// - category: the three-letter category code
	// - number: the alphanumeric flight number
	// - ssr: the alphanumeric SSR code
	// - departure: the four-letter departure airport code
	// - departure_time: the four-digit departure time
	// - arrival: the four-letter arrival airport code
	depPatternString = `^\((?P<category>[A-Z]{3})-(?P<number>[A-Z0-9]+)(\/(?P<ssr>[A-Z0-9]+))?-(?P<dep>[A-Z]{4})(?P<dep_time>\d{4})-(?P<arr>[A-Z]{4})\)$`

	// fplPatternString represents the regular expression pattern used to match flight plan patterns.
	// The pattern matches strings in the format: "(CATEGORY-NUMBER-INDICATOR-AIRCRAFT-SURVE-DEPARTURE-DEPARTURE_TIME-SPEED-LEVEL-ROUTE-DESTINATION-ESTT-ALTER-OTHER)".
	// The pattern captures the following named groups:
	// - category: the three-letter category code
	// - number: the alphanumeric flight number
	// - indicator: the two-letter indicator
	// - aircraft: the alphanumeric aircraft type, optionally followed by a slash and an uppercase letter
	// - surve: any character sequence (surveillance information)
	// - departure: the four-letter departure airport code
	// - departure_time: the four-digit departure time
	// - speed: one or more uppercase letters followed by one or more digits (e.g., N123)
	// - level: one or more alphanumeric characters (e.g., FL350)
	// - route: one or more characters (including newline) representing the flight route
	// - destination: the four-letter destination airport code
	// - estt: the four-digit estimated time of arrival
	// - alter: one or more sequences of a whitespace character followed by exactly four uppercase letters
	// - other: any other relevant information, starting with three uppercase letters followed by a forward slash and zero or more characters (including newline)

	fplPatternString = `\((?P<category>[A-Z]{3})-(?P<number>[A-Z]+\d+)-(?P<indicator>[A-Z]{2})\n-(?P<aircraft>[A-Z]+\d+\/?[A-Z]?)\n?-(?P<surve>.*)\n?-(?P<dep>[A-Z]{4})(?P<dep_time>\d{4})\n?-(?P<speed>[A-Z]+\d+)(?P<level>[A-Z0-9]+)\s+(?P<route>(.|\n)+)\n-(?P<dest>[A-Z]{4})(?P<estt>\d{4})\s?(?P<alter>(\s[A-Z]{4})+)\n?-([A-Z]{3}\/(?:[A-Z]{4}\d{4}\s?)+)?(?P<other>(?m)[A-Z]{3}\/(.|\n)*)\)$`

	cnlPatternString = `^\((?P<category>[A-Z]{3})-(?P<number>\w+\d+)-?(?P<dep>[A-Z]{4})?-?(?<arr>[A-Z]{4})\)$`

	dlaPatternString = `^\((?P<category>[A-Z]{3})-(?P<number>\w+\d+)-?(?P<dep>[A-Z]{4})(?P<dep_time>\d{4})?-?(?<arr>[A-Z]{4})(?<arr_time>\d{4})?\)$`
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
	"CNL": {
		Patterns: []PatternConfig{
			{
				Pattern:    cnlPatternString,
				Comments:   "Pattern for CNL message",
				Expression: regexp.MustCompile(cnlPatternString),
			},
		},
	},
	"DLA": {
		Patterns: []PatternConfig{
			{
				Pattern:    dlaPatternString,
				Comments:   "Pattern for DLA message",
				Expression: regexp.MustCompile(dlaPatternString),
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
	// loadLoggerConfig()

	// log.Debugf("Config loaded: %+v", config)
	// loadLoggerConfig()
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
	// fmt.Println("config validation passed")
	return nil
}
