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
}

// consumerConfig holds normalized consumer configuration values.
type consumerConfig struct {
	subject         string
	consumerName    string
	mode            string
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

	// Initialize managers
	if consumer.config.mode == "jetstream" {
		consumer.consumerManager = NewConsumerManager(js, normCfg.streamName, normCfg.consumerName, normCfg.subject, logger)
		// Use StreamManager with full configuration
		streamSubjects := []string{normCfg.subject}
		if publisherSubject := strings.TrimSpace(cfg.Publisher.Topic); publisherSubject != "" {
			streamSubjects = append(streamSubjects, publisherSubject)
		}
		streamSubjects = dedupeSubjects(streamSubjects)
		consumer.streamManager = NewStreamManager(js, normCfg.streamName, streamSubjects, logger)

		// Update fetcher with managers now that they're initialized
		if fetcher, ok := consumer.fetcher.(*defaultMessageFetcher); ok {
			fetcher.consumerManager = consumer.consumerManager
			fetcher.streamManager = consumer.streamManager
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

	} else {
		logger.Info("Running consumer in core NATS mode",
			zap.String("subject", normCfg.subject),
			zap.String("queue_group", cfg.Subscription.QueueGroup),
		)
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
		mode:                     c.config.mode,
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

	return &consumerConfig{
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

// Start starts consuming messages.
func (c *Consumer) Start(ctx context.Context) error {
	if c.config.mode == "core" {
		return c.startCore(ctx)
	}

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

// startCore starts the Core NATS consumer loop.
func (c *Consumer) startCore(ctx context.Context) error {
	queueGroup := c.cfg.Subscription.QueueGroup
	if queueGroup == "" {
		queueGroup = c.config.consumerName
	}

	handler := func(msg *nats.Msg) {
		if err := c.batchProcessor.ProcessMessage(ctx, msg); err != nil {
			isPermanent := app.IsPermanent(err)
			c.logger.Error("Failed to process message (core mode)",
				zap.String("subject", msg.Subject),
				zap.Error(err),
				zap.Bool("permanent", isPermanent),
			)
		}
	}

	sub, err := c.conn.QueueSubscribe(c.config.subject, queueGroup, handler)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", c.config.subject, err)
	}
	if err := c.conn.Flush(); err != nil {
		return fmt.Errorf("failed to flush NATS connection: %w", err)
	}

	c.logger.Info("Started core NATS subscription",
		zap.String("subject", c.config.subject),
		zap.String("queue_group", queueGroup),
	)

	<-ctx.Done()
	c.logger.Info("Stopping core NATS consumer", zap.Error(ctx.Err()))

	if err := sub.Drain(); err != nil && !errors.Is(err, nats.ErrConnectionClosed) {
		return fmt.Errorf("failed to drain core subscription: %w", err)
	}

	return ctx.Err()
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

		// Process batch
		c.batchProcessor.ProcessBatch(ctx, msgs)
	}
}

// emitConsumerStats periodically emits basic consumer statistics.
func (c *Consumer) emitConsumerStats(ctx context.Context) {
	ticker := time.NewTicker(c.config.monitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			info, err := c.js.ConsumerInfo(c.config.streamName, c.config.consumerName)
			if err != nil {
				continue
			}
			// Record pending messages for monitoring
			obsmetrics.RecordNATSConsumerPending(c.config.streamName, c.config.consumerName, info.NumPending)
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
