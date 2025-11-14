package app

import (
	"caatsm/internal/adapter"
	"caatsm/internal/adapter/parser"
	"context"
	"fmt"
	"go.uber.org/zap"
)

// MessageProcessor handles message processing
type MessageProcessor struct {
	parser     parser.Parser
	repository adapter.Repository
	publisher  adapter.Publisher
	logger     *zap.Logger
}

// NewMessageProcessor creates a new message processor
func NewMessageProcessor(
	parser parser.Parser,
	repository adapter.Repository,
	publisher adapter.Publisher,
	logger *zap.Logger,
) *MessageProcessor {
	return &MessageProcessor{
		parser:     parser,
		repository: repository,
		publisher:  publisher,
		logger:     logger,
	}
}

// Handle processes a message
func (p *MessageProcessor) Handle(ctx context.Context, raw []byte, msgID string) error {
	if raw == nil || len(raw) == 0 {
		return Permanent(fmt.Errorf("empty message"))
	}

	// Parse the message
	parsed := p.parser.Parse(string(raw))
	if parsed == nil {
		return Permanent(fmt.Errorf("parser returned nil"))
	}

	// Set the message ID from NATS
	parsed.Uuid = msgID

	// Log parsing result
	if !parsed.Parsed {
		p.logger.Info("Message not parsed",
			zap.String("msg_id", msgID),
			zap.String("content", parsed.Content),
		)
	} else {
		p.logger.Info("Message parsed successfully",
			zap.String("msg_id", msgID),
			zap.String("message_id", parsed.MessageID),
			zap.String("category", parsed.Category),
		)
	}

	// Insert into database
	if err := p.repository.InsertOne(ctx, parsed); err != nil {
		return fmt.Errorf("failed to insert message: %w", err)
	}

	// Publish parsed message
	if err := p.publisher.Publish(parsed); err != nil {
		// Log error but don't fail the entire operation
		p.logger.Error("Failed to publish message",
			zap.String("msg_id", msgID),
			zap.Error(err),
		)
		// Return error to trigger NAK and retry
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}
