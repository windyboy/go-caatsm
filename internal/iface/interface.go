package iface

import (
	"caatsm/internal/domain"
	"context"
	"time"
)

// MessageHandler handles incoming messages from NATS
type MessageHandler interface {
	HandleMessage(ctx context.Context, msg []byte, id string) error
}

// MessagePublisher publishes messages to NATS
type MessagePublisher interface {
	Publish(ctx context.Context, subject string, message interface{}) error
	PublishAsync(ctx context.Context, subject string, message interface{}) error
}

// TelegramRepository defines the interface for telegram data access
type TelegramRepository interface {
	// Create inserts a new telegram message
	Create(ctx context.Context, message *domain.ParsedMessage) error
	
	// CreateBatch inserts multiple telegram messages in a single transaction
	CreateBatch(ctx context.Context, messages []*domain.ParsedMessage) error
	
	// GetByID retrieves a telegram message by UUID
	GetByID(ctx context.Context, id string) (*domain.ParsedMessage, error)
	
	// List retrieves telegram messages with filters and pagination
	List(ctx context.Context, filters *TelegramFilters) ([]*domain.ParsedMessage, int64, error)
	
	// GetByTimeRange retrieves messages within a time range
	GetByTimeRange(ctx context.Context, start, end time.Time) ([]*domain.ParsedMessage, error)
	
	// Count returns the total count of messages matching filters
	Count(ctx context.Context, filters *TelegramFilters) (int64, error)
}

// TelegramFilters defines filtering options for telegram queries
type TelegramFilters struct {
	Category          string
	MessageID         string
	PrimaryAddress    string
	PriorityIndicator string
	StartTime         *time.Time
	EndTime           *time.Time
	Limit             int
	Offset            int
	OrderBy           string
	OrderDirection    string // ASC, DESC
}
