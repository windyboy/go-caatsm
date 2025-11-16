package nats

import (
	"caatsm/internal/app"
	"caatsm/internal/infra/config"
	obslogging "caatsm/internal/observability/logging"
	obsmetrics "caatsm/internal/observability/metrics"
	"caatsm/internal/observability/telemetry"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

// Consumer handles NATS JetStream message consumption
type Consumer struct {
	conn            *nats.Conn
	js              nats.JetStreamContext
	processor       *app.MessageProcessor
	cfg             *config.Config
	logger          *zap.Logger
	telemetry       telemetry.Recorder
	subject         string
	consumerName    string
	mode            string
	streamName      string
	dlqSubject      string
	ackWait         time.Duration
	batchSize       int
	batchTimeout    time.Duration
	monitorInterval time.Duration
	meter           metric.Meter
	ackPending      metric.Int64Histogram
	redelivered     metric.Int64Histogram
	pending         metric.Int64Histogram
	delivered       metric.Int64Histogram

	// simple backpressure / degradation state
	consecutiveProcessErrors int
}

// ProvideConsumer creates a NATS consumer
func ProvideConsumer(
	conn *nats.Conn,
	js nats.JetStreamContext,
	processor *app.MessageProcessor,
	cfg *config.Config,
	rec telemetry.Recorder,
	logger *zap.Logger,
) (*Consumer, error) {
	subject := cfg.EffectiveSubscriptionTopic()

	consumerName := cfg.NATS.Consumer
	if consumerName == "" {
		consumerName = "telegram-consumer"
	}

	mode := strings.ToLower(cfg.NATS.Mode)
	if mode == "" {
		mode = "jetstream"
	}

	streamName := cfg.NATS.Stream
	if streamName == "" {
		streamName = "TELEGRAM"
	}

	// DLQ routing is only meaningful in JetStream mode. Respect dlq.enabled to allow
	// environments to opt out cleanly even if a subject is configured.
	dlqSubject := ""
	if cfg.DLQ.Enabled {
		dlqSubject = strings.TrimSpace(cfg.DLQ.Subject)
	}

	ackWait := cfg.NATS.ConsumerRules.AckWait
	if ackWait == 0 {
		ackWait = cfg.Timeouts.AckWait
	}
	if ackWait == 0 {
		ackWait = 30 * time.Second
	}

	batchSize := cfg.App.BatchSize
	if batchSize == 0 {
		batchSize = 50
	}

	batchTimeout := cfg.App.BatchTimeout
	if batchTimeout == 0 {
		batchTimeout = 2 * time.Second
	}

	monitorInterval := cfg.App.MonitorInterval
	if monitorInterval <= 0 {
		monitorInterval = 30 * time.Second
	}

	consumer := &Consumer{
		conn:            conn,
		js:              js,
		processor:       processor,
		cfg:             cfg,
		logger:          logger,
		telemetry:       rec,
		subject:         subject,
		consumerName:    consumerName,
		mode:            mode,
		streamName:      streamName,
		dlqSubject:      dlqSubject,
		ackWait:         ackWait,
		batchSize:       batchSize,
		batchTimeout:    batchTimeout,
		monitorInterval: monitorInterval,
	}
	consumer.initMetrics()

	if consumer.mode == "jetstream" {
		// Create consumer if it doesn't exist
		if err := consumer.ensureConsumer(); err != nil {
			return nil, fmt.Errorf("failed to ensure consumer: %w", err)
		}
		// Validate DLQ configuration early so misconfiguration is visible at startup
		// rather than only when the first poison message appears.
		consumer.validateDLQ()
	} else {
		logger.Info("Running consumer in core NATS mode",
			zap.String("subject", subject),
			zap.String("queue_group", cfg.Subscription.QueueGroup),
		)
	}

	return consumer, nil
}

// ensureConsumer creates the consumer if it doesn't exist; if it already exists, it is reused.
func (c *Consumer) ensureConsumer() error {
	consumerConfig := &nats.ConsumerConfig{
		Durable:       c.consumerName,
		DeliverPolicy: mapDeliverPolicy(c.cfg.NATS.ConsumerRules.DeliverPolicy),
		AckPolicy:     nats.AckExplicitPolicy,
		AckWait:       c.ackWait,
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

	// First check if the consumer already exists to make this initialization idempotent.
	info, err := c.js.ConsumerInfo(c.streamName, c.consumerName)
	if err == nil && info != nil {
		c.logger.Info("Using existing JetStream consumer",
			zap.String("consumer", c.consumerName),
			zap.String("stream", c.streamName),
			zap.String("subject", c.subject),
		)
		return nil
	}
	if err != nil && !errors.Is(err, nats.ErrConsumerNotFound) {
		return fmt.Errorf("failed to fetch consumer info: %w", err)
	}

	// Consumer does not exist; create it.
	if _, err := c.js.AddConsumer(c.streamName, consumerConfig); err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	c.logger.Info("Created JetStream consumer",
		zap.String("consumer", c.consumerName),
		zap.String("stream", c.streamName),
		zap.String("subject", c.subject),
		zap.Duration("ack_wait", c.ackWait),
		zap.String("deliver_policy", c.cfg.NATS.ConsumerRules.DeliverPolicy),
		zap.String("replay_policy", c.cfg.NATS.ConsumerRules.ReplayPolicy),
	)

	return nil
}

// validateDLQ verifies whether DLQ routing should be enabled and, if so, whether
// the configured DLQ subject is bound to a JetStream stream. If validation fails,
// DLQ routing is disabled (by clearing c.dlqSubject) and a warning is logged,
// but the consumer is still allowed to start.
func (c *Consumer) validateDLQ() {
	if c == nil {
		return
	}

	// DLQ routing is only active in JetStream mode.
	if c.mode != "jetstream" {
		return
	}

	// If DLQ is not enabled in config, make sure we don't accidentally route to it.
	if !c.cfg.DLQ.Enabled {
		if strings.TrimSpace(c.dlqSubject) != "" {
			c.logger.Info("DLQ subject configured but dlq.enabled is false; DLQ routing disabled",
				zap.String("dlq_subject", c.dlqSubject),
			)
		}
		c.dlqSubject = ""
		return
	}

	subject := strings.TrimSpace(c.dlqSubject)
	if subject == "" {
		c.logger.Warn("DLQ enabled but dlq.subject is empty; DLQ routing disabled")
		return
	}

	if c.js == nil {
		c.logger.Warn("DLQ enabled but JetStream context is nil; DLQ routing disabled",
			zap.String("dlq_subject", subject),
		)
		c.dlqSubject = ""
		return
	}

	// Ensure the DLQ subject is actually bound to a JetStream stream. This avoids
	// the opaque `nats: no response from stream` error later when publishing.
	c.telemetry.RecordJSAPICall("dlq_validate_stream")
	streamName, err := c.js.StreamNameBySubject(subject)
	if err != nil || strings.TrimSpace(streamName) == "" {
		c.logger.Warn("DLQ subject not bound to any JetStream stream; DLQ routing disabled",
			zap.String("dlq_subject", subject),
			zap.Error(err),
		)
		c.dlqSubject = ""
		return
	}

	c.logger.Info("DLQ configuration validated",
		zap.String("dlq_subject", subject),
		zap.String("dlq_stream", streamName),
	)
}

// Start starts consuming messages
func (c *Consumer) Start(ctx context.Context) error {
	if c.mode == "core" {
		return c.startCore(ctx)
	}

	return c.startJetStream(ctx)
}

func (c *Consumer) startJetStream(ctx context.Context) error {
	// Create pull subscription
	sub, err := c.js.PullSubscribe(c.subject, c.consumerName, nats.Bind(c.streamName, c.consumerName))
	if err != nil {
		return fmt.Errorf("failed to create pull subscription: %w", err)
	}
	defer sub.Unsubscribe()

	c.logger.Info("Started consuming messages",
		zap.String("subject", c.subject),
		zap.String("consumer", c.consumerName),
		zap.String("stream", c.streamName),
	)

	c.logger.Info("Consumer pull configuration",
		zap.Int("batch_size", c.batchSize),
		zap.Duration("batch_timeout", c.batchTimeout),
		zap.Int("max_deliver", c.cfg.NATS.ConsumerRules.MaxDeliver),
		zap.Duration("ack_wait", c.ackWait),
		zap.String("deliver_policy", c.cfg.NATS.ConsumerRules.DeliverPolicy),
		zap.String("replay_policy", c.cfg.NATS.ConsumerRules.ReplayPolicy),
		zap.Int("backoff_steps", len(c.cfg.NATS.ConsumerRules.Backoff)),
	)

	statsCtx, statsCancel := context.WithCancel(ctx)
	defer statsCancel()
	go c.emitConsumerStats(statsCtx)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Stopping consumer", zap.Error(ctx.Err()))
			return ctx.Err()
		default:
		}

		// Fetch messages in batch
		msgs, err := sub.Fetch(c.batchSize, nats.MaxWait(c.batchTimeout))
		if err != nil {
			if errors.Is(err, nats.ErrTimeout) {
				// Timeout is expected when no messages are available
				continue
			}
			if errors.Is(err, nats.ErrNoResponders) {
				// JetStream API is currently unavailable (e.g., NATS just restarted or JetStream not ready).
				// Back off a bit to avoid log spam while allowing the system to recover.
				c.logger.Warn("JetStream not available, will retry",
					zap.Error(err),
					zap.String("stream", c.streamName),
					zap.String("consumer", c.consumerName),
				)
				time.Sleep(5 * time.Second)
				continue
			}
			c.logger.Error("Failed to fetch messages", zap.Error(err))
			time.Sleep(time.Second)
			continue
		}

		// Process each message
		// TODO: consider buffering messages to take advantage of Repository.InsertBatch for higher throughput.
		for _, msg := range msgs {
			start := time.Now()

			if err := c.processMessage(ctx, msg); err != nil {
				isPermanent := app.IsPermanent(err)
				elapsed := time.Since(start)

				c.logger.Error("Failed to process message",
					zap.String("subject", msg.Subject),
					zap.Error(err),
					zap.Bool("permanent", isPermanent),
				)

				result := obsmetrics.ResultFail
				if isPermanent {
					result = obsmetrics.ResultPermanentFail
				}
				c.telemetry.RecordMessageHandled(ctx, c.streamName, c.consumerName, result, elapsed)

				if isPermanent {
					c.consecutiveProcessErrors = 0
					// Poison/permanent message: route to DLQ if configured, then ACK
					if err := c.routeToDLQ(ctx, msg, err); err != nil {
						c.logger.Error("Failed to route permanent-error message to DLQ", zap.Error(err))
					}
					if ackErr := msg.Ack(); ackErr != nil {
						c.logger.Error("Failed to ACK permanent-error message", zap.Error(ackErr))
					}
					continue
				}

				// Transient error: increment error streak and apply simple backpressure if needed.
				if c.consecutiveProcessErrors < 0 {
					c.consecutiveProcessErrors = 0
				}
				c.consecutiveProcessErrors++
				if c.consecutiveProcessErrors >= 10 {
					// Apply a brief sleep to slow down consumption when the system
					// is failing many messages in a row (e.g. DB unavailable).
					backoff := time.Duration(c.consecutiveProcessErrors) * 100 * time.Millisecond
					if backoff > 5*time.Second {
						backoff = 5 * time.Second
					}
					c.logger.Warn("Applying backpressure due to consecutive processing errors",
						zap.Int("consecutive_errors", c.consecutiveProcessErrors),
						zap.Duration("sleep", backoff),
					)
					time.Sleep(backoff)
				}

				// Transient error: request redelivery with optional delay
				c.telemetry.RecordRetry(ctx, c.streamName, c.consumerName, obsmetrics.RetryReasonProcessorError)
				if nakErr := c.nakWithStrategy(msg); nakErr != nil {
					c.logger.Error("Failed to NAK message", zap.Error(nakErr))
				}
				continue
			}

			// Successful processing resets the error streak.
			if c.consecutiveProcessErrors > 0 {
				c.consecutiveProcessErrors = 0
			}

			// ACK the message
			if ackErr := msg.Ack(); ackErr != nil {
				c.logger.Error("Failed to ACK message", zap.Error(ackErr))
			} else {
				elapsed := time.Since(start)
				c.telemetry.RecordMessageHandled(ctx, c.streamName, c.consumerName, "ok", elapsed)
			}
		}
	}
}

