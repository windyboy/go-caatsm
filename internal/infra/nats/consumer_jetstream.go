package nats

import (
	"caatsm/internal/app"
	obsmetrics "caatsm/internal/infra/metrics"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

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

// recoverJetStreamResources attempts to recreate the stream and consumer in
// dev/test environments if they are missing. It is safe to call multiple times.
func (c *Consumer) recoverJetStreamResources() error {
	if c.js == nil {
		return fmt.Errorf("jetstream context is nil")
	}
	if c.cfg == nil {
		return fmt.Errorf("config is nil")
	}

	// Ensure stream exists (dev/test may auto-create, prod will error).
	if err := EnsureStream(c.js, c.cfg, c.logger); err != nil {
		return fmt.Errorf("ensure stream %s: %w", c.streamName, err)
	}

	// Ensure durable consumer exists and is properly bound.
	if err := c.ensureConsumer(); err != nil {
		return fmt.Errorf("ensure consumer %s: %w", c.consumerName, err)
	}

	return nil
}

// createPullSubscriptionWithRecovery creates a pull subscription and, in
// dev/test environments, attempts to self-heal missing stream/consumer
// by recreating them once.
func (c *Consumer) createPullSubscriptionWithRecovery() (*nats.Subscription, error) {
	sub, err := c.js.PullSubscribe(c.subject, c.consumerName, nats.Bind(c.streamName, c.consumerName))
	if err == nil {
		return sub, nil
	}

	if isJetStreamResourceNotFound(err) && isDevLikeEnv() && shouldBootstrapStream() {
		c.logger.Warn("PullSubscribe failed due to missing JetStream resources; attempting to recreate",
			zap.Error(err),
			zap.String("stream", c.streamName),
			zap.String("consumer", c.consumerName),
		)
		if recErr := c.recoverJetStreamResources(); recErr != nil {
			return nil, fmt.Errorf("failed to recover JetStream resources: %w", recErr)
		}
		// Retry subscription after successful recovery.
		sub, err = c.js.PullSubscribe(c.subject, c.consumerName, nats.Bind(c.streamName, c.consumerName))
		if err != nil {
			return nil, fmt.Errorf("failed to create pull subscription after recovery: %w", err)
		}
		return sub, nil
	}

	return nil, fmt.Errorf("failed to create pull subscription: %w", err)
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
	if errors.Is(err, nats.ErrTimeout) {
		// Timeout is expected when no messages are available.
		return true, nil
	}

	// JetStream API is currently unavailable (e.g., NATS just restarted or JetStream not ready).
	if errors.Is(err, nats.ErrNoResponders) {
		*fetchErrorStreak++
		backoff := time.Duration(*fetchErrorStreak) * time.Second
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
		c.logger.Warn("JetStream not available, will retry with backoff",
			zap.Error(err),
			zap.String("stream", c.streamName),
			zap.String("consumer", c.consumerName),
			zap.Duration("backoff", backoff),
		)
		// Use context-aware sleep instead of blocking time.Sleep
		if !sleepWithContext(ctx, backoff) {
			return false, ctx.Err()
		}
		return true, nil
	}

	// Underlying consumer/stream removed while app is running.
	if isJetStreamResourceNotFound(err) {
		if isDevLikeEnv() && shouldBootstrapStream() {
			c.logger.Warn("JetStream consumer or stream missing; attempting to recreate",
				zap.Error(err),
				zap.String("stream", c.streamName),
				zap.String("consumer", c.consumerName),
			)
			if recErr := c.recoverJetStreamResources(); recErr != nil {
				return false, recErr
			}

			// Recreate subscription after successful recovery.
			(*sub).Unsubscribe()
			newSub, subErr := c.createPullSubscriptionWithRecovery()
			if subErr != nil {
				return false, subErr
			}
			*sub = newSub
			*fetchErrorStreak = 0
			return true, nil
		}

		// Production: treat as configuration/operational error.
		c.logger.Error("JetStream consumer or stream missing; not auto-recreating in this environment",
			zap.Error(err),
			zap.String("stream", c.streamName),
			zap.String("consumer", c.consumerName),
		)
		return false, err
	}

	// Generic error path with modest backoff.
	*fetchErrorStreak++
	backoff := time.Duration(*fetchErrorStreak) * time.Second
	if backoff > 10*time.Second {
		backoff = 10 * time.Second
	}
	c.logger.Error("Failed to fetch messages; backing off",
		zap.Error(err),
		zap.Duration("backoff", backoff),
	)
	// Use context-aware sleep instead of blocking time.Sleep
	if !sleepWithContext(ctx, backoff) {
		return false, ctx.Err()
	}
	return true, nil
}

// processBatch processes a batch of messages, handling errors and applying backpressure.
func (c *Consumer) processBatch(ctx context.Context, msgs []*nats.Msg) {
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
				// Use context-aware sleep instead of blocking time.Sleep
				if !sleepWithContext(ctx, backoff) {
					// Context canceled, stop processing batch
					return
				}
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
