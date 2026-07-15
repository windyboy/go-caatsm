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

// DLQHandler defines the interface for dead letter queue operations
type DLQHandler interface {
	RouteToDLQ(ctx context.Context, msg *nats.Msg, cause error) error
	ValidateDLQ() error
}

// defaultDLQHandler implements DLQHandler interface
type defaultDLQHandler struct {
	js           nats.JetStreamContext
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
	// Basic DLQ routing implementation
	payload := map[string]any{
		"subject":     msg.Subject,
		"stream":      h.streamName,
		"consumer":    h.consumerName,
		"error":       cause.Error(),
		"received_at": time.Now().UTC(),
		"body":        string(msg.Data),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		h.logger.Error("failed to marshal DLQ payload", zap.Error(err))
		return err
	}

	pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err = h.js.Publish(h.dlqSubject, data, nats.Context(pubCtx))
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
	if h.js == nil {
		return fmt.Errorf("JetStream context is nil")
	}

	_, err := h.js.StreamNameBySubject(h.dlqSubject)
	if err != nil {
		return fmt.Errorf("DLQ subject %s not bound to any JetStream stream: %w", h.dlqSubject, err)
	}

	h.logger.Info("DLQ configuration validated",
		zap.String("dlq_subject", h.dlqSubject),
	)

	return nil
}
