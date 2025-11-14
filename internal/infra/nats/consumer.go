package nats

import (
	"caatsm/internal/app"
	"caatsm/internal/infra/config"
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"github.com/nats-io/nats.go"
	"time"
)

// Consumer handles NATS JetStream message consumption
type Consumer struct {
	js        nats.JetStreamContext
	processor *app.MessageProcessor
	cfg       *config.Config
	logger    *zap.Logger
	subject   string
	consumerName string
}

// ProvideConsumer creates a NATS consumer
func ProvideConsumer(
	js nats.JetStreamContext,
	processor *app.MessageProcessor,
	cfg *config.Config,
	logger *zap.Logger,
) (*Consumer, error) {
	subject := cfg.Subscription.Topic
	if subject == "" {
		subject = "telegram.>"
	}

	consumerName := cfg.NATS.Consumer
	if consumerName == "" {
		consumerName = "telegram-consumer"
	}

	consumer := &Consumer{
		js:           js,
		processor:    processor,
		cfg:          cfg,
		logger:       logger,
		subject:      subject,
		consumerName: consumerName,
	}

	// Create consumer if it doesn't exist
	if err := consumer.ensureConsumer(); err != nil {
		return nil, fmt.Errorf("failed to ensure consumer: %w", err)
	}

	return consumer, nil
}

// ensureConsumer creates the consumer if it doesn't exist
func (c *Consumer) ensureConsumer() error {
	streamName := c.cfg.NATS.Stream
	if streamName == "" {
		streamName = "TELEGRAM"
	}

	consumerConfig := &nats.ConsumerConfig{
		Durable:       c.consumerName,
		DeliverPolicy: nats.DeliverAllPolicy,
		AckPolicy:     nats.AckExplicitPolicy,
		AckWait:       c.cfg.Timeouts.AckWait,
		MaxDeliver:    5, // Maximum number of delivery attempts
		FilterSubject: c.subject,
	}

	_, err := c.js.AddConsumer(streamName, consumerConfig)
	if err != nil && err != nats.ErrConsumerNameAlreadyInUse {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	if err == nil {
		c.logger.Info("Created JetStream consumer",
			zap.String("consumer", c.consumerName),
			zap.String("stream", streamName),
			zap.String("subject", c.subject),
		)
	}

	return nil
}

// Start starts consuming messages
func (c *Consumer) Start(ctx context.Context) error {
	streamName := c.cfg.NATS.Stream
	if streamName == "" {
		streamName = "TELEGRAM"
	}

	// Create pull subscription
	sub, err := c.js.PullSubscribe(c.subject, c.consumerName, nats.Bind(streamName, c.consumerName))
	if err != nil {
		return fmt.Errorf("failed to create pull subscription: %w", err)
	}
	defer sub.Unsubscribe()

	c.logger.Info("Started consuming messages",
		zap.String("subject", c.subject),
		zap.String("consumer", c.consumerName),
		zap.String("stream", streamName),
	)

	batchSize := c.cfg.App.BatchSize
	if batchSize == 0 {
		batchSize = 50
	}
	batchTimeout := c.cfg.App.BatchTimeout
	if batchTimeout == 0 {
		batchTimeout = 2 * time.Second
	}

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Stopping consumer", zap.Error(ctx.Err()))
			return ctx.Err()
		default:
		}

		// Fetch messages in batch
		msgs, err := sub.Fetch(batchSize, nats.MaxWait(batchTimeout))
		if err != nil {
			if errors.Is(err, nats.ErrTimeout) {
				// Timeout is expected when no messages are available
				continue
			}
			c.logger.Error("Failed to fetch messages", zap.Error(err))
			time.Sleep(time.Second)
			continue
		}

		// Process each message
		for _, msg := range msgs {
			if err := c.processMessage(ctx, msg); err != nil {
				c.logger.Error("Failed to process message",
					zap.String("subject", msg.Subject),
					zap.Error(err),
				)
				// NAK the message to retry
				if nakErr := msg.Nak(); nakErr != nil {
					c.logger.Error("Failed to NAK message", zap.Error(nakErr))
				}
			} else {
				// ACK the message
				if ackErr := msg.Ack(); ackErr != nil {
					c.logger.Error("Failed to ACK message", zap.Error(ackErr))
				}
			}
		}
	}
}

// processMessage processes a single message
func (c *Consumer) processMessage(ctx context.Context, msg *nats.Msg) error {
	msgID := msg.Header.Get("Nats-Msg-Id")
	if msgID == "" {
		// Use reply subject or generate a simple ID
		if msg.Reply != "" {
			msgID = msg.Reply
		} else {
			msgID = fmt.Sprintf("msg-%d", time.Now().UnixNano())
		}
	}

	c.logger.Debug("Processing message",
		zap.String("subject", msg.Subject),
		zap.String("msg_id", msgID),
		zap.Int("data_size", len(msg.Data)),
	)

	// Call processor
	if err := c.processor.Handle(ctx, msg.Data, msgID); err != nil {
		return fmt.Errorf("processor error: %w", err)
	}

	return nil
}
