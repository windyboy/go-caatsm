package nats

import (
	"caatsm/internal/config"
	"caatsm/internal/telegram"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

// MessageHandler processes raw telegram payloads delivered by the consumer.
type MessageHandler interface {
	Handle(ctx context.Context, raw []byte, msgID string) error
}

// Recorder captures consumer-level and monitor-level telemetry metrics.
type Recorder interface {
	MonitorRecorder
	RecordMessageHandled(ctx context.Context, stream, consumer, result string, elapsed time.Duration)
	RecordRetry(ctx context.Context, stream, consumer, reason string)
	RecordDLQMessage(ctx context.Context, stream, consumer string)
	RecordDLQPublishFailure(ctx context.Context, stream, consumer string)
}

type ackableMsg interface {
	Ack(opts ...nats.AckOpt) error
	Nak(opts ...nats.AckOpt) error
	NakWithDelay(delay time.Duration, opts ...nats.AckOpt) error
	Metadata() (*nats.MsgMetadata, error)
}

// Consumer handles NATS JetStream pull consumption, backoff, DLQ routing, and monitoring.
type Consumer struct {
	conn     *nats.Conn
	js       nats.JetStreamContext
	handler  MessageHandler
	cfg      *config.Config
	logger   *zap.Logger
	recorder Recorder

	monitor    *ConsumerMonitor
	dlqHandler DLQHandler

	streamName        string
	publishStreamName string
	dlqStreamName     string
	consumerName      string
	subject           string
	batchSize         int
	batchTimeout      time.Duration
	ackWait           time.Duration
	maxDeliver        int
	dlqPublishTimeout time.Duration
	backoff           []time.Duration

	consecutiveErrors int
}

// NewConsumer initializes a JetStream pull consumer and ensures/verifies its stream topology.
func NewConsumer(
	conn *nats.Conn,
	js nats.JetStreamContext,
	handler MessageHandler,
	cfg *config.Config,
	rec Recorder,
	logger *zap.Logger,
) (*Consumer, error) {
	if js == nil {
		return nil, fmt.Errorf("JetStream context is required")
	}
	if handler == nil {
		return nil, fmt.Errorf("message handler is required")
	}
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	if rec == nil {
		rec = noopRecorder{}
	}

	c := &Consumer{
		conn:              conn,
		js:                js,
		handler:           handler,
		cfg:               cfg,
		logger:            logger,
		recorder:          rec,
		streamName:        orDefault(cfg.NATS.Stream, "TELEGRAM"),
		publishStreamName: cfg.EffectivePublishStream(),
		dlqStreamName:     cfg.EffectiveDLQStream(),
		consumerName:      orDefault(cfg.NATS.Consumer, "telegram-consumer"),
		subject:           cfg.EffectiveSubscriptionTopic(),
		batchSize:         cfg.App.BatchSize,
		batchTimeout:      cfg.App.BatchTimeout,
		ackWait:           orDefaultDuration(cfg.NATS.ConsumerRules.AckWait, 30*time.Second),
		maxDeliver:        cfg.NATS.ConsumerRules.MaxDeliver,
		backoff:           cfg.NATS.ConsumerRules.Backoff,
	}

	if c.batchSize <= 0 {
		c.batchSize = 50
	}
	if c.batchTimeout <= 0 {
		c.batchTimeout = 2 * time.Second
	}

	c.monitor = NewConsumerMonitor(logger, cfg, js, c.streamName, c.consumerName, cfg.App.MonitorInterval, rec)
	if cfg.DLQ.Enabled && cfg.DLQ.Subject != "" {
		c.dlqHandler = &defaultDLQHandler{
			js:           js,
			dlqSubject:   cfg.DLQ.Subject,
			streamName:   c.streamName,
			consumerName: c.consumerName,
			logger:       logger,
			recorder:     rec,
		}
	}

	initCtx, initCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer initCancel()
	if err := c.ensureInfrastructure(initCtx); err != nil {
		return nil, err
	}

	return c, nil
}

// Start begins the pull consumption loop until ctx is cancelled.
func (c *Consumer) Start(ctx context.Context) error {
	c.logger.Info("Starting consumer",
		zap.String("stream", c.streamName),
		zap.String("consumer", c.consumerName),
		zap.String("subject", c.subject),
	)

	monitorCtx, cancelMonitor := context.WithCancel(ctx)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.monitor.Start(monitorCtx)
	}()
	defer func() {
		cancelMonitor()
		wg.Wait()
	}()

	sub, err := c.subscribe(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if sub != nil {
			_ = sub.Unsubscribe()
		}
	}()

	c.recorder.RecordJSAPICall("consumer_info")
	if info, err := c.js.ConsumerInfo(c.streamName, c.consumerName, nats.Context(ctx)); err == nil {
		c.syncMaxDeliver(info)
		c.monitor.RecordInitialPending(info.NumPending)
	}

	fetchErrorStreak := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		c.recorder.RecordJSAPICall("fetch")
		fetchCtx, fetchCancel := context.WithTimeout(ctx, c.batchTimeout)
		msgs, err := sub.Fetch(c.batchSize, nats.Context(fetchCtx))
		fetchCancel()

		if err != nil {
			if ctx.Err() != nil || errors.Is(err, context.Canceled) {
				return nil
			}
			if errors.Is(err, nats.ErrTimeout) || errors.Is(err, context.DeadlineExceeded) {
				fetchErrorStreak = 0
				continue
			}

			fetchErrorStreak++
			delay := fetchBackoff(fetchErrorStreak)
			c.logger.Warn("Fetch error, retrying with backoff",
				zap.Int("streak", fetchErrorStreak),
				zap.Duration("backoff", delay),
				zap.Error(err),
			)

			if !sleepWithContext(ctx, delay) {
				return nil
			}

			if recoveredSub := c.recoverSubscription(ctx, sub); recoveredSub != nil {
				sub = recoveredSub
			}
			continue
		}

		fetchErrorStreak = 0
		c.processBatch(ctx, msgs)
	}
}

