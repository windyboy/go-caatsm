package nats

import (
	"caatsm/internal/infra/log"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

// processBatch processes a batch of messages, handling errors and applying backpressure.
// It checks context cancellation between messages for faster shutdown.
func (c *Consumer) processBatch(ctx context.Context, msgs []*nats.Msg) {
	for _, msg := range msgs {
		// Check context before processing each message
		select {
		case <-ctx.Done():
			c.logger.Info("Stopping batch processing due to cancellation",
				zap.Int("remaining_messages", len(msgs)),
			)
			return
		default:
		}
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
		c.telemetry.RecordMessageHandled(ctx, c.config.streamName, c.config.consumerName, "ok", elapsed)
	}
}

// processMessage processes a single message.
func (c *Consumer) processMessage(ctx context.Context, msg *nats.Msg) error {
	ctx, span := otel.Tracer("caatsm/nats").Start(ctx, "Consumer.processMessage")
	defer span.End()

	// Set semantic messaging attributes
	span.SetAttributes(
		attribute.String("messaging.system", "nats"),
		attribute.String("messaging.operation.name", "receive"),
		attribute.String("messaging.destination.name", msg.Subject),
		attribute.String("messaging.consumer.group.name", c.config.consumerName),
		attribute.String("caatsm.stream", c.config.streamName),
	)

	msgID, source, err := c.resolveMsgID(msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("unable to resolve message id: %w", err)
	}
	if source != "header" {
		c.logger.Warn("Message missing NATS id header; using fallback",
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

	msgLogger := log.WithMessageContext(c.logger, log.MessageFields{
		Service:        "caatsm-consumer",
		TransportMsgID: msgID,
		Stream:         c.config.streamName,
		Consumer:       c.config.consumerName,
		Subject:        msg.Subject,
		JSSequence:     jsSeq,
	})

	msgLogger.Debug("Processing message",
		zap.Int("data_size", len(msg.Data)),
		zap.String("msg_id_source", source),
	)

	// Call processor
	if err := c.processor.Handle(ctx, msg.Data, msgID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("processor error: %w", err)
	}

	span.SetAttributes(attribute.String("telegram.msg_id", msgID))
	return nil
}

// resolveMsgID extracts or generates a message ID.
func (c *Consumer) resolveMsgID(msg *nats.Msg) (string, string, error) {
	if id := msg.Header.Get("Nats-Msg-Id"); id != "" {
		return id, "header", nil
	}

	if c.config.mode == "core" {
		return uuid.NewString(), "generated", nil
	}

	meta, err := msg.Metadata()
	if err != nil {
		return "", "", fmt.Errorf("fetch metadata: %w", err)
	}

	return fmt.Sprintf("js-%d", meta.Sequence.Stream), "metadata", nil
}
