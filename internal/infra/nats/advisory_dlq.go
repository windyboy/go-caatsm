package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// MaxDeliveriesAdvisoryEvent represents the advisory message published when
// a message reaches MaxDeliver attempts.
type MaxDeliveriesAdvisoryEvent struct {
	Type       string `json:"type"`
	Stream     string `json:"stream"`
	Consumer   string `json:"consumer"`
	StreamSeq  uint64 `json:"stream_seq"`
	Deliveries uint64 `json:"deliveries"`
	Time       string `json:"time"`
}

// AdvisoryDLQHandler handles messages that exhaust MaxDeliver attempts
// by subscribing to JetStream advisory events.
type AdvisoryDLQHandler struct {
	js           nats.JetStreamContext
	nc           *nats.Conn
	streamName   string
	consumerName string
	dlqSubject   string
	logger       *zap.Logger
	telemetry    TelemetryRecorder
}

// TelemetryRecorder is an interface for recording telemetry events.
// This matches the telemetry.Recorder interface used by Consumer.
type TelemetryRecorder interface {
	RecordDLQMessage(ctx context.Context, stream, consumer string)
	RecordDLQPublishFailure(ctx context.Context, stream, consumer string)
}

// NewAdvisoryDLQHandler creates a new advisory-based DLQ handler.
func NewAdvisoryDLQHandler(
	js nats.JetStreamContext,
	nc *nats.Conn,
	streamName string,
	consumerName string,
	dlqSubject string,
	logger *zap.Logger,
	telemetry TelemetryRecorder,
) (*AdvisoryDLQHandler, error) {
	return &AdvisoryDLQHandler{
		js:           js,
		nc:           nc,
		streamName:   streamName,
		consumerName: consumerName,
		dlqSubject:   dlqSubject,
		logger:       logger,
		telemetry:    telemetry,
	}, nil
}

// Start begins listening for advisory messages and routing failed messages to DLQ.
func (h *AdvisoryDLQHandler) Start(ctx context.Context) error {
	// Subscribe to advisory subject pattern
	// Format: $JS.EVENT.ADVISORY.CONSUMER.MAX_DELIVERIES.<STREAM>.<CONSUMER>
	advisorySubject := fmt.Sprintf("$JS.EVENT.ADVISORY.CONSUMER.MAX_DELIVERIES.%s.%s",
		h.streamName, h.consumerName)

	h.logger.Info("Starting advisory DLQ handler",
		zap.String("advisory_subject", advisorySubject),
		zap.String("stream", h.streamName),
		zap.String("consumer", h.consumerName),
		zap.String("dlq_subject", h.dlqSubject),
	)

	sub, err := h.nc.Subscribe(advisorySubject, func(msg *nats.Msg) {
		h.handleAdvisory(ctx, msg)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to advisory subject: %w", err)
	}

	// Wait for context cancellation
	go func() {
		<-ctx.Done()
		sub.Unsubscribe()
		h.logger.Info("Stopped advisory DLQ handler")
	}()

	return nil
}

// handleAdvisory processes an advisory message about max deliveries.
func (h *AdvisoryDLQHandler) handleAdvisory(ctx context.Context, advisoryMsg *nats.Msg) {
	var event MaxDeliveriesAdvisoryEvent
	if err := json.Unmarshal(advisoryMsg.Data, &event); err != nil {
		h.logger.Error("Failed to unmarshal advisory event",
			zap.Error(err),
			zap.String("data", string(advisoryMsg.Data)),
		)
		return
	}

	h.logger.Warn("Message reached MaxDeliver attempts",
		zap.String("stream", event.Stream),
		zap.String("consumer", event.Consumer),
		zap.Uint64("stream_seq", event.StreamSeq),
		zap.Uint64("deliveries", event.Deliveries),
	)

	// Retrieve the original message from the stream using GetMsg API
	originalMsg, err := h.js.GetMsg(h.streamName, event.StreamSeq)
	if err != nil {
		h.logger.Error("Failed to retrieve original message from stream",
			zap.Uint64("stream_seq", event.StreamSeq),
			zap.Error(err),
		)
		h.telemetry.RecordDLQPublishFailure(ctx, h.streamName, h.consumerName)
		return
	}

	// Extract message metadata
	msgID := ""
	subject := originalMsg.Subject
	if originalMsg.Header != nil {
		msgID = originalMsg.Header.Get("Nats-Msg-Id")
	}

	// Create enriched DLQ payload (similar to existing routeToDLQ)
	payload := map[string]interface{}{
		"transport_msg_id": msgID,
		"subject":          subject,
		"stream":           h.streamName,
		"consumer":         h.consumerName,
		"nats_sequence":    event.StreamSeq,
		"deliveries":       event.Deliveries,
		"error":            fmt.Sprintf("message exhausted max_deliver (%d) attempts", event.Deliveries),
		"received_at":      time.Now().UTC(),
		"body":             string(originalMsg.Data),
		"advisory_source":  true, // Flag to distinguish from immediate DLQ
	}

	data, err := json.Marshal(payload)
	if err != nil {
		h.logger.Error("Failed to marshal advisory DLQ payload", zap.Error(err))
		h.telemetry.RecordDLQPublishFailure(ctx, h.streamName, h.consumerName)
		return
	}

	// Publish to DLQ
	if _, err := h.js.Publish(h.dlqSubject, data); err != nil {
		h.logger.Error("Failed to publish advisory message to DLQ",
			zap.Uint64("stream_seq", event.StreamSeq),
			zap.Error(err),
		)
		h.telemetry.RecordDLQPublishFailure(ctx, h.streamName, h.consumerName)
		return
	}

	h.logger.Info("Routed max-deliveries message to DLQ",
		zap.Uint64("stream_seq", event.StreamSeq),
		zap.Uint64("deliveries", event.Deliveries),
	)
	h.telemetry.RecordDLQMessage(ctx, h.streamName, h.consumerName)
}
