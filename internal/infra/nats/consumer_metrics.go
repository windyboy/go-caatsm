package nats

import (
	obsmetrics "caatsm/internal/infra/metrics"
	"context"
	"time"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

// initMetrics initializes OpenTelemetry metrics.
func (c *Consumer) initMetrics() {
	meter := otel.Meter("caatsm/nats")
	c.meter = meter

	if hist, err := meter.Int64Histogram("caatsm_nats_consumer_ack_pending"); err == nil {
		c.ackPending = hist
	}
	if hist, err := meter.Int64Histogram("caatsm_nats_consumer_redelivered"); err == nil {
		c.redelivered = hist
	}
	if hist, err := meter.Int64Histogram("caatsm_nats_consumer_pending"); err == nil {
		c.pending = hist
	}
	if hist, err := meter.Int64Histogram("caatsm_nats_consumer_delivered"); err == nil {
		c.delivered = hist
	}
}

// recordConsumerMetrics records consumer metrics from ConsumerInfo.
func (c *Consumer) recordConsumerMetrics(ctx context.Context, info *nats.ConsumerInfo) {
	if info == nil {
		return
	}
	if c.ackPending != nil {
		c.ackPending.Record(ctx, int64(info.NumAckPending))
	}
	if c.redelivered != nil {
		c.redelivered.Record(ctx, int64(info.NumRedelivered))
	}
	if c.pending != nil {
		c.pending.Record(ctx, int64(info.NumPending))
	}
	if c.delivered != nil {
		c.delivered.Record(ctx, int64(info.Delivered.Stream))
	}

	// Export an explicit pending messages gauge for Prometheus-based lag /
	// backlog alerts.
	obsmetrics.RecordNATSConsumerPending(c.config.streamName, c.config.consumerName, info.NumPending)
}

// emitConsumerStats periodically emits consumer statistics.
func (c *Consumer) emitConsumerStats(ctx context.Context) {
	ticker := time.NewTicker(c.config.monitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			info, err := c.js.ConsumerInfo(c.config.streamName, c.config.consumerName)
			if err != nil {
				c.logger.Warn("Failed to fetch consumer info", zap.Error(err))
				continue
			}

			c.logger.Debug("JetStream consumer metrics",
				zap.String("stream", c.config.streamName),
				zap.String("consumer", c.config.consumerName),
				zap.Uint64("num_ack_pending", uint64(info.NumAckPending)),
				zap.Uint64("num_redelivered", uint64(info.NumRedelivered)),
				zap.Uint64("num_pending", uint64(info.NumPending)),
				zap.Uint64("delivered_consumer_seq", uint64(info.Delivered.Consumer)),
				zap.Uint64("delivered_stream_seq", uint64(info.Delivered.Stream)),
			)
			c.recordConsumerMetrics(ctx, info)
		}
	}
}
