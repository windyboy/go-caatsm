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
	ServerTimeout  time.Duration `mapstructure:"server_timeout"`
	ReconnectWait  time.Duration `mapstructure:"reconnect_wait"`
	CloseTimeout   time.Duration `mapstructure:"close_timeout"`
	AckWaitTimeout time.Duration `mapstructure:"ack_wait_timeout"`
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
	// arrPatternString represents the regular expression pattern used to match arrival patterns.
	// The pattern matches strings in the format: "(TYPE-NUMBER-SSR-DEPARTURE-ARRIVAL)".
	// The pattern captures the following named groups:
	// - category: the three-letter category code
	// - number: the alphanumeric flight number
	// - ssr: the alphanumeric SSR code
	// - departure: the four-letter departure airport code
	// - arrival: the four-letter arrival airport code
	arrPatternString = `^\((?P<category>[A-Z]{3})\-(?P<number>[A-Z0-9]+)(\-(?P<ssr>[A-Z]+[0-9]+))?\-(?P<departure>[A-Z]{4})\-(?P<arrival>[A-Z]{4})(?P<time>\d{4})\)$`

	// depPatternString represents the regular expression pattern used to match departure patterns.
	// The pattern matches strings in the format: "(TYPE-NUMBER-SSR-DEPARTURE-DEPARTURE_TIME-ARRIVAL)".
	// The pattern captures the following named groups:
	// - category: the three-letter category code
	// - number: the alphanumeric flight number
	// - ssr: the alphanumeric SSR code
	// - departure: the four-letter departure airport code
	// - departure_time: the four-digit departure time
	// - arrival: the four-letter arrival airport code
	depPatternString = `^\((?P<category>[A-Z]{3})-(?P<number>[A-Z0-9]+)-(?P<ssr>[A-Z0-9]+)-(?P<departure>[A-Z]{4})-(?P<departure_time>\d{4})-(?P<arrival>[A-Z]{4})\)$`

	// fplPatternString is a regular expression designed to parse and extract detailed information from formatted flight plan strings.
	// The flight plan string is expected to follow a specific format, encapsulated by parentheses and containing various segments separated by hyphens.
	// Each segment captures a different aspect of the flight plan, as detailed below:
	//
	// - category: Matches a three-letter flight category code.
	// - number: Matches the flight number, consisting of letters followed by digits.
	// - indicator: Matches a two-letter indicator.
	// - aircraft: Matches the aircraft registration or type, which may include a slash and an optional letter.
	// - surve: Matches surveillance equipment information, capturing any characters in this segment.
	// - departure: Matches the four-letter code of the departure airport.
	// - departure_time: Matches the four-digit departure time.
	// - speed: Matches the speed, consisting of letters followed by digits.
	// - level: Matches the flight level, which can include letters and digits.
	// - route: Matches the flight route, capturing any characters in this segment.
	// - destination: Matches the four-letter code of the destination airport.
	// - estt: Matches the four-digit estimated time of arrival.
	// - alter: Matches the four-letter code of an alternate airport.
	// - pbn: Matches performance-based navigation equipment information.
	// - nav: Matches navigation equipment information.
	// - reg: Matches the aircraft registration.
	// - eet: Matches the estimated elapsed time, including the four-letter code and four-digit time.
	// - sel: Matches the SELCAL code.
	// - performance: Matches a single letter indicating the aircraft's performance category.
	// - rif: Matches reroute information, including any characters in this segment.
	// - remark: Matches remarks, capturing any characters in this segment.
	//
	// The regular expression uses named capture groups for each segment, allowing for easy extraction of specific information from a matched flight plan string.
	// fplPatternString = `\((?P<category>[A-Z]{3})\-(?P<number>[A-Z]+\d+)\-(?P<indicator>[A-Z]{2})(.*\n)?(.*\n)?\-(?P<aircraft>[A-Z]+\d+\/?[A-Z]?)\s*\-(?P<surve>.*)(.*\n)?\-(?P<departure>[A-Z]{4})(?P<departure_time>\d{4})(.*\n)?\-(?P<speed>[A-Z]+\d+)(?P<level>[A-Z0-9]+)\s(?P<route>.*)(.*\n)?\-(?P<destination>[A-Z]{4})(?P<estt>\d{4})\s(?P<alter>[A-Z]{4})(.*\n)?\-(?P<pbn>PBN\/[A-Z0-9]+)\s(?P<nav>NAV\/\w+)\sREG\/(?P<reg>[A-Z0-9]+)\sEET\/(?P<eet>\w{4}\d{4})\sSEL\/(?P<sel>\w+)\sPER\/(?P<performance>\w)\sRIF\/(?P<rif>\w+\s[A-Z0-9]+\s[A-Z]+)\s*RMK\/(?P<remark>.*)\)$`
	fplPatternString = `\((?P<category>[A-Z]{3})\-(?P<number>[A-Z]+\d+)\-(?P<indicator>[A-Z]{2})(.*\n)?(.*\n)?\-(?P<aircraft>[A-Z]+\d+\/?[A-Z]?)\s*\-(?P<surve>.*)(.*\n)?\-(?P<departure>[A-Z]{4})(?P<departure_time>\d{4})(.*\n)?\-(?P<speed>[A-Z]+\d+)(?P<level>[A-Z0-9]+)\s(?P<route>.*)(.*\n)?\-(?P<destination>[A-Z]{4})(?P<estt>\d{4})\s(?P<alter>[A-Z]{4})(.*\n)?\-(?P<pbn>PBN\/[A-Z0-9]+)\s(?P<nav>NAV\/\w+)\sREG\/(?P<reg>[A-Z0-9]+)\sEET\/(?P<eet>\w{4}\d{4})\sSEL\/(?P<sel>\w+)\sPER\/(?P<performance>\w)\sRIF\/(?P<rif>\w+\s[A-Z0-9]+\s[A-Z]+)\s*RMK\/(?P<remark>.*)\)$`
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
	// fmt.Println("config validation passed")
	return nil
}
