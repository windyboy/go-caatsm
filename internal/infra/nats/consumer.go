package nats

import (
	"caatsm/internal/app"
	"caatsm/internal/infra/config"
	obsmetrics "caatsm/internal/infra/metrics"
	"caatsm/internal/infra/telemetry"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// Consumer handles NATS JetStream message consumption with clean separation of concerns
type Consumer struct {
	// Core dependencies
	conn      *nats.Conn
	js        nats.JetStreamContext
	processor *app.MessageProcessor
	cfg       *config.Config
	logger    *zap.Logger
	telemetry telemetry.Recorder

	// Configuration
	config consumerConfig

	// Collaborators (injected for testability)
	fetcher        MessageFetcher
	batchProcessor MessageProcessor
	dlqHandler     DLQHandler

	// Resource managers
	consumerManager *ConsumerManager
	streamManager   *StreamManager

	// State
	consecutiveProcessErrors int

	// Message tracking for health monitoring
	lastMessageTime     time.Time
	lastMessageSequence uint64
	messageGapMutex     sync.RWMutex
}

// consumerConfig holds normalized consumer configuration values.
type consumerConfig struct {
	subject         string
	consumerName    string
	streamName      string
	dlqSubject      string
	ackWait         time.Duration
	batchSize       int
	batchTimeout    time.Duration
	monitorInterval time.Duration
}

// ProvideConsumer creates a NATS consumer with clean architecture.
func ProvideConsumer(
	conn *nats.Conn,
	js nats.JetStreamContext,
	processor *app.MessageProcessor,
	cfg *config.Config,
	rec telemetry.Recorder,
	logger *zap.Logger,
) (*Consumer, error) {
	normCfg := normalizeConsumerConfig(cfg)

	consumer := &Consumer{
		conn:      conn,
		js:        js,
		processor: processor,
		cfg:       cfg,
		logger:    logger,
		telemetry: rec,
		config:    *normCfg, // dereference the pointer
	}
	consumer.initCollaborators()

	// Initialize the pending messages metric early (set to 0) so it appears in Prometheus
	// even before the consumer starts. This ensures the metric is always visible.
	logger.Info("Initializing NATS consumer pending messages metric",
		zap.String("stream", normCfg.streamName),
		zap.String("consumer", normCfg.consumerName),
		zap.Uint64("pending", 0),
		zap.Bool("js_available", js != nil),
	)
	obsmetrics.RecordNATSConsumerPending(normCfg.streamName, normCfg.consumerName, 0)

	// Initialize managers
	consumer.consumerManager = NewConsumerManager(js, normCfg.streamName, normCfg.consumerName, normCfg.subject, logger)
	// Use StreamManager with full configuration
	streamSubjects := []string{normCfg.subject}
	if publisherSubject := strings.TrimSpace(cfg.Publisher.Topic); publisherSubject != "" {
		streamSubjects = append(streamSubjects, publisherSubject)
	}
	// Add DLQ subject to stream if DLQ is enabled
	if normCfg.dlqSubject != "" {
		streamSubjects = append(streamSubjects, normCfg.dlqSubject)
	}
	streamSubjects = dedupeSubjects(streamSubjects)
	consumer.streamManager = NewStreamManager(js, normCfg.streamName, streamSubjects, logger)

	// Update fetcher with managers now that they're initialized
	if fetcher, ok := consumer.fetcher.(*defaultMessageFetcher); ok {
		fetcher.consumerManager = consumer.consumerManager
		fetcher.streamManager = consumer.streamManager
	}

	// Ensure stream exists before creating consumer
	streamCfg := &StreamConfig{
		MaxMsgs:  cfg.NATS.StreamLimits.MaxMsgs,
		MaxBytes: cfg.NATS.StreamLimits.MaxBytes,
		MaxAge:   cfg.NATS.StreamLimits.MaxAge,
		Discard:  cfg.NATS.StreamLimits.Discard,
		Storage:  cfg.NATS.StreamLimits.Storage,
		Replicas: cfg.NATS.StreamLimits.Replicas,
	}
	if err := consumer.streamManager.EnsureStream(streamCfg); err != nil {
		return nil, fmt.Errorf("failed to ensure stream: %w", err)
	}

	// Create consumer if it doesn't exist
	consumerConfig := consumer.buildConsumerConfig()
	if err := consumer.consumerManager.EnsureConsumer(consumerConfig); err != nil {
		return nil, fmt.Errorf("failed to ensure consumer: %w", err)
	}
	// Validate DLQ configuration early so misconfiguration is visible at startup
	// rather than only when the first poison message appears.
	if err := consumer.validateDLQ(); err != nil {
		return nil, fmt.Errorf("DLQ validation failed: %w", err)
	}

	return consumer, nil
}

// initCollaborators initializes the collaborator components
func (c *Consumer) initCollaborators() {
	c.fetcher = &defaultMessageFetcher{
		batchSize:       c.config.batchSize,
		batchTimeout:    c.config.batchTimeout,
		logger:          c.logger,
		conn:            c.conn,
		js:              c.js,
		consumerManager: c.consumerManager,
		streamManager:   c.streamManager,
		config:          &c.config,
		cfg:             c.cfg,
	}

	// Initialize DLQ handler first if needed, so batch processor can reference it
	if c.config.dlqSubject != "" {
		c.dlqHandler = &defaultDLQHandler{
			js:           c.js,
			dlqSubject:   c.config.dlqSubject,
			streamName:   c.config.streamName,
			consumerName: c.config.consumerName,
			logger:       c.logger,
			telemetry:    c.telemetry,
		}
	}

	c.batchProcessor = &defaultBatchProcessor{
		processor:                c.processor,
		dlqHandler:               c.dlqHandler,
		logger:                   c.logger,
		telemetry:                c.telemetry,
		streamName:               c.config.streamName,
		consumerName:             c.config.consumerName,
		backoff:                  c.cfg.NATS.ConsumerRules.Backoff,
		consecutiveProcessErrors: &c.consecutiveProcessErrors,
	}
}

// normalizeConsumerConfig extracts and normalizes consumer configuration from the application config.
// This function can be unit-tested without requiring a JetStream context.
func normalizeConsumerConfig(cfg *config.Config) *consumerConfig {
	subject := cfg.EffectiveSubscriptionTopic()

	consumerName := cfg.NATS.Consumer
	if consumerName == "" {
		consumerName = "telegram-consumer"
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

	return &consumerConfig{
		subject:         subject,
		consumerName:    consumerName,
		streamName:      streamName,
		dlqSubject:      dlqSubject,
		ackWait:         ackWait,
		batchSize:       batchSize,
		batchTimeout:    batchTimeout,
		monitorInterval: monitorInterval,
	}
}

// buildConsumerConfig builds the NATS consumer configuration
func (c *Consumer) buildConsumerConfig() *nats.ConsumerConfig {
	return &nats.ConsumerConfig{
		Durable:       c.config.consumerName,
		DeliverPolicy: mapDeliverPolicy(c.cfg.NATS.ConsumerRules.DeliverPolicy),
		AckPolicy:     nats.AckExplicitPolicy,
		AckWait:       c.config.ackWait,
		ReplayPolicy:  mapReplayPolicy(c.cfg.NATS.ConsumerRules.ReplayPolicy),
		MaxDeliver:    c.cfg.NATS.ConsumerRules.MaxDeliver,
		MaxAckPending: c.cfg.NATS.ConsumerRules.MaxAckPending,
		FilterSubject: c.config.subject,
		BackOff:       c.cfg.NATS.ConsumerRules.Backoff,
	}
}

// Start starts consuming messages from JetStream.
func (c *Consumer) Start(ctx context.Context) error {
	return c.startJetStream(ctx)
}

// RouteToDLQ implements DLQHandler interface
func (c *Consumer) RouteToDLQ(ctx context.Context, msg *nats.Msg, cause error) error {
	if c.dlqHandler != nil {
		return c.dlqHandler.RouteToDLQ(ctx, msg, cause)
	}
	return nil
}

// ValidateDLQ implements DLQHandler interface
func (c *Consumer) ValidateDLQ() error {
	if c.dlqHandler != nil {
		return c.dlqHandler.ValidateDLQ()
	}
	return nil
}

// validateDLQ is a helper for internal use (lowercase)
func (c *Consumer) validateDLQ() error {
	return c.ValidateDLQ()
}

// createPullSubscription creates a pull subscription
func (c *Consumer) createPullSubscription() (*nats.Subscription, error) {
	return c.consumerManager.CreatePullSubscription()
}

// startJetStream starts the JetStream consumer loop.
func (c *Consumer) startJetStream(ctx context.Context) error {
	// Create pull subscription
	sub, err := c.createPullSubscription()
	if err != nil {
		return err
	}

	// Use a closure that always cleans up the current subscription.
	// When subscription is replaced in handleFetchError, this will clean up
	// whatever currentSub points to at shutdown time.
	var currentSub = sub
	cleanupSubscriber := func() {
		if currentSub != nil {
			if err := currentSub.Unsubscribe(); err != nil {
				c.logger.Error("Failed to unsubscribe subscription", zap.Error(err))
			}
			currentSub = nil
		}
	}
	defer cleanupSubscriber()

	c.logger.Info("Started consuming messages",
		zap.String("subject", c.config.subject),
		zap.String("consumer", c.config.consumerName),
		zap.String("stream", c.config.streamName),
	)

	// Record initial pending messages metric immediately
	// This ensures the metric appears in Prometheus right away
	if info, err := c.js.ConsumerInfo(c.config.streamName, c.config.consumerName); err == nil {
		c.logger.Info("Recording initial NATS consumer pending messages metric",
			zap.String("stream", c.config.streamName),
			zap.String("consumer", c.config.consumerName),
			zap.Uint64("pending", info.NumPending),
		)
		obsmetrics.RecordNATSConsumerPending(c.config.streamName, c.config.consumerName, info.NumPending)
	} else {
		c.logger.Warn("Failed to fetch initial consumer info for pending messages metric",
			zap.String("stream", c.config.streamName),
			zap.String("consumer", c.config.consumerName),
			zap.Error(err),
		)
	}

	statsCtx, statsCancel := context.WithCancel(ctx)
	defer statsCancel()
	go c.emitConsumerStats(statsCtx)

	var fetchErrorStreak int

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Stopping consumer", zap.Error(ctx.Err()))
			return ctx.Err()
		default:
		}

		// Fetch messages in batch
		msgs, err := c.fetcher.FetchBatch(ctx, currentSub)
		if err != nil {
			// If context was cancelled, return immediately
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				c.logger.Info("Stopping consumer due to context cancellation", zap.Error(err))
				return err
			}
			shouldContinue, handleErr := c.fetcher.HandleFetchError(ctx, err, &currentSub, &fetchErrorStreak)
			if !shouldContinue {
				return handleErr
			}
			continue
		}

		// Successful fetch -> reset error streak.
		if fetchErrorStreak > 0 {
			fetchErrorStreak = 0
		}

		// Update message tracking for health monitoring (track each message)
		for _, msg := range msgs {
			c.updateMessageTracking(msg)
		}

		// Process batch
		c.batchProcessor.ProcessBatch(ctx, msgs)
	}
}

