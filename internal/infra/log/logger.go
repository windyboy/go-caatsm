package log

import (
	"caatsm/internal/infra/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ProvideLogger creates a zap logger based on configuration
func ProvideLogger(cfg *config.Config) (*zap.Logger, error) {
	var zapConfig zap.Config
	
	if cfg.Log.Format == "console" {
		zapConfig = zap.NewDevelopmentConfig()
	} else {
		zapConfig = zap.NewProductionConfig()
	}
	
	// Set log level
	switch cfg.Log.Level {
	case "debug":
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case "info":
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case "warn":
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case "error":
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	default:
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}
	
	// Build logger
	logger, err := zapConfig.Build(zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	if err != nil {
		return nil, err
	}
	
	// Note: We still replace global logger for backward compatibility with parsers package
	// This will be removed once parsers are fully migrated to use dependency injection
	zap.ReplaceGlobals(logger)
	
	return logger, nil
}

