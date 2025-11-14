package nats

import (
	"caatsm/internal/adapter"
	"caatsm/internal/infra/config"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"github.com/nats-io/nats.go"
)

// Publisher publishes messages to NATS JetStream
type Publisher struct {
	js     nats.JetStreamContext
	cfg    *config.Config
	logger *zap.Logger
}

// ProvidePublisher creates a NATS publisher
func ProvidePublisher(
	js nats.JetStreamContext,
	cfg *config.Config,
	logger *zap.Logger,
) (adapter.Publisher, error) {
	return &Publisher{
		js:     js,
		cfg:    cfg,
		logger: logger,
	}, nil
}

// Publish publishes a message
func (p *Publisher) Publish(message interface{}) error {
	topic := p.cfg.Publisher.Topic
	if topic == "" {
		return fmt.Errorf("publisher topic is not configured")
	}

	// Marshal message to JSON
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Publish to JetStream
	_, err = p.js.Publish(topic, messageBytes)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	p.logger.Debug("Published message",
		zap.String("topic", topic),
		zap.Int("size", len(messageBytes)),
	)

	return nil
}
