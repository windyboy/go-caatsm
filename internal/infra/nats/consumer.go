package nats

import (
	"caatsm/internal/app"
	"caatsm/internal/infra/config"
	obsmetrics "caatsm/internal/infra/metrics"
	"caatsm/internal/infra/telemetry"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

// Consumer handles NATS JetStream message consumption.
// It consolidates stream management, fetching, and processing into a single, cohesive unit.
type Consumer struct {
	// Core dependencies
	conn      *nats.Conn
	js        nats.JetStreamContext
	processor *app.MessageProcessor
	cfg       *config.Config
	logger    *zap.Logger
	telemetry telemetry.Recorder

	// Components
	monitor    *ConsumerMonitor
	dlqHandler DLQHandler

	// Configuration
	streamName   string
	consumerName string
	subject      string
	batchSize    int
	batchTimeout time.Duration
	ackWait      time.Duration
	backoff      []time.Duration

	// State
	consecutiveErrors int
}

// ProvideConsumer initializes a NATS consumer, ensuring infrastructure exists.
func ProvideConsumer(
	conn *nats.Conn,
	js nats.JetStreamContext,
	processor *app.MessageProcessor,
	cfg *config.Config,
	rec telemetry.Recorder,
	logger *zap.Logger,
) (*Consumer, error) {
	// 1. Normalize Configuration
	c := &Consumer{
		conn:         conn,
		js:           js,
		processor:    processor,
		cfg:          cfg,
		logger:       logger,
		telemetry:    rec,
		streamName:   orDefault(cfg.NATS.Stream, "TELEGRAM"),
		consumerName: orDefault(cfg.NATS.Consumer, "telegram-consumer"),
		subject:      cfg.EffectiveSubscriptionTopic(),
		batchSize:    cfg.App.BatchSize,
		batchTimeout: cfg.App.BatchTimeout,
		ackWait:      orDefaultDuration(cfg.NATS.ConsumerRules.AckWait, 30*time.Second),
		backoff:      cfg.NATS.ConsumerRules.Backoff,
	}

	if c.batchSize <= 0 {
		c.batchSize = 50
	}
	if c.batchTimeout <= 0 {
		c.batchTimeout = 2 * time.Second
	}

	// 2. Initialize Components
	c.monitor = NewConsumerMonitor(logger, cfg, js, c.streamName, c.consumerName, cfg.App.MonitorInterval)
	if cfg.DLQ.Enabled && cfg.DLQ.Subject != "" {
		c.dlqHandler = &defaultDLQHandler{
			publisher:    js,
			dlqSubject:   cfg.DLQ.Subject,
			streamName:   c.streamName,
			consumerName: c.consumerName,
			logger:       logger,
			telemetry:    rec,
		}
	}

	// 3. Ensure Infrastructure (Stream & Consumer)
	if err := c.ensureInfrastructure(); err != nil {
		return nil, err
	}

	return c, nil
}

// Start begins the main consumption loop.
func (c *Consumer) Start(ctx context.Context) error {
	c.logger.Info("Starting consumer",
		zap.String("stream", c.streamName),
		zap.String("consumer", c.consumerName),
		zap.String("subject", c.subject),
	)

	// Start background monitoring
	monitorCtx, cancelMonitor := context.WithCancel(ctx)
	defer cancelMonitor()
	go c.monitor.Start(monitorCtx)

	// Create subscription
	sub, err := c.js.PullSubscribe(c.subject, c.consumerName, nats.BindStream(c.streamName))
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}
	defer sub.Unsubscribe()

	// Initial metric recording
	if info, err := c.js.ConsumerInfo(c.streamName, c.consumerName); err == nil {
		c.monitor.RecordInitialPending(info.NumPending)
	}

	// Main Loop
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		msgs, err := sub.Fetch(c.batchSize, nats.MaxWait(c.batchTimeout))
		if err != nil {
			if errors.Is(err, nats.ErrTimeout) {
				continue // Normal timeout, just retry
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			// Log other errors but keep loop alive unless critical
			c.logger.Warn("Fetch error", zap.Error(err))
			time.Sleep(100 * time.Millisecond) // Slight backoff
			continue
		}

		c.processBatch(ctx, msgs)
	}
}

