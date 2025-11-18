package nats

import (
	"caatsm/internal/app"
	obsmetrics "caatsm/internal/infra/metrics"
	"context"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// handleMessageError handles errors that occur during message processing.
//
//nolint:unused // Reserved for potential future use or alternative implementation
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
	c.telemetry.RecordMessageHandled(ctx, c.config.streamName, c.config.consumerName, result, elapsed)

	processingResult := c.errorHandler.HandleProcessingError(c.consecutiveProcessErrors, err, c.logger, msg.Subject)

	if processingResult.IsPermanent {
		c.handlePermanentError(ctx, msg, err)
		return
	}

	c.handleTransientError(ctx, msg, processingResult)
}

// handlePermanentError handles permanent/poison messages.
//
//nolint:unused // Reserved for potential future use or alternative implementation
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
//
//nolint:unused // Reserved for potential future use or alternative implementation
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
	c.telemetry.RecordRetry(ctx, c.config.streamName, c.config.consumerName, obsmetrics.RetryReasonProcessorError)
	if nakErr := c.nakWithStrategy(msg); nakErr != nil {
		c.logger.Error("Failed to NAK message", zap.Error(nakErr))
	}
}
