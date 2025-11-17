package nats

import (
	"caatsm/internal/infra/log"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

// processMessage processes a single message.
func (c *Consumer) processMessage(ctx context.Context, msg *nats.Msg) error {
	ctx, span := otel.Tracer("caatsm/nats").Start(ctx, "Consumer.processMessage")
	defer span.End()
	span.SetAttributes(attribute.String("nats.subject", msg.Subject))

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
		Stream:         c.streamName,
		Consumer:       c.consumerName,
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

	if c.mode == "core" {
		return uuid.NewString(), "generated", nil
	}

	meta, err := msg.Metadata()
	if err != nil {
		return "", "", fmt.Errorf("fetch metadata: %w", err)
	}

	return fmt.Sprintf("js-%d", meta.Sequence.Stream), "metadata", nil
}

