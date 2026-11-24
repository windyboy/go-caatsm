package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// DLQPublishTimeout is the bounded timeout budget allocated for publishing a dead-letter message.
const DLQPublishTimeout = 5 * time.Second

// DLQHandler defines dead-letter queue routing and startup validation.
type DLQHandler interface {
	RouteToDLQ(ctx context.Context, msg *nats.Msg, cause error) error
	ValidateDLQ(ctx context.Context) error
}

type dlqRecorder interface {
	RecordDLQMessage(ctx context.Context, stream, consumer string)
	RecordDLQPublishFailure(ctx context.Context, stream, consumer string)
	RecordJSAPICall(operation string)
}

type defaultDLQHandler struct {
	js           nats.JetStreamContext
	dlqSubject   string
	streamName   string
	consumerName string
	logger       *zap.Logger
	recorder     dlqRecorder
}

func (h *defaultDLQHandler) RouteToDLQ(ctx context.Context, msg *nats.Msg, cause error) error {
	if h.js == nil {
		return fmt.Errorf("JetStream context is nil")
	}
	if msg == nil {
		return fmt.Errorf("message is nil")
	}

	errText := ""
	if cause != nil {
		errText = cause.Error()
	}

	payload := map[string]any{
		"subject":     msg.Subject,
		"stream":      h.streamName,
		"consumer":    h.consumerName,
		"error":       errText,
		"received_at": time.Now().UTC(),
		"body":        string(msg.Data),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		h.logger.Error("failed to marshal DLQ payload", zap.Error(err))
		return err
	}

	pubCtx, cancel := context.WithTimeout(ctx, DLQPublishTimeout)
	defer cancel()

	if h.recorder != nil {
		h.recorder.RecordJSAPICall("dlq_publish")
	}

	if _, err = h.js.Publish(h.dlqSubject, data, nats.Context(pubCtx)); err != nil {
		h.logger.Error("failed to publish to DLQ",
			zap.String("dlq_subject", h.dlqSubject),
			zap.Error(err),
		)
		if h.recorder != nil {
			h.recorder.RecordDLQPublishFailure(ctx, h.streamName, h.consumerName)
		}
		return err
	}

	if h.recorder != nil {
		h.recorder.RecordDLQMessage(ctx, h.streamName, h.consumerName)
	}
	return nil
}

func (h *defaultDLQHandler) ValidateDLQ(ctx context.Context) error {
	if h.js == nil {
		return fmt.Errorf("JetStream context is nil")
	}
	if h.recorder != nil {
		h.recorder.RecordJSAPICall("dlq_validate")
	}
	if _, err := h.js.StreamNameBySubject(h.dlqSubject, nats.Context(ctx)); err != nil {
		return fmt.Errorf("DLQ subject %s not bound to any JetStream stream: %w", h.dlqSubject, err)
	}
	h.logger.Info("DLQ configuration validated", zap.String("dlq_subject", h.dlqSubject))
	return nil
}
