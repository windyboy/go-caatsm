package app

import (
	"caatsm/internal/adapter"
	"caatsm/internal/adapter/parser"
	"context"
	"fmt"
	"strings"
	"time"

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
	if len(raw) == 0 {
		return Permanent(fmt.Errorf("empty message"))
	}

	receivedAt := time.Now()

	parsed := p.parser.Parse(string(raw))
	if parsed == nil {
		return Permanent(fmt.Errorf("parser returned nil"))
	}

	if msgID != "" {
		if parsed.Comments == "" {
			parsed.Comments = fmt.Sprintf("nats_msg_id=%s", msgID)
		} else if !strings.Contains(parsed.Comments, "nats_msg_id=") {
			parsed.Comments = fmt.Sprintf("%s; nats_msg_id=%s", parsed.Comments, msgID)
		}
	}
	if parsed.ReceivedAt.IsZero() {
		parsed.ReceivedAt = receivedAt
	}
	if parsed.ParsedAt.IsZero() {
		parsed.ParsedAt = time.Now()
	}

	// Log parsing result
	if !parsed.Parsed {
		p.logger.Warn("Message not parsed",
			zap.String("msg_id", msgID),
			zap.String("message_id", parsed.MessageID),
			zap.String("category", parsed.Category),
			zap.String("content_preview", truncateContent(parsed.Content, 256)),
		)
	} else {
		p.logger.Info("Message parsed successfully",
			zap.String("msg_id", msgID),
			zap.String("message_id", parsed.MessageID),
			zap.String("category", parsed.Category),
			zap.Time("received_at", parsed.ReceivedAt),
			zap.Time("parsed_at", parsed.ParsedAt),
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
		// Mark as permanent so the consumer will ack instead of retrying
		return Permanent(fmt.Errorf("failed to publish message: %w", err))
	}

	return nil
}

func truncateContent(content string, limit int) string {
	if limit <= 0 || len(content) <= limit {
		return content
	}
	if limit <= 3 {
		return content[:limit]
	}
	return content[:limit-3] + "..."
}
