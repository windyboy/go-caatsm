package nats

import (
	"caatsm/internal/adapter/dto"
	"caatsm/internal/infra/config"
	"caatsm/internal/port"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// Publisher publishes messages to NATS JetStream
type Publisher struct {
	js     nats.JetStreamContext
	cfg    *config.Config
	logger *zap.Logger
}

// ProvidePublisher creates a NATS JetStream publisher.
func ProvidePublisher(
	js nats.JetStreamContext,
	nc *nats.Conn,
	cfg *config.Config,
	logger *zap.Logger,
) (port.Publisher, error) {
	if js == nil {
		return nil, fmt.Errorf("JetStream context is required")
	}
	return &Publisher{
		js:     js,
		cfg:    cfg,
		logger: logger,
	}, nil
}

// Publish publishes a message
func (p *Publisher) Publish(message any) error {
	topic := p.cfg.Publisher.Topic
	if topic == "" {
		p.logger.Error("publisher topic is not configured")
		return fmt.Errorf("publisher topic is not configured")
	}

	// Marshal message to JSON
	messageBytes, err := json.Marshal(message)
	if err != nil {
		p.logger.Error("failed to marshal message",
			zap.String("topic", topic),
			zap.Error(err),
		)
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Build JetStream message to attach dedup headers
	jsMsg := nats.NewMsg(topic)
	jsMsg.Data = messageBytes

	switch typed := message.(type) {
	case *dto.ParsedTelegram:
		if typed != nil && typed.Uuid != "" {
			jsMsg.Header.Set("Nats-Msg-Id", typed.Uuid)
		} else {
			jsMsg.Header.Set("Nats-Msg-Id", uuid.NewString())
		}
	default:
		jsMsg.Header.Set("Nats-Msg-Id", uuid.NewString())
	}

	// Publish to JetStream
	_, err = p.js.PublishMsg(jsMsg)
	if err != nil {
		// Distinguish temporary JetStream unavailability from permanent config errors.
		if errors.Is(err, nats.ErrNoResponders) {
			p.logger.Error("transient publish error (no responders)",
				zap.String("topic", topic),
				zap.Int("message_size", len(messageBytes)),
				zap.Error(err),
			)
			return fmt.Errorf("transient publish error (no responders): %w", err)
		}
		p.logger.Error("failed to publish message",
			zap.String("topic", topic),
			zap.Int("message_size", len(messageBytes)),
			zap.Error(err),
		)
		return fmt.Errorf("failed to publish message: %w", err)
	}

	p.logger.Debug("Published message",
		zap.String("topic", topic),
		zap.Int("size", len(messageBytes)),
	)

	return nil
}
