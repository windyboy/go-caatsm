package adapter

import (
	"context"
	"caatsm/internal/domain"
)

// Repository defines the interface for message persistence
type Repository interface {
	// InsertOne inserts a single telegram message
	InsertOne(ctx context.Context, msg *domain.ParsedMessage) error
	
	// InsertBatch inserts multiple telegram messages in a batch
	InsertBatch(ctx context.Context, msgs []*domain.ParsedMessage) error

	// InsertRaw captures an unparsed or failed telegram for later analysis.
	InsertRaw(ctx context.Context, msg *domain.ParsedMessage) error
}

