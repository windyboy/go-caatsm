package nats

import (
	"caatsm/internal/app"
	"caatsm/internal/infra/config"
	"caatsm/internal/infra/telemetry"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

// MessageFetcher defines the interface for fetching messages from NATS
type MessageFetcher interface {
	FetchBatch(ctx context.Context, sub *nats.Subscription) ([]*nats.Msg, error)
	HandleFetchError(ctx context.Context, err error, sub **nats.Subscription, fetchErrorStreak *int) (bool, error)
}

// MessageProcessor defines the interface for processing message batches
type MessageProcessor interface {
	ProcessBatch(ctx context.Context, msgs []*nats.Msg)
}

// DLQHandler defines the interface for dead letter queue operations
type DLQHandler interface {
	RouteToDLQ(ctx context.Context, msg *nats.Msg, cause error) error
	ValidateDLQ() error
}

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
	errorHandler   *ErrorHandler

	// Resource managers
	consumerManager *ConsumerManager
	streamManager   *StreamManager

	// Advisory DLQ handler for messages exhausting MaxDeliver
	advisoryDLQHandler *AdvisoryDLQHandler

	// Metrics
	meter       metric.Meter
	ackPending  metric.Int64Histogram
	redelivered metric.Int64Histogram
	pending     metric.Int64Histogram
	delivered   metric.Int64Histogram

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

// defaultMessageFetcher implements MessageFetcher interface
type defaultMessageFetcher struct {
	batchSize       int
	batchTimeout    time.Duration
	logger          *zap.Logger
	conn            *nats.Conn
	js              nats.JetStreamContext
	consumerManager *ConsumerManager
	streamManager   *StreamManager
	config          *consumerConfig
	cfg             *config.Config
}

func (f *defaultMessageFetcher) FetchBatch(ctx context.Context, sub *nats.Subscription) ([]*nats.Msg, error) {
	return f.fetchBatch(ctx, sub)
}

// fetchBatch fetches a batch of messages from the subscription with context awareness
func (f *defaultMessageFetcher) fetchBatch(ctx context.Context, sub *nats.Subscription) ([]*nats.Msg, error) {
	// Check context before fetching
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Use a shorter timeout for better responsiveness to cancellation
	timeout := f.batchTimeout
	if timeout > 500*time.Millisecond {
		timeout = 500 * time.Millisecond
	}

	return sub.Fetch(f.batchSize, nats.MaxWait(timeout))
}

func (f *defaultMessageFetcher) HandleFetchError(ctx context.Context, err error, sub **nats.Subscription, fetchErrorStreak *int) (bool, error) {
	// Check context cancellation first
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		f.logger.Info("Fetch error due to context cancellation", zap.Error(err))
		return false, err
	}

	// Timeout errors are expected when no messages are available - not an error condition
	if errors.Is(err, nats.ErrTimeout) {
		return true, nil
	}

	// Check connection health before proceeding
	if f.conn != nil {
		status := f.conn.Status()
		if status != nats.CONNECTED {
			f.logger.Warn("NATS connection not in CONNECTED state",
				zap.String("status", status.String()),
				zap.Error(err),
			)
			// Connection is down - this is a transient error, apply backoff
			*fetchErrorStreak++
			backoff := f.calculateExponentialBackoff(*fetchErrorStreak)
			f.logger.Warn("Connection unhealthy, applying backoff before retry",
				zap.String("status", status.String()),
				zap.Int("error_streak", *fetchErrorStreak),
				zap.Duration("backoff", backoff),
			)
			if !sleepWithContext(ctx, backoff) {
				return false, ctx.Err()
			}
			// Check if connection recovered after backoff
			if f.conn.Status() == nats.CONNECTED {
				*fetchErrorStreak = 0
				return true, nil
			}
			// Still not connected - continue with error handling
		}
	}

	// Check for connection closed errors
	if errors.Is(err, nats.ErrConnectionClosed) {
		f.logger.Error("NATS connection closed",
			zap.Error(err),
			zap.String("stream", f.config.streamName),
			zap.String("consumer", f.config.consumerName),
		)
		// Connection closed is fatal - cannot recover subscription
		if *sub != nil {
			(*sub).Unsubscribe()
			*sub = nil
		}
		return false, fmt.Errorf("connection closed: %w", err)
	}

	// JetStream API unavailable (e.g., NATS restarted or JetStream not ready)
	if errors.Is(err, nats.ErrNoResponders) {
		*fetchErrorStreak++
		backoff := f.calculateExponentialBackoff(*fetchErrorStreak)
		backoff = min(backoff, 30*time.Second)
		f.logger.Warn("JetStream not available, will retry with backoff",
			zap.Error(err),
			zap.String("stream", f.config.streamName),
			zap.String("consumer", f.config.consumerName),
			zap.Int("error_streak", *fetchErrorStreak),
			zap.Duration("backoff", backoff),
		)
		if !sleepWithContext(ctx, backoff) {
			return false, ctx.Err()
		}
		return true, nil
	}

	// Check for JetStream resource not found errors
	if isJetStreamResourceNotFound(err) {
		if isDevLikeEnv() && shouldBootstrapStream() {
			f.logger.Warn("JetStream consumer or stream missing; attempting to recreate",
				zap.Error(err),
				zap.String("stream", f.config.streamName),
				zap.String("consumer", f.config.consumerName),
			)
			// Attempt to recover resources and recreate subscription
			if f.consumerManager == nil || f.streamManager == nil {
				return false, fmt.Errorf("cannot recover: consumer/stream manager not available: %w", err)
			}
			consumerConfig := f.buildConsumerConfig()
			if recErr := f.consumerManager.RecoverResources(f.streamManager, consumerConfig); recErr != nil {
				return false, fmt.Errorf("failed to recover JetStream resources: %w", recErr)
			}
			// Unsubscribe old subscription before creating new one
			if *sub != nil {
				(*sub).Unsubscribe()
			}
			// Create new subscription
			newSub, subErr := f.consumerManager.CreatePullSubscription()
			if subErr != nil {
				return false, fmt.Errorf("failed to create pull subscription after recovery: %w", subErr)
			}
			*sub = newSub
			*fetchErrorStreak = 0
			f.logger.Info("Successfully recovered subscription after resource recreation")
			return true, nil
		}

		// Production: treat as configuration/operational error - fatal
		f.logger.Error("JetStream consumer or stream missing; not auto-recreating in this environment",
			zap.Error(err),
			zap.String("stream", f.config.streamName),
			zap.String("consumer", f.config.consumerName),
		)
		if *sub != nil {
			(*sub).Unsubscribe()
			*sub = nil
		}
		return false, fmt.Errorf("JetStream resource not found: %w", err)
	}

	// Check for network/temporary errors
	if f.isTemporaryError(err) {
		*fetchErrorStreak++
		backoff := f.calculateExponentialBackoff(*fetchErrorStreak)
		f.logger.Warn("Temporary network error, applying backoff",
			zap.Error(err),
			zap.Int("error_streak", *fetchErrorStreak),
			zap.Duration("backoff", backoff),
		)
		if !sleepWithContext(ctx, backoff) {
			return false, ctx.Err()
		}
		// Verify subscription is still valid before returning success
		if *sub != nil && f.conn != nil && f.conn.Status() == nats.CONNECTED {
			return true, nil
		}
		// Subscription or connection invalid - attempt recovery
		return f.attemptSubscriptionRecovery(ctx, sub, fetchErrorStreak)
	}

	// Generic error path with exponential backoff
	*fetchErrorStreak++
	backoff := f.calculateExponentialBackoff(*fetchErrorStreak)
	f.logger.Error("Failed to fetch messages; backing off",
		zap.Error(err),
		zap.Int("error_streak", *fetchErrorStreak),
		zap.Duration("backoff", backoff),
	)
	if !sleepWithContext(ctx, backoff) {
		return false, ctx.Err()
	}

	// Verify subscription and connection health before returning success
	if *sub == nil || (f.conn != nil && f.conn.Status() != nats.CONNECTED) {
		return f.attemptSubscriptionRecovery(ctx, sub, fetchErrorStreak)
	}

	return true, nil
}

