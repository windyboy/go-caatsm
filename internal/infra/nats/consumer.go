package nats

import (
	"caatsm/internal/app"
	"caatsm/internal/infra/config"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
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
	meter        metric.Meter
	ackPending   metric.Int64Histogram
	redelivered  metric.Int64Histogram
	pending      metric.Int64Histogram
	delivered    metric.Int64Histogram
}

// ProvideConsumer creates a NATS consumer
func ProvideConsumer(
	conn *nats.Conn,
	js nats.JetStreamContext,
	processor *app.MessageProcessor,
	cfg *config.Config,
	logger *zap.Logger,
) (*Consumer, error) {
	subject := cfg.EffectiveSubscriptionTopic()

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
	consumer.initMetrics()

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
	c.cfg.NATS.ConsumerRules.AckWait = ackWait

	consumerConfig := &nats.ConsumerConfig{
		Durable:       c.consumerName,
		DeliverPolicy: mapDeliverPolicy(c.cfg.NATS.ConsumerRules.DeliverPolicy),
		AckPolicy:     nats.AckExplicitPolicy,
		AckWait:       ackWait,
		ReplayPolicy:  mapReplayPolicy(c.cfg.NATS.ConsumerRules.ReplayPolicy),
		MaxDeliver:    c.cfg.NATS.ConsumerRules.MaxDeliver,
		MaxAckPending: c.cfg.NATS.ConsumerRules.MaxAckPending,
		FilterSubject: c.subject,
		BackOff:       c.cfg.NATS.ConsumerRules.Backoff,
	}
	if consumerConfig.DeliverPolicy == nats.DeliverByStartSequencePolicy && c.cfg.NATS.ConsumerRules.StartSequence > 0 {
		consumerConfig.OptStartSeq = c.cfg.NATS.ConsumerRules.StartSequence
	}
	if consumerConfig.DeliverPolicy == nats.DeliverByStartTimePolicy && strings.TrimSpace(c.cfg.NATS.ConsumerRules.StartTime) != "" {
		startTime, err := time.Parse(time.RFC3339, c.cfg.NATS.ConsumerRules.StartTime)
		if err != nil {
			c.logger.Warn("Invalid start time, falling back to deliver policy defaults",
				zap.String("start_time", c.cfg.NATS.ConsumerRules.StartTime),
				zap.Error(err),
			)
		} else {
			consumerConfig.OptStartTime = &startTime
		}
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
			zap.Duration("ack_wait", ackWait),
			zap.String("deliver_policy", c.cfg.NATS.ConsumerRules.DeliverPolicy),
			zap.String("replay_policy", c.cfg.NATS.ConsumerRules.ReplayPolicy),
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
		zap.String("deliver_policy", c.cfg.NATS.ConsumerRules.DeliverPolicy),
		zap.String("replay_policy", c.cfg.NATS.ConsumerRules.ReplayPolicy),
		zap.Int("backoff_steps", len(c.cfg.NATS.ConsumerRules.Backoff)),
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
		// TODO: consider buffering messages to take advantage of Repository.InsertBatch for higher throughput.
		for _, msg := range msgs {
			if err := c.processMessage(ctx, msg); err != nil {
				isPermanent := app.IsPermanent(err)
				c.logger.Error("Failed to process message",
					zap.String("subject", msg.Subject),
					zap.Error(err),
					zap.Bool("permanent", isPermanent),
				)

				if isPermanent {
					if ackErr := msg.Ack(); ackErr != nil {
						c.logger.Error("Failed to ACK permanent-error message", zap.Error(ackErr))
					}
					continue
				}

				// Transient error: request redelivery with optional delay
				if nakErr := c.nakWithStrategy(msg); nakErr != nil {
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
			c.recordConsumerMetrics(ctx, info)
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

func (c *Consumer) initMetrics() {
	meter := otel.Meter("caatsm/nats")
	c.meter = meter

	if hist, err := meter.Int64Histogram("nats.consumer.ack_pending"); err == nil {
		c.ackPending = hist
	}
	if hist, err := meter.Int64Histogram("nats.consumer.redelivered"); err == nil {
		c.redelivered = hist
	}
	if hist, err := meter.Int64Histogram("nats.consumer.pending"); err == nil {
		c.pending = hist
	}
	if hist, err := meter.Int64Histogram("nats.consumer.delivered"); err == nil {
		c.delivered = hist
	}
}

func (c *Consumer) recordConsumerMetrics(ctx context.Context, info *nats.ConsumerInfo) {
	if info == nil {
		return
	}
	if c.ackPending != nil {
		c.ackPending.Record(ctx, int64(info.NumAckPending))
	}
	if c.redelivered != nil {
		c.redelivered.Record(ctx, int64(info.NumRedelivered))
	}
	if c.pending != nil {
		c.pending.Record(ctx, int64(info.NumPending))
	}
	if c.delivered != nil {
		c.delivered.Record(ctx, int64(info.Delivered.Stream))
	}
}

func (c *Consumer) nakWithStrategy(msg *nats.Msg) error {
	backoff := c.cfg.NATS.ConsumerRules.Backoff
	if len(backoff) == 0 {
		return msg.Nak()
	}

	meta, err := msg.Metadata()
	if err != nil {
		c.logger.Warn("Failed to read metadata for backoff strategy", zap.Error(err))
		return msg.Nak()
	}

	attempt := int(meta.NumDelivered)
	index := attempt - 1
	if index < 0 {
		index = 0
	}
	if index >= len(backoff) {
		index = len(backoff) - 1
	}
	delay := backoff[index]
	if delay <= 0 {
		return msg.Nak()
	}

	return msg.NakWithDelay(delay)
}

// processMessage processes a single message
func (c *Consumer) processMessage(ctx context.Context, msg *nats.Msg) error {
	ctx, span := otel.Tracer("caatsm/nats").Start(ctx, "Consumer.processMessage")
	defer span.End()
	span.SetAttributes(attribute.String("nats.subject", msg.Subject))

	msgID, source, err := c.resolveMsgID(msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("unable to resolve message id: %w", err)
	}
	if source != "header" {
		c.logger.Warn("Message missing NATS id header; using fallback",
			zap.String("subject", msg.Subject),
			zap.String("msg_id_source", source),
			zap.String("msg_id", msgID),
		)
	}

	c.logger.Debug("Processing message",
		zap.String("subject", msg.Subject),
		zap.String("msg_id", msgID),
		zap.Int("data_size", len(msg.Data)),
	)

	// Call processor
	if err := c.processor.Handle(ctx, msg.Data, msgID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("processor error: %w", err)
	}

	span.SetAttributes(attribute.String("telegram.msg_id", msgID))
	return nil
}

func (c *Consumer) resolveMsgID(msg *nats.Msg) (string, string, error) {
	if id := msg.Header.Get("Nats-Msg-Id"); id != "" {
		return id, "header", nil
	}

	meta, err := msg.Metadata()
	if err != nil {
		return "", "", fmt.Errorf("fetch metadata: %w", err)
	}

	return fmt.Sprintf("js-%d", meta.Sequence.Stream), "metadata", nil
}

func mapDeliverPolicy(value string) nats.DeliverPolicy {
	switch strings.ToLower(value) {
	case "new":
		return nats.DeliverNewPolicy
	case "last":
		return nats.DeliverLastPolicy
	case "last_per_subject":
		return nats.DeliverLastPerSubjectPolicy
	case "sequence":
		return nats.DeliverByStartSequencePolicy
	case "time":
		return nats.DeliverByStartTimePolicy
	default:
		return nats.DeliverAllPolicy
	}
}

func mapReplayPolicy(value string) nats.ReplayPolicy {
	switch strings.ToLower(value) {
	case "original":
		return nats.ReplayOriginalPolicy
	default:
		return nats.ReplayInstantPolicy
	}
}