func (c *Consumer) subscribe(ctx context.Context) (*nats.Subscription, error) {
	sub, err := c.js.PullSubscribe(c.subject, c.consumerName, nats.Bind(c.streamName, c.consumerName), nats.Context(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to bind pull subscription on %s/%s: %w", c.streamName, c.consumerName, err)
	}
	return sub, nil
}

func (c *Consumer) recoverSubscription(ctx context.Context, oldSub *nats.Subscription) *nats.Subscription {
	if ctx.Err() != nil {
		return nil
	}
	if err := c.ensureInfrastructure(ctx); err != nil {
		c.logger.Warn("JetStream infrastructure check failed during recovery", zap.Error(err))
		return nil
	}
	newSub, err := c.subscribe(ctx)
	if err != nil {
		c.logger.Warn("Failed to re-subscribe JetStream pull consumer", zap.Error(err))
		return nil
	}
	if oldSub != nil {
		_ = oldSub.Unsubscribe()
	}
	c.logger.Info("Recovered JetStream pull subscription",
		zap.String("stream", c.streamName),
		zap.String("consumer", c.consumerName),
	)
	return newSub
}

func fetchBackoff(streak int) time.Duration {
	if streak <= 1 {
		return 500 * time.Millisecond
	}
	delay := 500 * time.Millisecond
	for i := 1; i < streak; i++ {
		if delay >= 5*time.Second {
			return 5 * time.Second
		}
		delay *= 2
	}
	if delay > 5*time.Second {
		return 5 * time.Second
	}
	return delay
}

func sleepWithContext(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// Shutdown drains the NATS connection respecting the ctx deadline.
func (c *Consumer) Shutdown(ctx context.Context) error {
	if c.conn == nil {
		return nil
	}
	c.logger.Info("Draining NATS connection...")

	closed := c.conn.StatusChanged(nats.CLOSED)
	defer c.conn.RemoveStatusListener(closed)
	if err := c.conn.Drain(); err != nil {
		if errors.Is(err, nats.ErrConnectionClosed) {
			return nil
		}
		return err
	}

	select {
	case <-closed:
		return nil
	case <-ctx.Done():
		c.conn.Close()
		return ctx.Err()
	}
}

func (c *Consumer) processBatch(ctx context.Context, msgs []*nats.Msg) {
	for i, msg := range msgs {
		select {
		case <-ctx.Done():
			for _, rem := range msgs[i:] {
				if rem != nil {
					_ = rem.Nak()
				}
			}
			return
		default:
			c.monitor.TrackMessage(msg)
			c.processMsg(ctx, msg)
		}
	}
}

func (c *Consumer) inFlightContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(parent), c.cfg.EffectiveCloseTimeout())
}

