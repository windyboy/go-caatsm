package log

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	testConfigFileName        = "./configs/logger.test.json"
	prodConfigFileName        = "./configs/logger.json"
	developmentConfigFileName = "./configs/logger.dev.json"

	envTest = "test"
	envProd = "prod"
	envDev  = "dev"
)

type loggerConfig struct {
	ZapConfig        zap.Config       `json:"zapConfig"`
	LumberjackConfig lumberjackConfig `json:"lumberjackConfig"`
}

type lumberjackConfig struct {
	Filename   string `json:"filename"`
	MaxSize    int    `json:"maxSize"`
	MaxBackups int    `json:"maxBackups"`
	MaxAge     int    `json:"maxAge"`
	Compress   bool   `json:"compress"`
}

var (
	sugarOnce sync.Once
	baseOnce  sync.Once
	sugar     *zap.SugaredLogger
	base      *zap.Logger
)

// Sugared returns the shared sugared logger instance.
func Sugared() *zap.SugaredLogger {
	sugarOnce.Do(func() {
		if base == nil {
			base = buildLogger()
		}
		sugar = base.Sugar()
	})
	return sugar
}

// Logger returns the shared structured logger instance.
func Logger() *zap.Logger {
	baseOnce.Do(func() {
		base = buildLogger()
	})
	return base
}

func buildLogger() *zap.Logger {
	env := currentEnv()
	cfg, err := loadConfigFile(selectedConfig(env))
	if err != nil {
		fmt.Printf("logger: using fallback configuration due to error: %v\n", err)
		logger, _ := zap.NewDevelopment()
		return logger
	}

	var sink zapcore.WriteSyncer
	if env == envProd {
		sink = zapcore.AddSync(&lumberjack.Logger{
			Filename:   cfg.LumberjackConfig.Filename,
			MaxSize:    cfg.LumberjackConfig.MaxSize,
			MaxBackups: cfg.LumberjackConfig.MaxBackups,
			MaxAge:     cfg.LumberjackConfig.MaxAge,
			Compress:   cfg.LumberjackConfig.Compress,
		})
	} else {
		sink = zapcore.AddSync(os.Stdout)
	}

	encoder := zapcore.NewJSONEncoder(cfg.ZapConfig.EncoderConfig)
	level := parseLevel(cfg.ZapConfig.Level.String())

	core := zapcore.NewCore(encoder, sink, level)
	return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
}

func currentEnv() string {
	if mode := os.Getenv("TELE_MODE"); mode != "" {
		return mode
	}
	return envDev
}

func selectedConfig(env string) string {
	switch env {
	case envTest:
		return testConfigFileName
	case envProd:
		return prodConfigFileName
	default:
		return developmentConfigFileName
	}
}

func loadConfigFile(path string) (loggerConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return loggerConfig{}, fmt.Errorf("open logger config: %w", err)
	}
	defer file.Close()

	var cfg loggerConfig
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return loggerConfig{}, fmt.Errorf("decode logger config: %w", err)
	}
	return cfg, nil
}

func parseLevel(level string) zapcore.Level {
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