func (c *Consumer) startCore(ctx context.Context) error {
	queueGroup := c.cfg.Subscription.QueueGroup
	if queueGroup == "" {
		queueGroup = c.consumerName
	}

	handler := func(msg *nats.Msg) {
		if err := c.processMessage(ctx, msg); err != nil {
			isPermanent := app.IsPermanent(err)
			c.logger.Error("Failed to process message (core mode)",
				zap.String("subject", msg.Subject),
				zap.Error(err),
				zap.Bool("permanent", isPermanent),
			)
		}
	}

	sub, err := c.conn.QueueSubscribe(c.subject, queueGroup, handler)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", c.subject, err)
	}
	if err := c.conn.Flush(); err != nil {
		return fmt.Errorf("failed to flush NATS connection: %w", err)
	}

	c.logger.Info("Started core NATS subscription",
		zap.String("subject", c.subject),
		zap.String("queue_group", queueGroup),
	)

	<-ctx.Done()
	c.logger.Info("Stopping core NATS consumer", zap.Error(ctx.Err()))

	if err := sub.Drain(); err != nil && !errors.Is(err, nats.ErrConnectionClosed) {
		return fmt.Errorf("failed to drain core subscription: %w", err)
	}

	return ctx.Err()
}

func (c *Consumer) emitConsumerStats(ctx context.Context) {
	ticker := time.NewTicker(c.monitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			info, err := c.js.ConsumerInfo(c.streamName, c.consumerName)
			if err != nil {
				c.logger.Warn("Failed to fetch consumer info", zap.Error(err))
				continue
			}

			c.logger.Debug("JetStream consumer metrics",
				zap.String("stream", c.streamName),
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

	if hist, err := meter.Int64Histogram("caatsm_nats_consumer_ack_pending"); err == nil {
		c.ackPending = hist
	}
	if hist, err := meter.Int64Histogram("caatsm_nats_consumer_redelivered"); err == nil {
		c.redelivered = hist
	}
	if hist, err := meter.Int64Histogram("caatsm_nats_consumer_pending"); err == nil {
		c.pending = hist
	}
	if hist, err := meter.Int64Histogram("caatsm_nats_consumer_delivered"); err == nil {
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

	// Export an explicit pending messages gauge for Prometheus-based lag /
	// backlog alerts.
	obsmetrics.RecordNATSConsumerPending(c.streamName, c.consumerName, info.NumPending)
}

// routeToDLQ publishes a copy of the failed message to the configured DLQ subject,
// including useful metadata for offline analysis. If DLQ is not configured or the
// consumer is not running in JetStream mode, this is a no-op.
func (c *Consumer) routeToDLQ(ctx context.Context, msg *nats.Msg, cause error) error {
	if c == nil || c.js == nil {
		return nil
	}
	if c.mode != "jetstream" {
		return nil
	}
	if strings.TrimSpace(c.dlqSubject) == "" {
		return nil
	}

	meta, _ := msg.Metadata()
	jsSeq := uint64(0)
	deliveries := uint64(0)
	if meta != nil {
		jsSeq = meta.Sequence.Stream
		deliveries = meta.NumDelivered
	}

	payload := map[string]interface{}{
		"transport_msg_id": msg.Header.Get("Nats-Msg-Id"),
		"subject":          msg.Subject,
		"stream":           c.streamName,
		"consumer":         c.consumerName,
		"nats_sequence":    jsSeq,
		"deliveries":       deliveries,
		"error":            fmt.Sprint(cause),
		"received_at":      time.Now().UTC(),
		"body":             string(msg.Data),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal dlq payload: %w", err)
	}

	if _, err := c.js.Publish(c.dlqSubject, data); err != nil {
		// nats.ErrNoResponders typically means that no JetStream stream is
		// configured to receive this subject, or JetStream is temporarily
		// unavailable. Surface this explicitly to make operational diagnosis
		// easier.
		if errors.Is(err, nats.ErrNoResponders) {
			c.telemetry.RecordDLQPublishFailure(ctx, c.streamName, c.consumerName)
			return fmt.Errorf("publish to dlq subject %s: no JetStream stream found for subject or JetStream unavailable: %w", c.dlqSubject, err)
		}
		c.telemetry.RecordDLQPublishFailure(ctx, c.streamName, c.consumerName)
		return fmt.Errorf("publish to dlq subject %s: %w", c.dlqSubject, err)
	}

	c.telemetry.RecordDLQMessage(ctx, c.streamName, c.consumerName)

	return nil
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

	// Attach structured logging context including stream/consumer and NATS metadata.
	jsSeq := uint64(0)
	if meta, metaErr := msg.Metadata(); metaErr == nil {
		jsSeq = meta.Sequence.Stream
		span.SetAttributes(
			attribute.Int64("nats.js.stream_seq", int64(meta.Sequence.Stream)),
			attribute.Int64("nats.js.consumer_seq", int64(meta.Sequence.Consumer)),
		)
	}

	msgLogger := obslogging.WithMessageContext(c.logger, obslogging.MessageFields{
		Service:        "caatsm-consumer",
		TransportMsgID: msgID,
		Stream:         c.streamName,
		Consumer:       c.consumerName,
		Subject:        msg.Subject,
		JSSequence:     jsSeq,
	})

	msgLogger.Debug("Processing message",
		zap.Int("data_size", len(msg.Data)),
		zap.String("msg_id_source", source),
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

	if c.mode == "core" {
		return uuid.NewString(), "generated", nil
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