// emitConsumerStats periodically emits basic consumer statistics.
func (c *Consumer) emitConsumerStats(ctx context.Context) {
	ticker := time.NewTicker(c.config.monitorInterval)
	defer ticker.Stop()

	// Record initial metric (0) to ensure it appears in Prometheus even before first tick
	c.logger.Info("Starting NATS consumer stats emission goroutine, recording initial pending metric",
		zap.String("stream", c.config.streamName),
		zap.String("consumer", c.config.consumerName),
		zap.Duration("interval", c.config.monitorInterval),
	)
	obsmetrics.RecordNATSConsumerPending(c.config.streamName, c.config.consumerName, 0)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			info, err := c.js.ConsumerInfo(c.config.streamName, c.config.consumerName)
			if err != nil {
				c.logger.Warn("Failed to fetch consumer info for pending messages metric",
					zap.String("stream", c.config.streamName),
					zap.String("consumer", c.config.consumerName),
					zap.Error(err),
				)
				continue
			}
			// Record pending messages for monitoring
			c.logger.Debug("Recording NATS consumer pending messages metric",
				zap.String("stream", c.config.streamName),
				zap.String("consumer", c.config.consumerName),
				zap.Uint64("pending", info.NumPending),
			)
			obsmetrics.RecordNATSConsumerPending(c.config.streamName, c.config.consumerName, info.NumPending)

			// Record AFTN health metrics
			gapSeconds := c.getMessageGapSeconds()
			healthy := c.isSerialReaderHealthy()

			obsmetrics.RecordMessageGap(c.config.streamName, c.config.consumerName, gapSeconds)
			obsmetrics.RecordSerialReaderHealth(c.config.streamName, c.config.consumerName, healthy)

			if !healthy {
				c.logger.Warn("Serial reader appears stalled - no messages received recently",
					zap.String("stream", c.config.streamName),
					zap.String("consumer", c.config.consumerName),
					zap.Float64("gap_seconds", gapSeconds),
					zap.Duration("threshold", c.cfg.AFTN.MessageGapThreshold),
				)
			}
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
		timeout = 2 * time.Second // Reduced from 10s for faster shutdown
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

// updateMessageTracking updates the last message time and sequence number for health monitoring.
// This should be called for every message received to track message flow and detect gaps.
func (c *Consumer) updateMessageTracking(msg *nats.Msg) {
	if msg == nil {
		return
	}

	c.messageGapMutex.Lock()
	defer c.messageGapMutex.Unlock()

	now := time.Now()
	c.lastMessageTime = now

	// Extract sequence number from message metadata
	if meta, err := msg.Metadata(); err == nil {
		currentSeq := meta.Sequence.Stream

		// Detect sequence gaps if we have a previous sequence
		if c.lastMessageSequence > 0 && c.cfg.AFTN.EnableSequenceGapDetection {
			if currentSeq > c.lastMessageSequence+1 {
				gapSize := currentSeq - c.lastMessageSequence - 1
				c.logger.Warn("Message sequence gap detected",
					zap.String("stream", c.config.streamName),
					zap.String("consumer", c.config.consumerName),
					zap.Uint64("last_sequence", c.lastMessageSequence),
					zap.Uint64("current_sequence", currentSeq),
					zap.Uint64("gap_size", gapSize),
				)
				obsmetrics.RecordSequenceGap(c.config.streamName, c.config.consumerName, gapSize)
			}
		}

		c.lastMessageSequence = currentSeq
	}
}

// getMessageGapSeconds returns the number of seconds since the last message was received.
// Returns 0 if no message has been received yet.
func (c *Consumer) getMessageGapSeconds() float64 {
	c.messageGapMutex.RLock()
	defer c.messageGapMutex.RUnlock()

	if c.lastMessageTime.IsZero() {
		return 0
	}

	return time.Since(c.lastMessageTime).Seconds()
}

// isSerialReaderHealthy returns true if messages are being received within the threshold.
// Returns false if the gap exceeds the configured message gap threshold.
func (c *Consumer) isSerialReaderHealthy() bool {
	c.messageGapMutex.RLock()
	defer c.messageGapMutex.RUnlock()

	// If we haven't received any messages yet, consider it healthy (initial state)
	if c.lastMessageTime.IsZero() {
		return true
	}

	gap := time.Since(c.lastMessageTime)
	return gap < c.cfg.AFTN.MessageGapThreshold
}
