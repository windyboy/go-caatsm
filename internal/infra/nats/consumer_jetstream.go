package nats

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

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

//nolint:unused // Reserved for potential future use or alternative implementation
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

//nolint:unused // Reserved for potential future use or alternative implementation
// fetchBatch fetches a batch of messages from the subscription.
// It respects context cancellation for faster shutdown.
func (c *Consumer) fetchBatch(ctx context.Context, sub *nats.Subscription) ([]*nats.Msg, error) {
	// Check context before fetching
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Use a shorter timeout for better responsiveness to cancellation
	// The batchTimeout is still used, but we'll check context more frequently
	timeout := c.config.batchTimeout
	if timeout > 500*time.Millisecond {
		// Cap at 500ms to improve responsiveness while still allowing batching
		timeout = 500 * time.Millisecond
	}

	return sub.Fetch(c.config.batchSize, nats.MaxWait(timeout))
}

// handleFetchError handles errors during message fetching, including recovery logic.
// Returns true if the error was handled and consumption should continue, false otherwise.
func (c *Consumer) handleFetchError(ctx context.Context, err error, sub **nats.Subscription, fetchErrorStreak *int) (bool, error) {
	result := c.errorHandler.HandleFetchError(ctx, err, sub, fetchErrorStreak, c.config.streamName, c.config.consumerName, func() (*nats.Subscription, error) {
		if recErr := c.recoverJetStreamResources(); recErr != nil {
			return nil, recErr
		}
		if *sub != nil {
			if unsubErr := (*sub).Unsubscribe(); unsubErr != nil {
				c.logger.Error("Failed to unsubscribe during recovery", zap.Error(unsubErr))
			}
		}
		return c.createPullSubscriptionWithRecovery()
	})

	if result.RecoveredSub != nil {
		*sub = result.RecoveredSub
		*fetchErrorStreak = 0
	}

	return result.ShouldContinue, result.Error
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

	c.logger.Info("Consumer pull configuration",
		zap.Int("batch_size", c.config.batchSize),
		zap.Duration("batch_timeout", c.config.batchTimeout),
		zap.Int("max_deliver", c.cfg.NATS.ConsumerRules.MaxDeliver),
		zap.Duration("ack_wait", c.config.ackWait),
		zap.String("deliver_policy", c.cfg.NATS.ConsumerRules.DeliverPolicy),
		zap.String("replay_policy", c.cfg.NATS.ConsumerRules.ReplayPolicy),
		zap.Int("backoff_steps", len(c.cfg.NATS.ConsumerRules.Backoff)),
	)

	statsCtx, statsCancel := context.WithCancel(ctx)
	defer statsCancel()
	go c.emitConsumerStats(statsCtx)

	// Start advisory DLQ handler in background if configured
	if c.advisoryDLQHandler != nil {
		go func() {
			if err := c.advisoryDLQHandler.Start(ctx); err != nil {
				c.logger.Error("Advisory DLQ handler failed", zap.Error(err))
			}
		}()
	}

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