// calculateExponentialBackoff calculates exponential backoff duration with a cap
func (f *defaultMessageFetcher) calculateExponentialBackoff(streak int) time.Duration {
	if streak <= 0 {
		return 0
	}
	// Exponential backoff: 2^(streak-1) seconds, capped at 30 seconds
	backoff := time.Duration(1<<uint(min(streak-1, 5))) * time.Second
	return min(backoff, 30*time.Second)
}

// isTemporaryError checks if an error is a temporary/network error that might recover
func (f *defaultMessageFetcher) isTemporaryError(err error) bool {
	if err == nil {
		return false
	}
	// Check for typed temporary errors first
	var tempErr interface{ Temporary() bool }
	if errors.As(err, &tempErr) && tempErr.Temporary() {
		return true
	}
	// Fall back to string matching for external errors
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "temporary") ||
		strings.Contains(errStr, "network") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "broken pipe")
}

// attemptSubscriptionRecovery attempts to recover a subscription after errors
func (f *defaultMessageFetcher) attemptSubscriptionRecovery(ctx context.Context, sub **nats.Subscription, fetchErrorStreak *int) (bool, error) {
	if f.consumerManager == nil {
		f.logger.Error("Cannot recover subscription: consumer manager not available")
		return false, fmt.Errorf("consumer manager not available for recovery")
	}

	// Check connection health first
	if f.conn != nil && f.conn.Status() != nats.CONNECTED {
		f.logger.Warn("Connection not healthy, cannot recover subscription",
			zap.String("status", f.conn.Status().String()),
		)
		// Connection issue - return true to retry after backoff
		return true, nil
	}

	// Unsubscribe old subscription if it exists
	if *sub != nil {
		(*sub).Unsubscribe()
		*sub = nil
	}

	// Attempt to recreate subscription
	newSub, err := f.consumerManager.CreatePullSubscriptionWithRecovery(f.streamManager, f.buildConsumerConfig())
	if err != nil {
		f.logger.Error("Failed to recover subscription",
			zap.Error(err),
			zap.String("stream", f.config.streamName),
			zap.String("consumer", f.config.consumerName),
		)
		return false, fmt.Errorf("failed to recover subscription: %w", err)
	}

	*sub = newSub
	*fetchErrorStreak = 0
	f.logger.Info("Successfully recovered subscription")
	return true, nil
}

