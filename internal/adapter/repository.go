package adapter

import (
	"context"
	"caatsm/internal/model"
)

// Repository defines the interface for message persistence
type Repository interface {
	// InsertOne inserts a single telegram message
	InsertOne(ctx context.Context, msg *model.ParsedTelegram) error
	
	// InsertBatch inserts multiple telegram messages in a batch
	InsertBatch(ctx context.Context, msgs []*model.ParsedTelegram) error

	// InsertRaw captures an unparsed or failed telegram for later analysis.
	InsertRaw(ctx context.Context, msg *model.ParsedTelegram) error
}

