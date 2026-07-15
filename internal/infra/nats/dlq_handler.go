package nats

import (
	"caatsm/internal/infra/telemetry"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// dlqPublisher is the minimal interface needed by the DLQ handler,
// making it easy to mock in tests without requiring the full JetStreamContext.
type dlqPublisher interface {
	Publish(subj string, data []byte, opts ...nats.PubOpt) (*nats.PubAck, error)
	StreamNameBySubject(subj string, opts ...nats.JSOpt) (string, error)
}

// DLQHandler defines the interface for dead letter queue operations
type DLQHandler interface {
	RouteToDLQ(ctx context.Context, msg *nats.Msg, cause error) error
	ValidateDLQ() error
}

// defaultDLQHandler implements DLQHandler interface
type defaultDLQHandler struct {
	publisher    dlqPublisher
	dlqSubject   string
	streamName   string
	consumerName string
	logger       *zap.Logger
	telemetry    telemetry.Recorder
}

func (h *defaultDLQHandler) RouteToDLQ(ctx context.Context, msg *nats.Msg, cause error) error {
	return h.routeToDLQInternal(ctx, msg, cause)
}

func (h *defaultDLQHandler) ValidateDLQ() error {
	return h.validateDLQInternal()
}

func (h *defaultDLQHandler) routeToDLQInternal(ctx context.Context, msg *nats.Msg, cause error) error {
	payload := map[string]any{
		"subject":          msg.Subject,
		"stream":           h.streamName,
		"consumer":         h.consumerName,
		"error":            cause.Error(),
		"received_at":      time.Now().UTC(),
		"body":             string(msg.Data),
		"transport_msg_id": msg.Header.Get("Nats-Msg-Id"),
	}
	// Enrich with JetStream metadata when available
	if meta, err := msg.Metadata(); err == nil {
		payload["nats_sequence"] = meta.Sequence.Stream
		payload["deliveries"] = meta.NumDelivered
	}
	if msg.Reply != "" {
		payload["reply"] = msg.Reply
	}
	if len(msg.Header) > 0 {
		headers := make(map[string][]string, len(msg.Header))
		for k, v := range msg.Header {
			headers[k] = v
		}
		payload["headers"] = headers
	}

	data, err := json.Marshal(payload)
	if err != nil {
		h.logger.Error("failed to marshal DLQ payload", zap.Error(err))
		return err
	}

	pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err = h.publisher.Publish(h.dlqSubject, data, nats.Context(pubCtx))
	if err != nil {
		h.logger.Error("failed to publish to DLQ",
			zap.String("dlq_subject", h.dlqSubject),
			zap.Error(err),
		)
		h.telemetry.RecordDLQPublishFailure(ctx, h.streamName, h.consumerName)
		return err
	}

	h.telemetry.RecordDLQMessage(ctx, h.streamName, h.consumerName)
	return nil
}

func (h *defaultDLQHandler) validateDLQInternal() error {
	if h.publisher == nil {
		return fmt.Errorf("JetStream context is nil")
	}

	_, err := h.publisher.StreamNameBySubject(h.dlqSubject)
	if err != nil {
		return fmt.Errorf("DLQ subject %s not bound to any JetStream stream: %w", h.dlqSubject, err)
	}

	h.logger.Info("DLQ configuration validated",
		zap.String("dlq_subject", h.dlqSubject),
	)

	return nil
}
