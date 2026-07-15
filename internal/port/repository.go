package port

import (
	"context"
	"errors"

	"caatsm/internal/adapter/dto"
)

// ErrDuplicate is returned by InsertOne when a duplicate business key is detected.
// Callers should treat this as a successful no-op (the message was already persisted).
var ErrDuplicate = errors.New("duplicate message")

// Repository defines the interface for message persistence
type Repository interface {
	// InsertOne inserts a single telegram message.
	// Returns ErrDuplicate when the business key (message_id, date_time) already exists.
	InsertOne(ctx context.Context, msg *dto.ParsedTelegram) error

	// InsertBatch inserts multiple telegram messages in a batch
	InsertBatch(ctx context.Context, msgs []*dto.ParsedTelegram) error

	// InsertRaw captures an unparsed or failed telegram for later analysis.
	InsertRaw(ctx context.Context, msg *dto.ParsedTelegram) error
}

