package utils

import (
	"encoding/json"
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	TestConfigFileName        = "./configs/logger.test.json"
	ProdConfigFileName        = ".configs/logger.json"
	DevelopmentConfigFileName = "./configs/logger.dev.json"
	EnvTest                   = "test"
	EnvProd                   = "prod"
	EnvDev                    = "dev"
)

type LoggerConfig struct {
	ZapConfig        zap.Config       `json:"zapConfig"`
	LumberjackConfig LumberjackConfig `json:"lumberjackConfig"`
}

type LumberjackConfig struct {
	Filename   string `json:"filename"`
	MaxSize    int    `json:"maxSize"`
	MaxBackups int    `json:"maxBackups"`
	MaxAge     int    `json:"maxAge"`
	Compress   bool   `json:"compress"`
}

var (
	sugar *zap.SugaredLogger
	log   *zap.Logger
)

func load() {
	if log == nil {
		env := getEnv()
		// fmt.Printf("Environment: %s\n", env)
		configFile := getConfigFile(env)

		config, err := loadConfig(configFile)
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			log, _ = zap.NewDevelopment()
			sugar = log.Sugar()
			return
		}

		var logWriter zapcore.WriteSyncer
		if env == EnvProd {
			logWriter = zapcore.AddSync(&lumberjack.Logger{
				Filename:   config.LumberjackConfig.Filename,
				MaxSize:    config.LumberjackConfig.MaxSize,
				MaxBackups: config.LumberjackConfig.MaxBackups,
				MaxAge:     config.LumberjackConfig.MaxAge,
				Compress:   config.LumberjackConfig.Compress,
			})
		} else {
			logWriter = zapcore.AddSync(os.Stdout)
		}

		encoder := zapcore.NewJSONEncoder(config.ZapConfig.EncoderConfig)
		level := parseLogLevel(config.ZapConfig.Level.String())

		core := zapcore.NewCore(
			encoder,
			logWriter,
			level,
		)

		log = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
		sugar = log.Sugar()
	}
}

func GetSugaredLogger() *zap.SugaredLogger {
	load()
	return log.Sugar()
}

func getEnv() string {
	env := os.Getenv("TELE_MODE")
	if env == "" {
		env = EnvDev
	}
	return env
}

func getConfigFile(env string) string {
	switch env {
	case EnvTest:
		return TestConfigFileName
	case EnvProd:
		return ProdConfigFileName
	default:
		return DevelopmentConfigFileName
	}
}

func loadConfig(configFile string) (LoggerConfig, error) {
	file, err := os.Open(configFile)
	if err != nil {
		return LoggerConfig{}, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	var config LoggerConfig
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return LoggerConfig{}, fmt.Errorf("error decoding config: %v", err)
	}
	return config, nil
}

func GetLogger() *zap.SugaredLogger {
	if sugar == nil {
		load()
	}
	return sugar
}

func parseLogLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "dpanic":
		return zapcore.DPanicLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}
