package nats

import (
	"caatsm/internal/infra/config"
	"context"
	"errors"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// MessageFetcher defines the interface for fetching messages from NATS
type MessageFetcher interface {
	FetchBatch(ctx context.Context, sub *nats.Subscription) ([]*nats.Msg, error)
	HandleFetchError(ctx context.Context, err error, sub **nats.Subscription, fetchErrorStreak *int) (bool, error)
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
	// Context cancellation - stop processing
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		f.logger.Info("Fetch error due to context cancellation", zap.Error(err))
		return false, err
	}

	// Timeout is normal - continue
	if errors.Is(err, nats.ErrTimeout) {
		return true, nil
	}

	// Connection issues - apply simple backoff
	*fetchErrorStreak++
	backoff := f.calculateExponentialBackoff(*fetchErrorStreak)
	f.logger.Warn("Fetch error, applying backoff",
		zap.Error(err),
		zap.Int("error_streak", *fetchErrorStreak),
		zap.Duration("backoff", backoff),
	)

	if !sleepWithContext(ctx, backoff) {
		return false, ctx.Err()
	}

	return true, nil
}

// calculateExponentialBackoff calculates exponential backoff duration with a cap
func (f *defaultMessageFetcher) calculateExponentialBackoff(streak int) time.Duration {
	if streak <= 0 {
		return 0
	}
	// Simple exponential backoff: 2^(streak-1) seconds, capped at 30 seconds
	backoff := time.Duration(1<<uint(min(streak-1, 5))) * time.Second
	return min(backoff, 30*time.Second)
}
