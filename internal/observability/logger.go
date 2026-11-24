package observability

import (
	"caatsm/internal/config"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// NewLogger creates a Zap logger from configuration without mutating global logger state.
func NewLogger(cfg *config.Config) (*zap.Logger, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config must not be nil")
	}

	format := strings.ToLower(strings.TrimSpace(cfg.Log.Format))
	levelStr := strings.ToLower(strings.TrimSpace(cfg.Log.Level))

	var zapConfig zap.Config
	if cfg.Log.Development || format == "console" {
		zapConfig = zap.NewDevelopmentConfig()
	} else {
		zapConfig = zap.NewProductionConfig()
	}

	var level zapcore.Level
	switch levelStr {
	case "debug":
		level = zapcore.DebugLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}
	zapConfig.Level = zap.NewAtomicLevelAt(level)

	outputs := cfg.Log.Output
	if len(outputs) == 0 {
		outputs = []string{"stdout"}
	}

	var encoder zapcore.Encoder
	if format == "console" {
		encoder = zapcore.NewConsoleEncoder(zapConfig.EncoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(zapConfig.EncoderConfig)
	}

	var cores []zapcore.Core
	for _, output := range outputs {
		var writeSyncer zapcore.WriteSyncer

		switch output {
		case "stdout":
			writeSyncer = zapcore.AddSync(os.Stdout)
		case "stderr":
			writeSyncer = zapcore.AddSync(os.Stderr)
		default:
			filePath := output
			if output == "file" {
				filePath = cfg.Log.File
				if filePath == "" {
					filePath = "logs/caatsm.log"
				}
			}
			dir := filepath.Dir(filePath)
			if dir != "" && dir != "." {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return nil, err
				}
			}
			lj := &lumberjack.Logger{
				Filename:   filePath,
				MaxSize:    cfg.Log.MaxSize,
				MaxBackups: cfg.Log.MaxBackups,
				MaxAge:     cfg.Log.MaxAge,
				Compress:   cfg.Log.Compress,
			}
			writeSyncer = zapcore.AddSync(lj)
		}

		cores = append(cores, zapcore.NewCore(encoder, writeSyncer, zapConfig.Level))
	}

	core := zapcore.NewTee(cores...)
	var opts []zap.Option
	if !cfg.Log.DisableCaller {
		opts = append(opts, zap.AddCaller())
	}
	if !cfg.Log.DisableStacktrace {
		opts = append(opts, zap.AddStacktrace(zapcore.ErrorLevel))
	}

	if cfg.Log.Sampling != nil && cfg.Log.Sampling.Initial > 0 {
		tick := cfg.Log.Sampling.Tick
		if tick == 0 {
			tick = time.Second
		}
		opts = append(opts, zap.WrapCore(func(c zapcore.Core) zapcore.Core {
			return zapcore.NewSamplerWithOptions(
				c,
				tick,
				cfg.Log.Sampling.Initial,
				cfg.Log.Sampling.Thereafter,
			)
		}))
	}

	return zap.New(core, opts...), nil
}
