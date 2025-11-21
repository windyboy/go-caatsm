package nats

import (
	"caatsm/internal/app"
	"caatsm/internal/infra/log"
	obsmetrics "caatsm/internal/infra/metrics"
	"caatsm/internal/infra/telemetry"
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

// MessageProcessor defines the interface for processing message batches
type MessageProcessor interface {
	ProcessBatch(ctx context.Context, msgs []*nats.Msg)
	ProcessMessage(ctx context.Context, msg *nats.Msg) error
}

// ProcessingErrorResult represents the result of handling a processing error
type ProcessingErrorResult struct {
	IsPermanent             bool
	ShouldApplyBackpressure bool
	BackpressureDelay       time.Duration
}

// defaultBatchProcessor implements MessageProcessor interface
type defaultBatchProcessor struct {
	processor  *app.MessageProcessor
	dlqHandler DLQHandler
	logger     *zap.Logger
	telemetry  telemetry.Recorder
	// Configuration needed for processing
	streamName   string
	consumerName string
	backoff      []time.Duration
	// Pointer to consecutive errors counter (shared with Consumer)
	consecutiveProcessErrors *int
}

func (p *defaultBatchProcessor) ProcessBatch(ctx context.Context, msgs []*nats.Msg) {
	for _, msg := range msgs {
		// Check context before processing each message
		select {
		case <-ctx.Done():
			p.logger.Info("Stopping batch processing due to cancellation",
				zap.Int("remaining_messages", len(msgs)),
			)
			return
		default:
		}
		p.processSingleMessage(ctx, msg)
	}
}

// processSingleMessage processes a single message with error handling and backpressure.
func (p *defaultBatchProcessor) processSingleMessage(ctx context.Context, msg *nats.Msg) {
	start := time.Now()

	if err := p.ProcessMessage(ctx, msg); err != nil {
		p.handleMessageError(ctx, msg, err, time.Since(start))
		return
	}

	// Successful processing resets the error streak.
	if p.consecutiveProcessErrors != nil && *p.consecutiveProcessErrors > 0 {
		*p.consecutiveProcessErrors = 0
	}

	elapsed := time.Since(start)

	// ACK the message
	if ackErr := msg.Ack(); ackErr != nil {
		p.logger.Error("Failed to ACK message", zap.Error(ackErr))
		// Still record metrics even if ACK fails
	}
	p.telemetry.RecordMessageHandled(ctx, p.streamName, p.consumerName, "ok", elapsed)
}

// ProcessMessage processes a single message.
func (p *defaultBatchProcessor) ProcessMessage(ctx context.Context, msg *nats.Msg) error {
	ctx, span := otel.Tracer("caatsm/nats").Start(ctx, "Consumer.processMessage")
	defer span.End()

	// Set semantic messaging attributes
	span.SetAttributes(
		attribute.String("messaging.system", "nats"),
		attribute.String("messaging.operation.name", "receive"),
		attribute.String("messaging.destination.name", msg.Subject),
		attribute.String("messaging.consumer.group.name", p.consumerName),
		attribute.String("caatsm.stream", p.streamName),
	)

	msgID, source, err := p.resolveMsgID(msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("unable to resolve message id: %w", err)
	}
	if source != "header" {
		p.logger.Warn("Message missing NATS id header; using fallback",
			zap.String("subject", msg.Subject),
			zap.String("msg_id_source", source),
			zap.String("msg_id", msgID),
		)
	}

	// Attach structured logging context including stream/consumer and NATS metadata.
	jsSeq := uint64(0)
	if meta, metaErr := msg.Metadata(); metaErr == nil {
		jsSeq = meta.Sequence.Stream
		span.SetAttributes(
			attribute.Int64("nats.js.stream_seq", int64(meta.Sequence.Stream)),
			attribute.Int64("nats.js.consumer_seq", int64(meta.Sequence.Consumer)),
		)
	}

	msgLogger := log.WithMessageContext(p.logger, log.MessageFields{
		Service:        "caatsm-consumer",
		TransportMsgID: msgID,
		Stream:         p.streamName,
		Consumer:       p.consumerName,
		Subject:        msg.Subject,
		JSSequence:     jsSeq,
	})

	msgLogger.Debug("Processing message",
		zap.Int("data_size", len(msg.Data)),
		zap.String("msg_id_source", source),
	)

	// Call processor
	if err := p.processor.Handle(ctx, msg.Data, msgID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("processor error: %w", err)
	}

	span.SetAttributes(attribute.String("telegram.msg_id", msgID))
	return nil
}

// resolveMsgID extracts or generates a message ID.
func (p *defaultBatchProcessor) resolveMsgID(msg *nats.Msg) (string, string, error) {
	if id := msg.Header.Get("Nats-Msg-Id"); id != "" {
		return id, "header", nil
	}

	meta, err := msg.Metadata()
	if err != nil {
		return "", "", fmt.Errorf("fetch metadata: %w", err)
	}

	return fmt.Sprintf("js-%d", meta.Sequence.Stream), "metadata", nil
}

// handleMessageError handles errors that occur during message processing.
func (p *defaultBatchProcessor) handleMessageError(ctx context.Context, msg *nats.Msg, err error, elapsed time.Duration) {
	// Check if context is cancelled before processing
	select {
	case <-ctx.Done():
		p.logger.Warn("Skipping error handling due to context cancellation",
			zap.String("subject", msg.Subject),
		)
		return
	default:
	}

	// Extract message ID for better error logging
	msgID, _, _ := p.resolveMsgID(msg)
	if msgID == "" {
		msgID = "unknown"
	}

	isPermanent := app.IsPermanent(err)
	p.logger.Error("Failed to process message",
		zap.String("subject", msg.Subject),
		zap.String("msg_id", msgID),
		zap.Error(err),
		zap.Bool("permanent", isPermanent),
	)

	result := obsmetrics.ResultFail
	if isPermanent {
		result = obsmetrics.ResultPermanentFail
	}
	p.telemetry.RecordMessageHandled(ctx, p.streamName, p.consumerName, result, elapsed)

	consecutiveErrors := 0
	if p.consecutiveProcessErrors != nil {
		consecutiveErrors = *p.consecutiveProcessErrors
	}

	processingResult := ProcessingErrorResult{IsPermanent: isPermanent}
	if !isPermanent && consecutiveErrors >= 10 {
		processingResult.ShouldApplyBackpressure = true
		processingResult.BackpressureDelay = time.Duration(consecutiveErrors) * 100 * time.Millisecond
		if processingResult.BackpressureDelay > 5*time.Second {
			processingResult.BackpressureDelay = 5 * time.Second
		}
	}

	if processingResult.IsPermanent {
		p.handlePermanentError(ctx, msg, err)
		return
	}

	p.handleTransientError(ctx, msg, processingResult)
}

// handlePermanentError handles permanent/poison messages.
func (p *defaultBatchProcessor) handlePermanentError(ctx context.Context, msg *nats.Msg, err error) {
	if p.consecutiveProcessErrors != nil {
		*p.consecutiveProcessErrors = 0
	}

	// Extract message ID for better logging
	msgID, _, _ := p.resolveMsgID(msg)
	if msgID == "" {
		msgID = "unknown"
	}

	// Poison/permanent message: route to DLQ if configured, then ACK
	dlqRouted := false
	if p.dlqHandler != nil {
		if dlqErr := p.dlqHandler.RouteToDLQ(ctx, msg, err); dlqErr != nil {
			p.logger.Error("Failed to route permanent-error message to DLQ",
				zap.String("subject", msg.Subject),
				zap.String("msg_id", msgID),
				zap.Error(dlqErr),
				zap.NamedError("original_error", err),
			)
			// Note: We still ACK the message even if DLQ routing fails to prevent
			// infinite redelivery of poison messages. The error is logged for manual investigation.
		} else {
			dlqRouted = true
			p.logger.Info("Permanent-error message routed to DLQ",
				zap.String("subject", msg.Subject),
				zap.String("msg_id", msgID),
			)
		}
	} else {
		p.logger.Warn("Permanent-error message but DLQ handler not configured - message will be ACKed without DLQ routing",
			zap.String("subject", msg.Subject),
			zap.String("msg_id", msgID),
			zap.String("hint", "Enable DLQ by setting dlq.enabled=true and dlq.subject in config to route poison messages for inspection"),
		)
	}

	// ACK the message to prevent redelivery
	// Even if DLQ routing failed, we ACK to avoid infinite retries of poison messages
	if ackErr := msg.Ack(); ackErr != nil {
		p.logger.Error("Failed to ACK permanent-error message",
			zap.String("subject", msg.Subject),
			zap.String("msg_id", msgID),
			zap.Bool("dlq_routed", dlqRouted),
			zap.Error(ackErr),
		)
	}
}

// handleTransientError handles transient errors with backpressure and redelivery.
func (p *defaultBatchProcessor) handleTransientError(ctx context.Context, msg *nats.Msg, processingResult ProcessingErrorResult) {
	// Increment error streak
	if p.consecutiveProcessErrors != nil {
		if *p.consecutiveProcessErrors < 0 {
			*p.consecutiveProcessErrors = 0
		}
		*p.consecutiveProcessErrors++
	}

	if processingResult.ShouldApplyBackpressure {
		consecutiveErrors := 0
		if p.consecutiveProcessErrors != nil {
			consecutiveErrors = *p.consecutiveProcessErrors
		}
		p.logger.Warn("Applying backpressure due to consecutive processing errors",
			zap.Int("consecutive_errors", consecutiveErrors),
			zap.Duration("sleep", processingResult.BackpressureDelay),
		)
		// Use context-aware sleep instead of blocking time.Sleep
		if !sleepWithContext(ctx, processingResult.BackpressureDelay) {
			// Context canceled, stop processing
			return
		}
	}

	// Transient error: request redelivery with optional delay
	p.telemetry.RecordRetry(ctx, p.streamName, p.consumerName, obsmetrics.RetryReasonProcessorError)
	if nakErr := p.nakWithStrategy(msg); nakErr != nil {
		p.logger.Error("Failed to NAK message", zap.Error(nakErr))
	}
}

// nakWithStrategy sends a NAK with appropriate delay based on retry attempt.
func (p *defaultBatchProcessor) nakWithStrategy(msg *nats.Msg) error {
	if len(p.backoff) == 0 {
		return msg.Nak()
	}

	meta, err := msg.Metadata()
	if err != nil {
		p.logger.Warn("Failed to read metadata for backoff strategy", zap.Error(err))
		return msg.Nak()
	}

	attempt := int(meta.NumDelivered)
	index := attempt - 1
	if index < 0 {
		index = 0
	}
	if index >= len(p.backoff) {
		index = len(p.backoff) - 1
	}
	delay := p.backoff[index]
	if delay <= 0 {
		return msg.Nak()
	}

	return msg.NakWithDelay(delay)
}
