package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"caatsm/internal/config"
	applog "caatsm/internal/infra/log"

	"github.com/nats-io/nats.go"
)

type Publisher struct {
	js     nats.JetStreamContext
	config *config.Config
}

func NewPublisher(js nats.JetStreamContext, cfg *config.Config) (*Publisher, error) {
	return &Publisher{
		js:     js,
		config: cfg,
	}, nil
}

// Publish publishes a message to JetStream synchronously
func (p *Publisher) Publish(ctx context.Context, subject string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	_, err = p.js.Publish(subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// PublishAsync publishes a message to JetStream asynchronously
func (p *Publisher) PublishAsync(ctx context.Context, subject string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	pubAckFuture, err := p.js.PublishAsync(subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish message async: %w", err)
	}

	go func() {
		if err := pubAckFuture.Ok(); err != nil {
			logger := applog.Sugared()
			logger.Errorf("Failed to publish message asynchronously: %v", err)
		}
	}()

	return nil
}
