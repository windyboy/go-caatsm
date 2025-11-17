package log

import (
	"caatsm/internal/infra/config"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// ProvideLogger creates a zap logger based on configuration
func ProvideLogger(cfg *config.Config) (*zap.Logger, error) {
	var zapConfig zap.Config

	if cfg.Log.Development || cfg.Log.Format == "console" {
		zapConfig = zap.NewDevelopmentConfig()
	} else {
		zapConfig = zap.NewProductionConfig()
	}

	// Set log level
	var level zapcore.Level
	switch cfg.Log.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}
	zapConfig.Level = zap.NewAtomicLevelAt(level)

	// Determine outputs
	outputs := cfg.Log.Output
	if len(outputs) == 0 {
		outputs = []string{"stdout"} // Default to stdout
	}

	// Create encoder
	var encoder zapcore.Encoder
	if cfg.Log.Format == "json" {
		encoder = zapcore.NewJSONEncoder(zapConfig.EncoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(zapConfig.EncoderConfig)
	}

	// Build cores for different outputs
	var cores []zapcore.Core

	// Create cores for each output
	for _, output := range outputs {
		var writeSyncer zapcore.WriteSyncer

		switch output {
		case "stdout":
			writeSyncer = zapcore.AddSync(os.Stdout)
		case "stderr":
			writeSyncer = zapcore.AddSync(os.Stderr)
		case "file":
			filePath := cfg.Log.File
			if filePath == "" {
				// Use default path if file path is not specified
				filePath = "logs/caatsm.log"
			}
			// Ensure log directory exists
			dir := filepath.Dir(filePath)
			if dir != "" && dir != "." {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return nil, err
				}
			}

			// Configure log rotation
			lj := &lumberjack.Logger{
				Filename:   filePath,
				MaxSize:    cfg.Log.MaxSize, // MB
				MaxBackups: cfg.Log.MaxBackups,
				MaxAge:     cfg.Log.MaxAge, // days
				Compress:   cfg.Log.Compress,
			}
			writeSyncer = zapcore.AddSync(lj)
		default:
			// Treat as file path
			dir := filepath.Dir(output)
			if dir != "" && dir != "." {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return nil, err
				}
			}
			lj := &lumberjack.Logger{
				Filename:   output,
				MaxSize:    cfg.Log.MaxSize,
				MaxBackups: cfg.Log.MaxBackups,
				MaxAge:     cfg.Log.MaxAge,
				Compress:   cfg.Log.Compress,
			}
			writeSyncer = zapcore.AddSync(lj)
		}

		cores = append(cores, zapcore.NewCore(encoder, writeSyncer, zapConfig.Level))
	}

	// Combine cores
	core := zapcore.NewTee(cores...)

	// Build options
	opts := []zap.Option{}
	if !cfg.Log.DisableCaller {
		opts = append(opts, zap.AddCaller())
	}
	if !cfg.Log.DisableStacktrace {
		opts = append(opts, zap.AddStacktrace(zapcore.ErrorLevel))
	}

	// Add sampling if configured
	if cfg.Log.Sampling != nil && cfg.Log.Sampling.Initial > 0 {
		tick := cfg.Log.Sampling.Tick
		if tick == 0 {
			tick = time.Second // Default to 1 second
		}
		opts = append(opts, zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewSamplerWithOptions(
				core,
				tick,
				cfg.Log.Sampling.Initial,
				cfg.Log.Sampling.Thereafter,
			)
		}))
	}

	// Build logger
	logger := zap.New(core, opts...)

	// Note: We still replace global logger for backward compatibility with parsers package
	// This will be removed once parsers are fully migrated to use dependency injection
	zap.ReplaceGlobals(logger)

	return logger, nil
}
