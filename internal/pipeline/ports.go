package pipeline

import (
	"caatsm/internal/telegram"
	"context"
	"time"
)

// Parser parses a raw telegram payload into a Message.
type Parser interface {
	Parse(raw string) (*telegram.Message, error)
}

// Validator validates protocol rules (such as AFTN header constraints) on a parsed Message.
type Validator interface {
	Validate(msg *telegram.Message) error
}

// Store persists parsed and raw/failed telegrams.
type Store interface {
	InsertOne(ctx context.Context, msg *telegram.Message) error
	InsertRaw(ctx context.Context, msg *telegram.Message) error
}

// Publisher publishes processed telegrams downstream.
type Publisher interface {
	Publish(ctx context.Context, msg *telegram.Message) error
}

// Recorder records processing telemetry and metrics.
type Recorder interface {
	RecordProcessingResult(ctx context.Context, status, category string, elapsed time.Duration)
	RecordPublishFailure(ctx context.Context, category string)
	RecordFailure(stage string)
	RecordAFTNValidationError(ctx context.Context, errorType string)
}

// Options configures Processor pipeline behavior.
type Options struct {
	ValidateAFTN bool
}
