package nats

import (
	"caatsm/internal/app"
	"caatsm/internal/infra/config"
	"caatsm/internal/infra/telemetry"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
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

	// managers for resource lifecycle
	consumerManager *ConsumerManager
	streamManager   *StreamManager
	errorHandler    *ErrorHandler

	// simple backpressure / degradation state
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

// ProvideConsumer creates a NATS consumer.
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
		conn:            conn,
		js:              js,
		processor:       processor,
		cfg:             cfg,
		logger:          logger,
		telemetry:       rec,
		subject:         normCfg.subject,
		consumerName:    normCfg.consumerName,
		mode:            normCfg.mode,
		streamName:      normCfg.streamName,
		dlqSubject:      normCfg.dlqSubject,
		ackWait:         normCfg.ackWait,
		batchSize:       normCfg.batchSize,
		batchTimeout:    normCfg.batchTimeout,
		monitorInterval: normCfg.monitorInterval,
	}
	consumer.initMetrics()

	// Initialize managers
	consumer.errorHandler = NewErrorHandler(logger)
	if consumer.mode == "jetstream" {
		consumer.consumerManager = NewConsumerManager(js, normCfg.streamName, normCfg.consumerName, normCfg.subject, logger)
		consumer.streamManager = NewStreamManager(js, normCfg.streamName, []string{normCfg.subject}, logger)

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

// buildConsumerConfig builds the NATS consumer configuration
func (c *Consumer) buildConsumerConfig() *nats.ConsumerConfig {
	return &nats.ConsumerConfig{
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
}

// Start starts consuming messages.
func (c *Consumer) Start(ctx context.Context) error {
	if c.mode == "core" {
		return c.startCore(ctx)
	}

	return c.startJetStream(ctx)
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
