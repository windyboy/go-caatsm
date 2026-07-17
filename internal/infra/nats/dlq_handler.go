package nats

import (
	"caatsm/internal/infra/telemetry"
	"context"
	"encoding/base64"
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
}

// DLQHandler defines the DLQ operation used by the consumer.
type DLQHandler interface {
	RouteToDLQ(ctx context.Context, msg *nats.Msg, cause error) error
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

type dlqPayload struct {
	SchemaVersion    string      `json:"schema_version"`
	MessageID        string      `json:"message_id"`
	Subject          string      `json:"subject"`
	Stream           string      `json:"stream"`
	Consumer         string      `json:"consumer"`
	StreamSequence   uint64      `json:"stream_sequence"`
	ConsumerSequence uint64      `json:"consumer_sequence"`
	DeliveryCount    uint64      `json:"delivery_count"`
	Error            string      `json:"error"`
	Headers          nats.Header `json:"headers"`
	BodyBase64       string      `json:"body_base64"`
	ReceivedAt       time.Time   `json:"received_at"`
}

func (h *defaultDLQHandler) RouteToDLQ(ctx context.Context, msg *nats.Msg, cause error) error {
	headers := make(nats.Header, len(msg.Header))
	for key, values := range msg.Header {
		headers[key] = append([]string(nil), values...)
	}
	payload := dlqPayload{
		SchemaVersion: "v1",
		MessageID:     msg.Header.Get("Nats-Msg-Id"),
		Subject:       msg.Subject,
		Stream:        h.streamName,
		Consumer:      h.consumerName,
		Error:         cause.Error(),
		Headers:       headers,
		BodyBase64:    base64.StdEncoding.EncodeToString(msg.Data),
		ReceivedAt:    time.Now().UTC(),
	}
	if meta, err := msg.Metadata(); err == nil {
		payload.StreamSequence = meta.Sequence.Stream
		payload.ConsumerSequence = meta.Sequence.Consumer
		payload.DeliveryCount = meta.NumDelivered
		if payload.MessageID == "" {
			payload.MessageID = fmt.Sprintf("%s:%d", h.streamName, meta.Sequence.Stream)
		}
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
