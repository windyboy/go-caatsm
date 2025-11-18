package nats

import (
	"caatsm/internal/app"
	obsmetrics "caatsm/internal/infra/metrics"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// ensureConsumer creates the consumer if it doesn't exist; if it already exists, it is reused.
func (c *Consumer) ensureConsumer() error {
	consumerConfig := c.buildConsumerConfig()
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

	return c.consumerManager.EnsureConsumer(consumerConfig)
}

// recoverJetStreamResources attempts to recreate the stream and consumer in
// dev/test environments if they are missing. It is safe to call multiple times.
func (c *Consumer) recoverJetStreamResources() error {
	if c.js == nil {
		return fmt.Errorf("jetstream context is nil")
	}
	if c.cfg == nil {
		return fmt.Errorf("config is nil")
	}

	consumerConfig := c.buildConsumerConfig()
	return c.consumerManager.RecoverResources(c.streamManager, consumerConfig)
}

// createPullSubscriptionWithRecovery creates a pull subscription and, in
// dev/test environments, attempts to self-heal missing stream/consumer
// by recreating them once.
func (c *Consumer) createPullSubscriptionWithRecovery() (*nats.Subscription, error) {
	consumerConfig := c.buildConsumerConfig()
	return c.consumerManager.CreatePullSubscriptionWithRecovery(c.streamManager, consumerConfig)
}

// nakWithStrategy sends a NAK with appropriate delay based on retry attempt.
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

// sleepWithContext sleeps for the specified duration, but returns early if the context is canceled.
// Returns true if the full duration was slept, false if the context was canceled.
func sleepWithContext(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// fetchBatch fetches a batch of messages from the subscription.
func (c *Consumer) fetchBatch(sub *nats.Subscription) ([]*nats.Msg, error) {
	return sub.Fetch(c.batchSize, nats.MaxWait(c.batchTimeout))
}

// handleFetchError handles errors during message fetching, including recovery logic.
// Returns true if the error was handled and consumption should continue, false otherwise.
func (c *Consumer) handleFetchError(ctx context.Context, err error, sub **nats.Subscription, fetchErrorStreak *int) (bool, error) {
	result := c.errorHandler.HandleFetchError(ctx, err, sub, fetchErrorStreak, c.streamName, c.consumerName, func() (*nats.Subscription, error) {
		if recErr := c.recoverJetStreamResources(); recErr != nil {
			return nil, recErr
		}
		(*sub).Unsubscribe()
		return c.createPullSubscriptionWithRecovery()
	})

	if result.RecoveredSub != nil {
		*sub = result.RecoveredSub
		*fetchErrorStreak = 0
	}

	return result.ShouldContinue, result.Error
}

// processBatch processes a batch of messages, handling errors and applying backpressure.
func (c *Consumer) processBatch(ctx context.Context, msgs []*nats.Msg) {
	for _, msg := range msgs {
		c.processSingleMessage(ctx, msg)
	}
}

// processSingleMessage processes a single message with error handling and backpressure.
func (c *Consumer) processSingleMessage(ctx context.Context, msg *nats.Msg) {
	start := time.Now()

	if err := c.processMessage(ctx, msg); err != nil {
		c.handleMessageError(ctx, msg, err, time.Since(start))
		return
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

// handleMessageError handles errors that occur during message processing.
func (c *Consumer) handleMessageError(ctx context.Context, msg *nats.Msg, err error, elapsed time.Duration) {
	c.logger.Error("Failed to process message",
		zap.String("subject", msg.Subject),
		zap.Error(err),
		zap.Bool("permanent", app.IsPermanent(err)),
	)

	result := obsmetrics.ResultFail
	if app.IsPermanent(err) {
		result = obsmetrics.ResultPermanentFail
	}
	c.telemetry.RecordMessageHandled(ctx, c.streamName, c.consumerName, result, elapsed)

	processingResult := c.errorHandler.HandleProcessingError(c.consecutiveProcessErrors, err, c.logger, msg.Subject)

	if processingResult.IsPermanent {
		c.handlePermanentError(ctx, msg, err)
		return
	}

	c.handleTransientError(ctx, msg, processingResult)
}

// handlePermanentError handles permanent/poison messages.
func (c *Consumer) handlePermanentError(ctx context.Context, msg *nats.Msg, err error) {
	c.consecutiveProcessErrors = 0
	// Poison/permanent message: route to DLQ if configured, then ACK
	if dlqErr := c.routeToDLQ(ctx, msg, err); dlqErr != nil {
		c.logger.Error("Failed to route permanent-error message to DLQ", zap.Error(dlqErr))
	}
	if ackErr := msg.Ack(); ackErr != nil {
		c.logger.Error("Failed to ACK permanent-error message", zap.Error(ackErr))
	}
}

// handleTransientError handles transient errors with backpressure and redelivery.
func (c *Consumer) handleTransientError(ctx context.Context, msg *nats.Msg, processingResult ProcessingErrorResult) {
	// Increment error streak
	if c.consecutiveProcessErrors < 0 {
		c.consecutiveProcessErrors = 0
	}
	c.consecutiveProcessErrors++

	if processingResult.ShouldApplyBackpressure {
		c.logger.Warn("Applying backpressure due to consecutive processing errors",
			zap.Int("consecutive_errors", c.consecutiveProcessErrors),
			zap.Duration("sleep", processingResult.BackpressureDelay),
		)
		// Use context-aware sleep instead of blocking time.Sleep
		if !sleepWithContext(ctx, processingResult.BackpressureDelay) {
			// Context canceled, stop processing
			return
		}
	}

	// Transient error: request redelivery with optional delay
	c.telemetry.RecordRetry(ctx, c.streamName, c.consumerName, obsmetrics.RetryReasonProcessorError)
	if nakErr := c.nakWithStrategy(msg); nakErr != nil {
		c.logger.Error("Failed to NAK message", zap.Error(nakErr))
	}
}

// startJetStream starts the JetStream consumer loop.
func (c *Consumer) startJetStream(ctx context.Context) error {
	// Create pull subscription (with simple self-healing in dev/test).
	sub, err := c.createPullSubscriptionWithRecovery()
	if err != nil {
		return err
	}

	// Use a closure that always cleans up the current subscription.
	// When subscription is replaced in handleFetchError, this will clean up
	// whatever currentSub points to at shutdown time.
	var currentSub *nats.Subscription = sub
	cleanupSubscriber := func() {
		if currentSub != nil {
			currentSub.Unsubscribe()
			currentSub = nil
		}
	}
	defer cleanupSubscriber()

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

	var fetchErrorStreak int

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Stopping consumer", zap.Error(ctx.Err()))
			return ctx.Err()
		default:
		}

		// Fetch messages in batch
		msgs, err := c.fetchBatch(currentSub)
		if err != nil {
			shouldContinue, handleErr := c.handleFetchError(ctx, err, &currentSub, &fetchErrorStreak)
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
		c.processBatch(ctx, msgs)
	}
}
