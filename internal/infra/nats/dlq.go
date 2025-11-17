package nats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// validateDLQ verifies whether DLQ routing should be enabled and, if so, whether
// the configured DLQ subject is bound to a JetStream stream. Returns an error
// if DLQ is enabled but misconfigured, allowing the caller to fail fast.
func (c *Consumer) validateDLQ() error {
	if c == nil {
		return nil
	}

	// DLQ routing is only active in JetStream mode.
	if c.mode != "jetstream" {
		return nil
	}

	// If DLQ is not enabled in config, make sure we don't accidentally route to it.
	if !c.cfg.DLQ.Enabled {
		if strings.TrimSpace(c.dlqSubject) != "" {
			c.logger.Info("DLQ subject configured but dlq.enabled is false; DLQ routing disabled",
				zap.String("dlq_subject", c.dlqSubject),
			)
		}
		c.dlqSubject = ""
		return nil
	}

	subject := strings.TrimSpace(c.dlqSubject)
	if subject == "" {
		return fmt.Errorf("DLQ enabled but dlq.subject is empty")
	}

	if c.js == nil {
		return fmt.Errorf("DLQ enabled but JetStream context is nil")
	}

	// Ensure the DLQ subject is actually bound to a JetStream stream. This avoids
	// the opaque `nats: no response from stream` error later when publishing.
	c.telemetry.RecordJSAPICall("dlq_validate_stream")
	streamName, err := c.js.StreamNameBySubject(subject)
	if err != nil || strings.TrimSpace(streamName) == "" {
		return fmt.Errorf("DLQ subject %s not bound to any JetStream stream: %w", subject, err)
	}

	c.logger.Info("DLQ configuration validated",
		zap.String("dlq_subject", subject),
		zap.String("dlq_stream", streamName),
	)

	return nil
}

// routeToDLQ publishes a copy of the failed message to the configured DLQ subject,
// including useful metadata for offline analysis. If DLQ is not configured or the
// consumer is not running in JetStream mode, this is a no-op.
func (c *Consumer) routeToDLQ(ctx context.Context, msg *nats.Msg, cause error) error {
	if c == nil || c.js == nil {
		return nil
	}
	if c.mode != "jetstream" {
		return nil
	}
	if strings.TrimSpace(c.dlqSubject) == "" {
		return nil
	}

	meta, _ := msg.Metadata()
	jsSeq := uint64(0)
	deliveries := uint64(0)
	if meta != nil {
		jsSeq = meta.Sequence.Stream
		deliveries = meta.NumDelivered
	}

	payload := map[string]interface{}{
		"transport_msg_id": msg.Header.Get("Nats-Msg-Id"),
		"subject":          msg.Subject,
		"stream":           c.streamName,
		"consumer":         c.consumerName,
		"nats_sequence":    jsSeq,
		"deliveries":       deliveries,
		"error":            fmt.Sprint(cause),
		"received_at":      time.Now().UTC(),
		"body":             string(msg.Data),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		c.logger.Error("failed to marshal DLQ payload",
			zap.String("stream", c.streamName),
			zap.String("consumer", c.consumerName),
			zap.String("dlq_subject", c.dlqSubject),
			zap.Error(err),
		)
		return fmt.Errorf("marshal dlq payload: %w", err)
	}

	if _, err := c.js.Publish(c.dlqSubject, data); err != nil {
		// nats.ErrNoResponders typically means that no JetStream stream is
		// configured to receive this subject, or JetStream is temporarily
		// unavailable. Surface this explicitly to make operational diagnosis
		// easier.
		if errors.Is(err, nats.ErrNoResponders) {
			c.logger.Error("transient DLQ publish error (no responders)",
				zap.String("stream", c.streamName),
				zap.String("consumer", c.consumerName),
				zap.String("dlq_subject", c.dlqSubject),
				zap.Int("payload_size", len(data)),
				zap.Error(err),
			)
			c.telemetry.RecordDLQPublishFailure(ctx, c.streamName, c.consumerName)
			return fmt.Errorf("publish to dlq subject %s: no JetStream stream found for subject or JetStream unavailable: %w", c.dlqSubject, err)
		}
		c.logger.Error("failed to publish to DLQ",
			zap.String("stream", c.streamName),
			zap.String("consumer", c.consumerName),
			zap.String("dlq_subject", c.dlqSubject),
			zap.Int("payload_size", len(data)),
			zap.Error(err),
		)
		c.telemetry.RecordDLQPublishFailure(ctx, c.streamName, c.consumerName)
		return fmt.Errorf("publish to dlq subject %s: %w", c.dlqSubject, err)
	}

	c.telemetry.RecordDLQMessage(ctx, c.streamName, c.consumerName)

	return nil
}