// buildConsumerConfig builds the NATS consumer configuration
func (f *defaultMessageFetcher) buildConsumerConfig() *nats.ConsumerConfig {
	if f.cfg == nil {
		return nil
	}
	return &nats.ConsumerConfig{
		Durable:       f.config.consumerName,
		DeliverPolicy: mapDeliverPolicy(f.cfg.NATS.ConsumerRules.DeliverPolicy),
		AckPolicy:     nats.AckExplicitPolicy,
		AckWait:       f.config.ackWait,
		ReplayPolicy:  mapReplayPolicy(f.cfg.NATS.ConsumerRules.ReplayPolicy),
		MaxDeliver:    f.cfg.NATS.ConsumerRules.MaxDeliver,
		MaxAckPending: f.cfg.NATS.ConsumerRules.MaxAckPending,
		FilterSubject: f.config.subject,
		BackOff:       f.cfg.NATS.ConsumerRules.Backoff,
	}
}

// defaultBatchProcessor implements MessageProcessor interface
type defaultBatchProcessor struct {
	processor    *app.MessageProcessor
	dlqHandler   DLQHandler
	errorHandler *ErrorHandler
	logger       *zap.Logger
	telemetry    telemetry.Recorder
}

func (p *defaultBatchProcessor) ProcessBatch(ctx context.Context, msgs []*nats.Msg) {
	// This will be implemented when we refactor the batch processing
}

// defaultDLQHandler implements DLQHandler interface
type defaultDLQHandler struct {
	js           nats.JetStreamContext
	dlqSubject   string
	streamName   string
	consumerName string
	logger       *zap.Logger
	telemetry    telemetry.Recorder
}

func (h *defaultDLQHandler) RouteToDLQ(ctx context.Context, msg *nats.Msg, cause error) error {
	return h.routeToDLQInternal(ctx, msg, cause)
}

func (h *defaultDLQHandler) ValidateDLQ() error {
	return h.validateDLQInternal()
}

func (h *defaultDLQHandler) routeToDLQInternal(ctx context.Context, msg *nats.Msg, cause error) error {
	// Basic DLQ routing implementation
	payload := map[string]any{
		"subject":     msg.Subject,
		"stream":      h.streamName,
		"consumer":    h.consumerName,
		"error":       cause.Error(),
		"received_at": time.Now().UTC(),
		"body":        string(msg.Data),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		h.logger.Error("failed to marshal DLQ payload", zap.Error(err))
		return err
	}

	_, err = h.js.Publish(h.dlqSubject, data)
	if err != nil {
		h.logger.Error("failed to publish to DLQ",
			zap.String("dlq_subject", h.dlqSubject),
			zap.Error(err),
		)
		h.telemetry.RecordDLQPublishFailure(ctx, h.streamName, h.consumerName)
		return err
	}

	h.telemetry.RecordDLQMessage(ctx, h.streamName, h.consumerName)
	return nil
}

func (h *defaultDLQHandler) validateDLQInternal() error {
	if h.js == nil {
		return fmt.Errorf("JetStream context is nil")
	}

	_, err := h.js.StreamNameBySubject(h.dlqSubject)
	if err != nil {
		return fmt.Errorf("DLQ subject %s not bound to any JetStream stream: %w", h.dlqSubject, err)
	}

	h.logger.Info("DLQ configuration validated",
		zap.String("dlq_subject", h.dlqSubject),
	)

	return nil
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

	c.batchProcessor = &defaultBatchProcessor{
		processor:    c.processor,
		dlqHandler:   c.dlqHandler,
		errorHandler: c.errorHandler,
		logger:       c.logger,
		telemetry:    c.telemetry,
	}

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
		conn:         conn,
		js:           js,
		processor:    processor,
		cfg:          cfg,
		logger:       logger,
		telemetry:    rec,
		config:       *normCfg, // dereference the pointer
		errorHandler: NewErrorHandler(logger),
	}
	consumer.initMetrics()
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
		consumer.streamManager = NewStreamManagerWithConfig(js, normCfg.streamName, streamSubjects, &cfg.NATS.StreamLimits, logger)

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

		// Initialize advisory DLQ handler if DLQ is enabled
		if normCfg.dlqSubject != "" && cfg.DLQ.Enabled {
			advisoryHandler, err := NewAdvisoryDLQHandler(
				js,
				conn,
				normCfg.streamName,
				normCfg.consumerName,
				normCfg.dlqSubject,
				logger,
				rec,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create advisory DLQ handler: %w", err)
			}
			consumer.advisoryDLQHandler = advisoryHandler
		}
	} else {
		logger.Info("Running consumer in core NATS mode",
			zap.String("subject", normCfg.subject),
			zap.String("queue_group", cfg.Subscription.QueueGroup),
		)
	}

	return consumer, nil
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