func (c *Consumer) dlqTimeout() time.Duration {
	if c.dlqPublishTimeout > 0 {
		return c.dlqPublishTimeout
	}
	return DLQPublishTimeout
}

func (c *Consumer) processMsg(ctx context.Context, msg *nats.Msg) {
	start := time.Now()
	msgCtx, msgCancel := c.inFlightContext(ctx)
	defer msgCancel()

	if msg != nil && msg.Header != nil {
		msgCtx = otel.GetTextMapPropagator().Extract(msgCtx, natsHeaderCarrier(msg.Header))
	}

	msgCtx, span := otel.Tracer("caatsm/nats").Start(msgCtx, "Consumer.processMsg")
	defer span.End()

	msgID := c.resolveMsgID(msg)
	span.SetAttributes(
		attribute.String("messaging.system", "nats"),
		attribute.String("messaging.message_id", msgID),
		attribute.String("caatsm.stream", c.streamName),
	)

	publishID, metaErr := c.deriveInboundPublishID(msg)
	if metaErr != nil {
		c.logger.Error("Failed to read JetStream metadata for outbound deduplication key; retrying",
			zap.String("stream", c.streamName),
			zap.String("consumer", c.consumerName),
			zap.String("msg_id", msgID),
			zap.Error(metaErr),
		)
		resultLabel := c.handleFailure(ctx, msgCtx, msg, msg, msgID, fmt.Errorf("read JetStream metadata: %w", metaErr))
		span.RecordError(metaErr)
		span.SetStatus(codes.Error, metaErr.Error())
		c.recorder.RecordMessageHandled(msgCtx, c.streamName, c.consumerName, resultLabel, time.Since(start))
		return
	}
	msgCtx = telegram.WithPublishID(msgCtx, publishID)

	err := c.handler.Handle(msgCtx, msg.Data, msgID)
	if err != nil {
		resultLabel := c.handleFailure(ctx, msgCtx, msg, msg, msgID, err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		c.recorder.RecordMessageHandled(msgCtx, c.streamName, c.consumerName, resultLabel, time.Since(start))
		return
	}

	if c.consecutiveErrors > 0 {
		c.consecutiveErrors = 0
	}
	if ackErr := msg.Ack(); ackErr != nil {
		c.logger.Warn("Failed to ACK", zap.String("msg_id", msgID), zap.Error(ackErr))
	}
	c.recorder.RecordMessageHandled(msgCtx, c.streamName, c.consumerName, resultOK, time.Since(start))
}

func (c *Consumer) deriveInboundPublishID(msg *nats.Msg) (string, error) {
	if msg == nil {
		return "", fmt.Errorf("nil NATS message")
	}
	meta, err := msg.Metadata()
	if err != nil {
		return "", err
	}
	if meta == nil || meta.Sequence.Stream == 0 {
		return "", fmt.Errorf("missing JetStream stream sequence in message metadata")
	}
	return formatInboundPublishID(meta, c.streamName), nil
}

func formatInboundPublishID(meta *nats.MsgMetadata, fallbackStream string) string {
	stream := meta.Stream
	if stream == "" {
		stream = fallbackStream
	}
	ts := meta.Timestamp.UTC().UnixNano()
	if meta.Domain != "" {
		return fmt.Sprintf("js:%s:%s:%d:%d", meta.Domain, stream, meta.Sequence.Stream, ts)
	}
	return fmt.Sprintf("js:%s:%d:%d", stream, meta.Sequence.Stream, ts)
}

func deliveryMeta(msg ackableMsg) (streamSeq uint64, numDelivered uint64) {
	if msg == nil {
		return 0, 0
	}
	meta, err := msg.Metadata()
	if err != nil || meta == nil {
		return 0, 0
	}
	return meta.Sequence.Stream, meta.NumDelivered
}

// handleFailure separates NATS DLQ payload forwarding (*nats.Msg) from ACK/NAK lifecycle (ackableMsg)
// so unit tests can exercise all branches deterministically.
// shutdownCtx cancels backpressure sleeps on shutdown, while traceCtx carries trace context from which
// DLQ publishing derives its own independent bounded timeout budget.
func (c *Consumer) handleFailure(shutdownCtx, traceCtx context.Context, rawMsg *nats.Msg, acker ackableMsg, msgID string, err error) string {
	isPermanent := isPermanentError(err)
	streamSeq, numDelivered := deliveryMeta(acker)
	exhausted := !isPermanent && c.maxDeliver > 0 && numDelivered >= uint64(c.maxDeliver)

	c.logger.Error("Processing failed",
		zap.String("stream", c.streamName),
		zap.String("consumer", c.consumerName),
		zap.String("msg_id", msgID),
		zap.Uint64("stream_sequence", streamSeq),
		zap.Uint64("num_delivered", numDelivered),
		zap.Int("max_deliver", c.maxDeliver),
		zap.Bool("permanent", isPermanent),
		zap.Bool("max_deliver_exhausted", exhausted),
		zap.Error(err),
	)

	if isPermanent || (exhausted && c.dlqHandler != nil && rawMsg != nil) {
		dlqCause := err
		if exhausted {
			dlqCause = fmt.Errorf("max deliveries (%d) exhausted: %w", c.maxDeliver, err)
		}
		if c.dlqHandler != nil && rawMsg != nil {
			dlqPubCtx, dlqCancel := context.WithTimeout(context.WithoutCancel(traceCtx), c.dlqTimeout())
			dlqErr := c.dlqHandler.RouteToDLQ(dlqPubCtx, rawMsg, dlqCause)
			dlqCancel()
			if dlqErr != nil {
				// DLQ write failed: do NOT ACK to prevent silent message loss; NAK with backoff.
				if exhausted || (c.maxDeliver > 0 && numDelivered >= uint64(c.maxDeliver)) {
					c.logger.Error("Delivery attempts exhausted and DLQ publish failed; message retained in inbound stream pending manual recovery",
						zap.String("stream", c.streamName),
						zap.String("consumer", c.consumerName),
						zap.String("msg_id", msgID),
						zap.Uint64("stream_sequence", streamSeq),
						zap.Uint64("num_delivered", numDelivered),
						zap.Int("max_deliver", c.maxDeliver),
						zap.Error(dlqErr),
						zap.NamedError("cause", err),
					)
				}
				c.consecutiveErrors++
				c.recorder.RecordRetry(traceCtx, c.streamName, c.consumerName, retryReasonDLQError)
				c.applyBackpressure(shutdownCtx)
				if acker != nil {
					_ = c.nakWithBackoff(acker)
				}
				return resultFail
			}
		}

		c.consecutiveErrors = 0
		if acker != nil {
			if ackErr := acker.Ack(); ackErr != nil {
				c.logger.Warn("Failed to ACK terminal message", zap.String("msg_id", msgID), zap.Error(ackErr))
			}
		}
		return resultPermanentFail
	}

	if exhausted {
		c.logger.Error("Delivery attempts exhausted with DLQ unavailable; message retained in inbound stream pending manual recovery",
			zap.String("stream", c.streamName),
			zap.String("consumer", c.consumerName),
			zap.String("msg_id", msgID),
			zap.Uint64("stream_sequence", streamSeq),
			zap.Uint64("num_delivered", numDelivered),
			zap.Int("max_deliver", c.maxDeliver),
			zap.Error(err),
		)
	}

	c.consecutiveErrors++
	c.recorder.RecordRetry(traceCtx, c.streamName, c.consumerName, retryReasonProcessorError)
	c.applyBackpressure(shutdownCtx)
	if acker != nil {
		_ = c.nakWithBackoff(acker)
	}
	return resultFail
}

func (c *Consumer) syncMaxDeliver(info *nats.ConsumerInfo) {
	if info == nil || info.Config.MaxDeliver == 0 {
		return
	}
	brokerMax := info.Config.MaxDeliver
	if c.cfg != nil && c.cfg.NATS.ConsumerRules.MaxDeliver > 0 && brokerMax != c.cfg.NATS.ConsumerRules.MaxDeliver {
		c.logger.Warn("Broker consumer max_deliver differs from local config; using broker value",
			zap.String("stream", c.streamName),
			zap.String("consumer", c.consumerName),
			zap.Int("configured_max_deliver", c.cfg.NATS.ConsumerRules.MaxDeliver),
			zap.Int("broker_max_deliver", brokerMax),
		)
	}
	c.maxDeliver = brokerMax
}

func (c *Consumer) nakWithBackoff(msg ackableMsg) error {
	if len(c.backoff) == 0 {
		return msg.Nak()
	}
	meta, err := msg.Metadata()
	if err != nil || meta == nil {
		return msg.Nak()
	}

	attempt := int(meta.NumDelivered)
	index := attempt - 1
	if index >= len(c.backoff) {
		index = len(c.backoff) - 1
	} else if index < 0 {
		index = 0
	}
	return msg.NakWithDelay(c.backoff[index])
}

func (c *Consumer) applyBackpressure(ctx context.Context) {
	if c.consecutiveErrors < 10 {
		return
	}
	delay := time.Duration(c.consecutiveErrors) * 100 * time.Millisecond
	if delay > 5*time.Second {
		delay = 5 * time.Second
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-ctx.Done():
	}
}

func (c *Consumer) ensureInfrastructure(ctx context.Context) error {
	if !c.cfg.ShouldAutoProvision() {
		return c.verifyInfrastructure(ctx)
	}

	// 1. Inbound stream (WorkQueuePolicy)
	if err := c.ensureStream(ctx, c.streamName, []string{c.subject}, nats.WorkQueuePolicy); err != nil {
		return fmt.Errorf("ensure inbound stream: %w", err)
	}

	// 2. Outbound stream (LimitsPolicy) so multiple downstream consumers can subscribe independently
	if c.cfg.Publisher.Topic != "" && c.publishStreamName != c.streamName {
		if err := c.ensureStream(ctx, c.publishStreamName, []string{c.cfg.Publisher.Topic}, nats.LimitsPolicy); err != nil {
			return fmt.Errorf("ensure outbound stream: %w", err)
		}
	}

	// 3. Dead-letter stream (LimitsPolicy)
	if c.cfg.DLQ.Enabled && c.cfg.DLQ.Subject != "" && c.dlqStreamName != c.streamName {
		if err := c.ensureStream(ctx, c.dlqStreamName, []string{c.cfg.DLQ.Subject}, nats.LimitsPolicy); err != nil {
			return fmt.Errorf("ensure dlq stream: %w", err)
		}
	}

	// 4. Durable pull consumer on inbound stream
	consumerCfg, err := c.buildConsumerConfig()
	if err != nil {
		return fmt.Errorf("build consumer config: %w", err)
	}

	c.recorder.RecordJSAPICall("consumer_info")
	info, err := c.js.ConsumerInfo(c.streamName, c.consumerName, nats.Context(ctx))
	if err != nil {
		if !isJetStreamResourceNotFound(err) {
			return fmt.Errorf("inspect consumer: %w", err)
		}
		c.recorder.RecordJSAPICall("add_consumer")
		info, err = c.js.AddConsumer(c.streamName, consumerCfg, nats.Context(ctx))
		if err != nil {
			return fmt.Errorf("ensure consumer: %w", err)
		}
	}
	c.syncMaxDeliver(info)

	if err := ctx.Err(); err != nil {
		return err
	}
	if c.dlqHandler != nil {
		if err := c.dlqHandler.ValidateDLQ(ctx); err != nil {
			return fmt.Errorf("verify DLQ: %w", err)
		}
	}

	return nil
}

func (c *Consumer) verifyInfrastructure(ctx context.Context) error {
	c.recorder.RecordJSAPICall("stream_info")
	if _, err := c.js.StreamInfo(c.streamName, nats.Context(ctx)); err != nil {
		return fmt.Errorf("verify inbound stream %s: %w", c.streamName, err)
	}

	if c.cfg.Publisher.Topic != "" && c.publishStreamName != c.streamName {
		c.recorder.RecordJSAPICall("stream_info")
		if _, err := c.js.StreamInfo(c.publishStreamName, nats.Context(ctx)); err != nil {
			return fmt.Errorf("verify outbound stream %s: %w", c.publishStreamName, err)
		}
	}

	c.recorder.RecordJSAPICall("consumer_info")
	info, err := c.js.ConsumerInfo(c.streamName, c.consumerName, nats.Context(ctx))
	if err != nil {
		return fmt.Errorf("verify consumer %s on stream %s: %w", c.consumerName, c.streamName, err)
	}
	c.syncMaxDeliver(info)

	if err := ctx.Err(); err != nil {
		return err
	}
	if c.dlqHandler != nil {
		if err := c.dlqHandler.ValidateDLQ(ctx); err != nil {
			return fmt.Errorf("verify DLQ: %w", err)
		}
	}

	return nil
}

func (c *Consumer) ensureStream(ctx context.Context, name string, subjects []string, retention nats.RetentionPolicy) error {
	c.recorder.RecordJSAPICall("stream_info")
	if _, err := c.js.StreamInfo(name, nats.Context(ctx)); err == nil {
		return nil
	} else if !isJetStreamResourceNotFound(err) {
		return fmt.Errorf("inspect stream %s: %w", name, err)
	}

	streamCfg := &nats.StreamConfig{
		Name:      name,
		Subjects:  dedupeSubjects(subjects),
		Retention: retention,
		MaxMsgs:   c.cfg.NATS.StreamLimits.MaxMsgs,
		MaxBytes:  c.cfg.NATS.StreamLimits.MaxBytes,
		MaxAge:    c.cfg.NATS.StreamLimits.MaxAge,
		Replicas:  c.cfg.NATS.StreamLimits.Replicas,
		Storage:   nats.FileStorage,
	}
	if strings.EqualFold(strings.TrimSpace(c.cfg.NATS.StreamLimits.Discard), "new") {
		streamCfg.Discard = nats.DiscardNew
	}
	if strings.EqualFold(strings.TrimSpace(c.cfg.NATS.StreamLimits.Storage), "memory") {
		streamCfg.Storage = nats.MemoryStorage
	}

	c.recorder.RecordJSAPICall("add_stream")
	if _, err := c.js.AddStream(streamCfg, nats.Context(ctx)); err != nil {
		return fmt.Errorf("add stream %s: %w", name, err)
	}

	return nil
}

func (c *Consumer) buildConsumerConfig() (*nats.ConsumerConfig, error) {
	deliverPolicy := mapDeliverPolicy(c.cfg.NATS.ConsumerRules.DeliverPolicy)
	consumerCfg := &nats.ConsumerConfig{
		Durable:       c.consumerName,
		FilterSubject: c.subject,
		AckPolicy:     nats.AckExplicitPolicy,
		AckWait:       c.ackWait,
		MaxDeliver:    c.cfg.NATS.ConsumerRules.MaxDeliver,
		MaxAckPending: c.cfg.NATS.ConsumerRules.MaxAckPending,
		DeliverPolicy: deliverPolicy,
		ReplayPolicy:  mapReplayPolicy(c.cfg.NATS.ConsumerRules.ReplayPolicy),
	}

	switch deliverPolicy {
	case nats.DeliverByStartSequencePolicy:
		consumerCfg.OptStartSeq = c.cfg.NATS.ConsumerRules.StartSequence
	case nats.DeliverByStartTimePolicy:
		if c.cfg.NATS.ConsumerRules.StartTime != "" {
			parsedTime, err := time.Parse(time.RFC3339, c.cfg.NATS.ConsumerRules.StartTime)
			if err != nil {
				return nil, fmt.Errorf("invalid start_time: %w", err)
			}
			consumerCfg.OptStartTime = &parsedTime
		}
	}

	return consumerCfg, nil
}

func (c *Consumer) resolveMsgID(msg *nats.Msg) string {
	if msg == nil {
		return "unknown"
	}
	if msg.Header != nil {
		if id := msg.Header.Get("Nats-Msg-Id"); id != "" {
			return id
		}
	}
	if meta, err := msg.Metadata(); err == nil && meta != nil {
		return fmt.Sprintf("js-%d", meta.Sequence.Stream)
	}
	return "unknown"
}

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