// Shutdown gracefully drains the connection.
func (c *Consumer) Shutdown(ctx context.Context) error {
	if c.conn == nil {
		return nil
	}
	c.logger.Info("Draining NATS connection...")
	return c.conn.Drain()
}

// processBatch iterates through a batch of messages.
func (c *Consumer) processBatch(ctx context.Context, msgs []*nats.Msg) {
	for _, msg := range msgs {
		select {
		case <-ctx.Done():
			return
		default:
			c.monitor.TrackMessage(msg)
			c.processMsg(ctx, msg)
		}
	}
}

// processMsg handles a single message: Trace -> App Logic -> Ack/Nak.
func (c *Consumer) processMsg(ctx context.Context, msg *nats.Msg) {
	start := time.Now()
	ctx, span := otel.Tracer("caatsm/nats").Start(ctx, "Consumer.processMsg")
	defer span.End()

	msgID := c.resolveMsgID(msg)
	
	// Add metadata to span/logger
	span.SetAttributes(
		attribute.String("messaging.system", "nats"),
		attribute.String("messaging.message_id", msgID),
		attribute.String("caatsm.stream", c.streamName),
	)

	// Execute Application Logic
	err := c.processor.Handle(ctx, msg.Data, msgID)
	
	// Handle Result
	if err != nil {
		c.handleError(ctx, msg, msgID, err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		c.telemetry.RecordMessageHandled(ctx, c.streamName, c.consumerName, obsmetrics.ResultFail, time.Since(start))
	} else {
		// Success
		if c.consecutiveErrors > 0 {
			c.consecutiveErrors = 0
		}
		if ackErr := msg.Ack(); ackErr != nil {
			c.logger.Warn("Failed to ACK", zap.String("msg_id", msgID), zap.Error(ackErr))
		}
		c.telemetry.RecordMessageHandled(ctx, c.streamName, c.consumerName, "ok", time.Since(start))
	}
}

// handleError decides whether to Ack (Permanent/DLQ) or Nak (Transient).
func (c *Consumer) handleError(ctx context.Context, msg *nats.Msg, msgID string, err error) {
	isPermanent := app.IsPermanent(err)
	c.logger.Error("Processing failed",
		zap.String("msg_id", msgID),
		zap.Error(err),
		zap.Bool("permanent", isPermanent),
	)

	if isPermanent {
		// Poison message: Route to DLQ -> Ack only if DLQ succeeds or DLQ is disabled
		c.consecutiveErrors = 0
		if c.dlqHandler != nil {
			if dlqErr := c.dlqHandler.RouteToDLQ(ctx, msg, err); dlqErr != nil {
				c.logger.Error("DLQ publish failed, NAKing original message for redelivery",
					zap.String("msg_id", msgID),
					zap.Error(dlqErr),
				)
				c.telemetry.RecordDLQPublishFailure(ctx, c.streamName, c.consumerName)
				_ = msg.Nak()
				return
			}
		}
		_ = msg.Ack()
		return
	}

	// Transient error: check if MaxDeliver exhausted
	meta, metaErr := msg.Metadata()
	if metaErr == nil && int(meta.NumDelivered) >= c.cfg.NATS.ConsumerRules.MaxDeliver {
		c.logger.Error("Transient error exhausted max deliveries, routing to DLQ",
			zap.String("msg_id", msgID),
			zap.Uint64("delivered", meta.NumDelivered),
			zap.Int("max_deliver", c.cfg.NATS.ConsumerRules.MaxDeliver),
		)
		c.consecutiveErrors = 0
		if c.dlqHandler != nil {
			if dlqErr := c.dlqHandler.RouteToDLQ(ctx, msg, err); dlqErr != nil {
				c.logger.Error("DLQ publish failed for exhausted message, NAKing for redelivery",
					zap.String("msg_id", msgID),
					zap.Error(dlqErr),
				)
				c.telemetry.RecordDLQPublishFailure(ctx, c.streamName, c.consumerName)
				_ = msg.Nak()
				return
			}
		}
		_ = msg.Ack()
		return
	}

	// Transient error: Backpressure -> Nak with Backoff
	c.consecutiveErrors++
	c.applyBackpressure(ctx)
	_ = c.nakWithBackoff(msg)
}

// nakWithBackoff calculates the appropriate NAK delay based on delivery attempts.
func (c *Consumer) nakWithBackoff(msg *nats.Msg) error {
	if len(c.backoff) == 0 {
		return msg.Nak()
	}
	meta, err := msg.Metadata()
	if err != nil {
		return msg.Nak()
	}
	
	// attempt is 1-based, index is 0-based
	attempt := int(meta.NumDelivered)
	index := attempt - 1
	if index >= len(c.backoff) {
		index = len(c.backoff) - 1
	} else if index < 0 {
		index = 0
	}
	
	return msg.NakWithDelay(c.backoff[index])
}

// applyBackpressure sleeps if error streak is high to protect the system.
func (c *Consumer) applyBackpressure(ctx context.Context) {
	if c.consecutiveErrors < 10 {
		return
	}
	delay := time.Duration(c.consecutiveErrors) * 100 * time.Millisecond
	if delay > 5*time.Second {
		delay = 5 * time.Second
	}
	
	select {
	case <-time.After(delay):
	case <-ctx.Done():
	}
}

// ensureInfrastructure creates the Stream and Consumer if they don't exist.
func (c *Consumer) ensureInfrastructure() error {
	// 1. Ensure Stream
	subjects := []string{c.subject}
	if c.cfg.Publisher.Topic != "" {
		subjects = append(subjects, c.cfg.Publisher.Topic)
	}
	if c.cfg.DLQ.Enabled && c.cfg.DLQ.Subject != "" {
		subjects = append(subjects, c.cfg.DLQ.Subject)
	}
	
	streamCfg := &nats.StreamConfig{
		Name:     c.streamName,
		Subjects: dedupeSubjects(subjects),
		Retention: nats.WorkQueuePolicy, // Defaulting to WorkQueue for queues
		MaxMsgs:  c.cfg.NATS.StreamLimits.MaxMsgs,
		MaxBytes: c.cfg.NATS.StreamLimits.MaxBytes,
		MaxAge:   c.cfg.NATS.StreamLimits.MaxAge,
		Replicas: c.cfg.NATS.StreamLimits.Replicas,
		Storage:  nats.FileStorage,
	}
	if c.cfg.NATS.StreamLimits.Discard == "new" {
		streamCfg.Discard = nats.DiscardNew
	}
	if c.cfg.NATS.StreamLimits.Storage == "memory" {
		streamCfg.Storage = nats.MemoryStorage
	}

	// Idempotent add/update
	if _, err := c.js.AddStream(streamCfg); err != nil {
		return fmt.Errorf("ensure stream: %w", err)
	}

	// 2. Ensure Consumer
	consumerCfg := &nats.ConsumerConfig{
		Durable:       c.consumerName,
		FilterSubject: c.subject,
		AckPolicy:     nats.AckExplicitPolicy,
		AckWait:       c.ackWait,
		MaxDeliver:    c.cfg.NATS.ConsumerRules.MaxDeliver,
		MaxAckPending: c.cfg.NATS.ConsumerRules.MaxAckPending,
		ReplayPolicy:  nats.ReplayInstantPolicy,
	}
	if c.cfg.NATS.ConsumerRules.ReplayPolicy == "original" {
		consumerCfg.ReplayPolicy = nats.ReplayOriginalPolicy
	}
	
	// Idempotent add/update
	if _, err := c.js.AddConsumer(c.streamName, consumerCfg); err != nil {
		return fmt.Errorf("ensure consumer: %w", err)
	}

	return nil
}

// resolveMsgID extracts the ID from headers or metadata.
func (c *Consumer) resolveMsgID(msg *nats.Msg) string {
	if id := msg.Header.Get("Nats-Msg-Id"); id != "" {
		return id
	}
	if meta, err := msg.Metadata(); err == nil {
		return fmt.Sprintf("js-%d", meta.Sequence.Stream)
	}
	return "unknown"
}

// --- Helpers ---

func orDefault(val, def string) string {
	if val != "" {
		return val
	}
	return def
}

func orDefaultDuration(val, def time.Duration) time.Duration {
	if val > 0 {
		return val
	}
	return def
}