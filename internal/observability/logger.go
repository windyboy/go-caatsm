package observability

import (
	"caatsm/internal/config"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// ErrorType represents a coarse-grained categorisation of errors for logging and alerting.
type ErrorType string

const (
	ErrorTypeBusiness  ErrorType = "business"
	ErrorTypeTransient ErrorType = "transient"
	ErrorTypeFatal     ErrorType = "fatal"
)

// Canonical logging field names for structured logs produced by the CAATSM receiver.
const (
	FieldService        = "service"
	FieldTransportMsgID = "transport_msg_id"
	FieldTelegramMsgID  = "telegram_message_id"
	FieldCategory       = "category"
	FieldStream         = "stream"
	FieldConsumer       = "consumer"
	FieldSubject        = "subject"
	FieldNATSSequence   = "nats_sequence"
	FieldRequestID      = "request_id"
	FieldTraceID        = "trace_id"
	FieldErrorType      = "error_type"
)

// MessageFields captures the common structured logging contract for message-processing logs.
type MessageFields struct {
	Service        string
	TransportMsgID string
	BusinessMsgID  string
	Category       string

	Stream     string
	Consumer   string
	Subject    string
	JSSequence uint64

	RequestID string
	TraceID   string
	ErrorType ErrorType
}

// WithMessageContext returns a logger pre-populated with structured fields from MessageFields.
func WithMessageContext(logger *zap.Logger, mf MessageFields) *zap.Logger {
	if logger == nil {
		return zap.NewNop()
	}

	fields := make([]zap.Field, 0, 12)
	if mf.Service != "" {
		fields = append(fields, zap.String(FieldService, mf.Service))
	}
	if mf.TransportMsgID != "" {
		fields = append(fields, zap.String(FieldTransportMsgID, mf.TransportMsgID))
	}
	if mf.BusinessMsgID != "" {
		fields = append(fields, zap.String(FieldTelegramMsgID, mf.BusinessMsgID))
	}
	if mf.Category != "" {
		fields = append(fields, zap.String(FieldCategory, mf.Category))
	}
	if mf.Stream != "" {
		fields = append(fields, zap.String(FieldStream, mf.Stream))
	}
	if mf.Consumer != "" {
		fields = append(fields, zap.String(FieldConsumer, mf.Consumer))
	}
	if mf.Subject != "" {
		fields = append(fields, zap.String(FieldSubject, mf.Subject))
	}
	if mf.JSSequence > 0 {
		fields = append(fields, zap.Uint64(FieldNATSSequence, mf.JSSequence))
	}
	if mf.RequestID != "" {
		fields = append(fields, zap.String(FieldRequestID, mf.RequestID))
	}
	if mf.TraceID != "" {
		fields = append(fields, zap.String(FieldTraceID, mf.TraceID))
	}
	if mf.ErrorType != "" {
		fields = append(fields, zap.String(FieldErrorType, string(mf.ErrorType)))
	}

	return logger.With(fields...)
}

// NewLogger creates a Zap logger from configuration without mutating global logger state.
func NewLogger(cfg *config.Config) (*zap.Logger, error) {
	var zapConfig zap.Config
	if cfg.Log.Development || cfg.Log.Format == "console" {
		zapConfig = zap.NewDevelopmentConfig()
	} else {
		zapConfig = zap.NewProductionConfig()
	}

	var level zapcore.Level
	switch cfg.Log.Level {
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
	if cfg.Log.Format == "json" {
		encoder = zapcore.NewJSONEncoder(zapConfig.EncoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(zapConfig.EncoderConfig)
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
