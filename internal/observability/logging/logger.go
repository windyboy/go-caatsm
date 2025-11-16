package logging

import "go.uber.org/zap"

// ErrorType represents a coarse-grained categorisation of errors for logging and alerting.
// Typical values:
//   - business: validation failures, domain rule violations, payload issues.
//   - transient: network / DB / NATS glitches that may succeed on retry.
//   - fatal: programming bugs, schema mismatches, or conditions that require operator action.
type ErrorType string

const (
	ErrorTypeBusiness  ErrorType = "business"
	ErrorTypeTransient ErrorType = "transient"
	ErrorTypeFatal     ErrorType = "fatal"
)

// Canonical logging field names for structured logs produced by the CAATSM
// receiver. Using constants avoids scattering magic strings and keeps log
// analysis queries stable over time.
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
// All fields are optional; empty values will simply be skipped.
type MessageFields struct {
	// Identifiers
	Service        string
	TransportMsgID string
	BusinessMsgID  string
	Category       string

	// NATS / JetStream context
	Stream     string
	Consumer   string
	Subject    string
	JSSequence uint64

	// Correlation / tracing
	RequestID string
	TraceID   string

	// Error classification
	ErrorType ErrorType
}

// WithMessageContext returns a logger pre-populated with the structured fields defined in MessageFields.
// This is the primary entry point for enforcing the logging contract in the codebase.
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
