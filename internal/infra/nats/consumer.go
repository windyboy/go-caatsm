package nats

import (
	"caatsm/internal/app"
	"caatsm/internal/infra/config"
	"context"
	"errors"
	"fmt"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"time"
)

// Consumer handles NATS JetStream message consumption
type Consumer struct {
	conn         *nats.Conn
	js           nats.JetStreamContext
	processor    *app.MessageProcessor
	cfg          *config.Config
	logger       *zap.Logger
	subject      string
	consumerName string
}

// ProvideConsumer creates a NATS consumer
func ProvideConsumer(
	conn *nats.Conn,
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
		conn:         conn,
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

	ackWait := c.cfg.NATS.ConsumerRules.AckWait
	if ackWait == 0 {
		ackWait = c.cfg.Timeouts.AckWait
	}
	if ackWait == 0 {
		ackWait = 30 * time.Second
	}

	consumerConfig := &nats.ConsumerConfig{
		Durable:       c.consumerName,
		DeliverPolicy: nats.DeliverAllPolicy,
		AckPolicy:     nats.AckExplicitPolicy,
		AckWait:       ackWait,
		MaxDeliver:    c.cfg.NATS.ConsumerRules.MaxDeliver,
		MaxAckPending: c.cfg.NATS.ConsumerRules.MaxAckPending,
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

	c.logger.Info("Consumer pull configuration",
		zap.Int("batch_size", batchSize),
		zap.Duration("batch_timeout", batchTimeout),
		zap.Int("max_deliver", c.cfg.NATS.ConsumerRules.MaxDeliver),
		zap.Duration("ack_wait", c.cfg.NATS.ConsumerRules.AckWait),
	)

	statsCtx, statsCancel := context.WithCancel(ctx)
	defer statsCancel()
	go c.emitConsumerStats(statsCtx, streamName)

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
				isPermanent := app.IsPermanent(err)
				c.logger.Error("Failed to process message",
					zap.String("subject", msg.Subject),
					zap.Error(err),
					zap.Bool("permanent", isPermanent),
				)

				if isPermanent {
					if termErr := msg.Term(); termErr != nil {
						c.logger.Error("Failed to TERM message", zap.Error(termErr))
					}
					continue
				}

				// Transient error: request redelivery
				if nakErr := msg.Nak(); nakErr != nil {
					c.logger.Error("Failed to NAK message", zap.Error(nakErr))
				}
				continue
			}

			// ACK the message
			if ackErr := msg.Ack(); ackErr != nil {
				c.logger.Error("Failed to ACK message", zap.Error(ackErr))
			}
		}
	}
}

func (c *Consumer) emitConsumerStats(ctx context.Context, streamName string) {
	interval := c.cfg.App.MonitorInterval
	if interval <= 0 {
		interval = 30 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			info, err := c.js.ConsumerInfo(streamName, c.consumerName)
			if err != nil {
				c.logger.Warn("Failed to fetch consumer info", zap.Error(err))
				continue
			}

			c.logger.Info("JetStream consumer metrics",
				zap.String("stream", streamName),
				zap.String("consumer", c.consumerName),
				zap.Uint64("num_ack_pending", uint64(info.NumAckPending)),
				zap.Uint64("num_redelivered", uint64(info.NumRedelivered)),
				zap.Uint64("num_pending", uint64(info.NumPending)),
				zap.Uint64("delivered_consumer_seq", uint64(info.Delivered.Consumer)),
				zap.Uint64("delivered_stream_seq", uint64(info.Delivered.Stream)),
			)
		}
	}
}

// Shutdown drains the underlying NATS connection gracefully.
func (c *Consumer) Shutdown(ctx context.Context) error {
	if c.conn == nil {
		return nil
	}

	timeout := c.cfg.Timeouts.Close
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	closeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.conn.Drain()
	}()

	select {
	case err := <-errCh:
		c.conn.Close()
		return err
	case <-closeCtx.Done():
		c.conn.Close()
		return fmt.Errorf("nats drain timeout: %w", closeCtx.Err())
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
