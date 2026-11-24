package nats

import (
	"caatsm/internal/config"
	"caatsm/internal/observability"
	"caatsm/internal/pipeline"
	"caatsm/internal/telegram"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

// Publisher publishes parsed telegram messages to JetStream with OTel trace propagation.
type Publisher struct {
	js     nats.JetStreamContext
	cfg    *config.Config
	logger *zap.Logger
}

// NewPublisher creates a NATS JetStream publisher.
func NewPublisher(
	js nats.JetStreamContext,
	cfg *config.Config,
	logger *zap.Logger,
) (*Publisher, error) {
	if js == nil {
		return nil, fmt.Errorf("JetStream context is required")
	}
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Publisher{
		js:     js,
		cfg:    cfg,
		logger: logger,
	}, nil
}

// Publish serializes msg, injects OpenTelemetry trace context headers, and publishes to JetStream.
func (p *Publisher) Publish(ctx context.Context, msg *telegram.Message) error {
	if msg == nil {
		return pipeline.Permanent(fmt.Errorf("message must not be nil"))
	}

	topic := p.cfg.Publisher.Topic
	if topic == "" {
		p.logger.Error("publisher topic is not configured")
		return pipeline.Permanent(fmt.Errorf("publisher topic is not configured"))
	}

	messageBytes, err := json.Marshal(msg)
	if err != nil {
		p.logger.Error("failed to marshal message", zap.String("topic", topic), zap.Error(err))
		return pipeline.Permanent(fmt.Errorf("failed to marshal message: %w", err))
	}

	jsMsg := nats.NewMsg(topic)
	jsMsg.Data = messageBytes
	if jsMsg.Header == nil {
		jsMsg.Header = make(nats.Header)
	}

	if msg.UUID != "" {
		jsMsg.Header.Set("Nats-Msg-Id", msg.UUID)
	} else {
		jsMsg.Header.Set("Nats-Msg-Id", uuid.NewString())
	}

	otel.GetTextMapPropagator().Inject(ctx, natsHeaderCarrier(jsMsg.Header))
	observability.RecordJSAPICall("publish")

	if _, err := p.js.PublishMsg(jsMsg, nats.Context(ctx)); err != nil {
		if isTransientPublishError(err) {
			p.logger.Error("transient publish error",
				zap.String("topic", topic),
				zap.Int("message_size", len(messageBytes)),
				zap.Error(err),
			)
			return fmt.Errorf("transient publish error: %w", err)
		}
		p.logger.Error("permanent publish error",
			zap.String("topic", topic),
			zap.Int("message_size", len(messageBytes)),
			zap.Error(err),
		)
		return pipeline.Permanent(fmt.Errorf("failed to publish message: %w", err))
	}

	p.logger.Debug("Published message",
		zap.String("topic", topic),
		zap.Int("size", len(messageBytes)),
	)
	return nil
}

func isTransientPublishError(err error) bool {
	return errors.Is(err, nats.ErrNoResponders) ||
		errors.Is(err, nats.ErrTimeout) ||
		errors.Is(err, nats.ErrConnectionClosed) ||
		errors.Is(err, nats.ErrDisconnected) ||
		errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled)
}
