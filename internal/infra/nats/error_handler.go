package nats

import (
	"caatsm/internal/app"
	"context"
	"errors"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// ErrorHandler handles various error scenarios in NATS operations
type ErrorHandler struct {
	logger *zap.Logger
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(logger *zap.Logger) *ErrorHandler {
	return &ErrorHandler{
		logger: logger,
	}
}

// FetchErrorResult represents the result of handling a fetch error
type FetchErrorResult struct {
	ShouldContinue bool
	RecoveredSub   *nats.Subscription
	Error          error
}

// HandleFetchError handles errors during message fetching with recovery logic
func (h *ErrorHandler) HandleFetchError(
	ctx context.Context,
	err error,
	sub **nats.Subscription,
	fetchErrorStreak *int,
	streamName, consumerName string,
	recoverFunc func() (*nats.Subscription, error),
) FetchErrorResult {
	if errors.Is(err, nats.ErrTimeout) {
		// Timeout is expected when no messages are available.
		return FetchErrorResult{ShouldContinue: true}
	}

	// JetStream API is currently unavailable (e.g., NATS just restarted or JetStream not ready).
	if errors.Is(err, nats.ErrNoResponders) {
		*fetchErrorStreak++
		backoff := time.Duration(*fetchErrorStreak) * time.Second
		backoff = min(backoff, 30*time.Second)
		h.logger.Warn("JetStream not available, will retry with backoff",
			zap.Error(err),
			zap.String("stream", streamName),
			zap.String("consumer", consumerName),
			zap.Duration("backoff", backoff),
		)
		if !sleepWithContext(ctx, backoff) {
			return FetchErrorResult{ShouldContinue: false, Error: ctx.Err()}
		}
		return FetchErrorResult{ShouldContinue: true}
	}

	// Underlying consumer/stream removed while app is running.
	if isJetStreamResourceNotFound(err) {
		if isDevLikeEnv() && shouldBootstrapStream() {
			h.logger.Warn("JetStream consumer or stream missing; attempting to recreate",
				zap.Error(err),
				zap.String("stream", streamName),
				zap.String("consumer", consumerName),
			)
			newSub, subErr := recoverFunc()
			if subErr != nil {
				return FetchErrorResult{ShouldContinue: false, Error: subErr}
			}
			*sub = newSub
			*fetchErrorStreak = 0
			return FetchErrorResult{ShouldContinue: true, RecoveredSub: newSub}
		}

		// Production: treat as configuration/operational error.
		h.logger.Error("JetStream consumer or stream missing; not auto-recreating in this environment",
			zap.Error(err),
			zap.String("stream", streamName),
			zap.String("consumer", consumerName),
		)
		return FetchErrorResult{ShouldContinue: false, Error: err}
	}

	// Generic error path with modest backoff.
	*fetchErrorStreak++
	backoff := time.Duration(*fetchErrorStreak) * time.Second
	backoff = min(backoff, 10*time.Second)
	h.logger.Error("Failed to fetch messages; backing off",
		zap.Error(err),
		zap.Duration("backoff", backoff),
	)
	if !sleepWithContext(ctx, backoff) {
		return FetchErrorResult{ShouldContinue: false, Error: ctx.Err()}
	}
	return FetchErrorResult{ShouldContinue: true}
}

// ProcessingErrorResult represents the result of handling a processing error
type ProcessingErrorResult struct {
	IsPermanent             bool
	ShouldApplyBackpressure bool
	BackpressureDelay       time.Duration
}

// HandleProcessingError analyzes processing errors and determines appropriate action
func (h *ErrorHandler) HandleProcessingError(
	consecutiveErrors int,
	err error,
	logger *zap.Logger,
	subject string,
) ProcessingErrorResult {
	isPermanent := app.IsPermanent(err)

	result := ProcessingErrorResult{
		IsPermanent: isPermanent,
	}

	if isPermanent {
		// Reset error streak for permanent errors
		return result
	}

	// Transient error: increment error streak and apply simple backpressure if needed.
	if consecutiveErrors >= 10 {
		result.ShouldApplyBackpressure = true
		result.BackpressureDelay = time.Duration(consecutiveErrors) * 100 * time.Millisecond
		result.BackpressureDelay = min(result.BackpressureDelay, 5*time.Second)
	}

	return result
}
