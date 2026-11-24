package observability

import (
	"caatsm/internal/config"
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Recorder provides a unified telemetry interface across Prometheus and OpenTelemetry.
type Recorder interface {
	RecordProcessingResult(ctx context.Context, status, category string, elapsed time.Duration)
	RecordPublishFailure(ctx context.Context, category string)
	RecordFailure(stage string)
	RecordMessageHandled(ctx context.Context, stream, consumer, result string, elapsed time.Duration)
	RecordRetry(ctx context.Context, stream, consumer, reason string)
	RecordDLQMessage(ctx context.Context, stream, consumer string)
	RecordDLQPublishFailure(ctx context.Context, stream, consumer string)
	RecordJSAPICall(operation string)
	RecordAFTNValidationError(ctx context.Context, errorType string)
	RecordDBQuery(operation, result string, elapsed time.Duration)
	RecordNATSConsumerPending(stream, consumer string, pending uint64)
	RecordMessageGap(stream, consumer string, gapSeconds float64)
	RecordSequenceGap(stream, consumer string, gapSize uint64)
	RecordSerialReaderHealth(stream, consumer string, healthy bool)
}

// NewRecorder wires a composite Recorder based on configuration flags.
func NewRecorder(cfg *config.Config) Recorder {
	if cfg == nil {
		return NewNoop()
	}

	var recorders []Recorder
	if cfg.Telemetry.Enabled {
		recorders = append(recorders, newOTelRecorder())
	}
	if !cfg.Monitoring.Disabled && cfg.Monitoring.EnableMetrics {
		recorders = append(recorders, newPromRecorder())
	}
	if len(recorders) == 0 {
		return NewNoop()
	}
	return NewComposite(recorders...)
}

type noopRecorder struct{}

// NewNoop returns a Recorder that performs no operations.
func NewNoop() Recorder { return &noopRecorder{} }

func (n *noopRecorder) RecordProcessingResult(context.Context, string, string, time.Duration) {}
func (n *noopRecorder) RecordPublishFailure(context.Context, string)                          {}
func (n *noopRecorder) RecordFailure(string)                                                  {}
func (n *noopRecorder) RecordMessageHandled(context.Context, string, string, string, time.Duration) {
}
func (n *noopRecorder) RecordRetry(context.Context, string, string, string)     {}
func (n *noopRecorder) RecordDLQMessage(context.Context, string, string)        {}
func (n *noopRecorder) RecordDLQPublishFailure(context.Context, string, string) {}
func (n *noopRecorder) RecordJSAPICall(string)                                  {}
func (n *noopRecorder) RecordAFTNValidationError(context.Context, string)       {}
func (n *noopRecorder) RecordDBQuery(string, string, time.Duration)             {}
func (n *noopRecorder) RecordNATSConsumerPending(string, string, uint64)        {}
func (n *noopRecorder) RecordMessageGap(string, string, float64)                {}
func (n *noopRecorder) RecordSequenceGap(string, string, uint64)                {}
func (n *noopRecorder) RecordSerialReaderHealth(string, string, bool)           {}

type compositeRecorder struct {
	recorders []Recorder
}

// NewComposite fans out all calls to a slice of underlying recorders.
func NewComposite(recorders ...Recorder) Recorder {
	var filtered []Recorder
	for _, r := range recorders {
		if r != nil {
			filtered = append(filtered, r)
		}
	}
	if len(filtered) == 0 {
		return NewNoop()
	}
	return &compositeRecorder{recorders: filtered}
}

func (c *compositeRecorder) RecordProcessingResult(ctx context.Context, status, category string, elapsed time.Duration) {
	for _, r := range c.recorders {
		r.RecordProcessingResult(ctx, status, category, elapsed)
	}
}

func (c *compositeRecorder) RecordPublishFailure(ctx context.Context, category string) {
	for _, r := range c.recorders {
		r.RecordPublishFailure(ctx, category)
	}
}

func (c *compositeRecorder) RecordFailure(stage string) {
	for _, r := range c.recorders {
		r.RecordFailure(stage)
	}
}

func (c *compositeRecorder) RecordMessageHandled(ctx context.Context, stream, consumer, result string, elapsed time.Duration) {
	for _, r := range c.recorders {
		r.RecordMessageHandled(ctx, stream, consumer, result, elapsed)
	}
}

func (c *compositeRecorder) RecordRetry(ctx context.Context, stream, consumer, reason string) {
	for _, r := range c.recorders {
		r.RecordRetry(ctx, stream, consumer, reason)
	}
}

func (c *compositeRecorder) RecordDLQMessage(ctx context.Context, stream, consumer string) {
	for _, r := range c.recorders {
		r.RecordDLQMessage(ctx, stream, consumer)
	}
}

func (c *compositeRecorder) RecordDLQPublishFailure(ctx context.Context, stream, consumer string) {
	for _, r := range c.recorders {
		r.RecordDLQPublishFailure(ctx, stream, consumer)
	}
}

func (c *compositeRecorder) RecordJSAPICall(operation string) {
	for _, r := range c.recorders {
		r.RecordJSAPICall(operation)
	}
}

func (c *compositeRecorder) RecordAFTNValidationError(ctx context.Context, errorType string) {
	for _, r := range c.recorders {
		r.RecordAFTNValidationError(ctx, errorType)
	}
}

func (c *compositeRecorder) RecordDBQuery(operation, result string, elapsed time.Duration) {
	for _, r := range c.recorders {
		r.RecordDBQuery(operation, result, elapsed)
	}
}

func (c *compositeRecorder) RecordNATSConsumerPending(stream, consumer string, pending uint64) {
	for _, r := range c.recorders {
		r.RecordNATSConsumerPending(stream, consumer, pending)
	}
}

func (c *compositeRecorder) RecordMessageGap(stream, consumer string, gapSeconds float64) {
	for _, r := range c.recorders {
		r.RecordMessageGap(stream, consumer, gapSeconds)
	}
}

func (c *compositeRecorder) RecordSequenceGap(stream, consumer string, gapSize uint64) {
	for _, r := range c.recorders {
		r.RecordSequenceGap(stream, consumer, gapSize)
	}
}

func (c *compositeRecorder) RecordSerialReaderHealth(stream, consumer string, healthy bool) {
	for _, r := range c.recorders {
		r.RecordSerialReaderHealth(stream, consumer, healthy)
	}
}

type promRecorder struct{}

func newPromRecorder() Recorder { return &promRecorder{} }

func (p *promRecorder) RecordProcessingResult(_ context.Context, status, category string, elapsed time.Duration) {
	if elapsed < 0 {
		elapsed = 0
	}
	RecordProcessed(status, category, elapsed)
}

func (p *promRecorder) RecordPublishFailure(_ context.Context, category string) {
	// Only record the category-specific publish failure counter here;
	// stage failure bucket ("publisher") is recorded separately by RecordFailure.
	RecordPublishFailure(category)
}

func (p *promRecorder) RecordFailure(stage string) {
	RecordFailure(stage)
}

func (p *promRecorder) RecordMessageHandled(_ context.Context, stream, consumer, result string, elapsed time.Duration) {
	RecordMessageHandled(stream, consumer, result, elapsed)
}

func (p *promRecorder) RecordRetry(_ context.Context, stream, consumer, reason string) {
	RecordRetry(stream, consumer, reason)
}

func (p *promRecorder) RecordDLQMessage(_ context.Context, stream, consumer string) {
	RecordDLQMessage(stream, consumer)
}

func (p *promRecorder) RecordDLQPublishFailure(_ context.Context, stream, consumer string) {
	RecordDLQPublishFailure(stream, consumer)
}

func (p *promRecorder) RecordJSAPICall(operation string) {
	RecordJSAPICall(operation)
}

func (p *promRecorder) RecordAFTNValidationError(ctx context.Context, errorType string) {
	RecordAFTNValidationError(ctx, errorType)
}

func (p *promRecorder) RecordDBQuery(operation, result string, elapsed time.Duration) {
	RecordDBQuery(operation, result, elapsed)
}

func (p *promRecorder) RecordNATSConsumerPending(stream, consumer string, pending uint64) {
	RecordNATSConsumerPending(stream, consumer, pending)
}

func (p *promRecorder) RecordMessageGap(stream, consumer string, gapSeconds float64) {
	RecordMessageGap(stream, consumer, gapSeconds)
}

func (p *promRecorder) RecordSequenceGap(stream, consumer string, gapSize uint64) {
	RecordSequenceGap(stream, consumer, gapSize)
}

func (p *promRecorder) RecordSerialReaderHealth(stream, consumer string, healthy bool) {
	RecordSerialReaderHealth(stream, consumer, healthy)
}

type otelRecorder struct {
	meter metric.Meter

	messageStatusAttrKey   attribute.Key
	messageCategoryAttrKey attribute.Key

	messageProcessedCounter   metric.Int64Counter
	messagePublishFailCounter metric.Int64Counter
	parseLatencyHistogram     metric.Float64Histogram
}

func newOTelRecorder() Recorder {
	meter := otel.Meter("caatsm/pipeline")

	statusKey := attribute.Key("message.status")
	categoryKey := attribute.Key("message.category")

	messageProcessedCounter, _ := meter.Int64Counter(
		"caatsm_messages_processed_total",
		metric.WithDescription("Total number of telegrams processed by the CAATSM processor."),
	)
	messagePublishFailCounter, _ := meter.Int64Counter(
		"caatsm_publish_failures_total",
		metric.WithDescription("Total number of telegram publish failures."),
	)
	parseLatencyHistogram, _ := meter.Float64Histogram(
		"caatsm_parse_duration_seconds",
		metric.WithDescription("Latency of parsing a telegram, in seconds."),
		metric.WithUnit("s"),
	)

	return &otelRecorder{
		meter:                     meter,
		messageStatusAttrKey:      statusKey,
		messageCategoryAttrKey:    categoryKey,
		messageProcessedCounter:   messageProcessedCounter,
		messagePublishFailCounter: messagePublishFailCounter,
		parseLatencyHistogram:     parseLatencyHistogram,
	}
}

func (o *otelRecorder) RecordProcessingResult(ctx context.Context, status, category string, elapsed time.Duration) {
	if elapsed < 0 {
		elapsed = 0
	}
	o.messageProcessedCounter.Add(ctx, 1,
		metric.WithAttributes(
			o.messageStatusAttrKey.String(status),
			o.messageCategoryAttrKey.String(category),
		),
	)
	o.parseLatencyHistogram.Record(ctx, elapsed.Seconds(),
		metric.WithAttributes(
			o.messageStatusAttrKey.String(status),
			o.messageCategoryAttrKey.String(category),
		),
	)
}

func (o *otelRecorder) RecordPublishFailure(ctx context.Context, category string) {
	o.messagePublishFailCounter.Add(ctx, 1,
		metric.WithAttributes(
			o.messageCategoryAttrKey.String(category),
		),
	)
}

func (o *otelRecorder) RecordFailure(string) {}
func (o *otelRecorder) RecordMessageHandled(context.Context, string, string, string, time.Duration) {
}
func (o *otelRecorder) RecordRetry(context.Context, string, string, string)     {}
func (o *otelRecorder) RecordDLQMessage(context.Context, string, string)        {}
func (o *otelRecorder) RecordDLQPublishFailure(context.Context, string, string) {}
func (o *otelRecorder) RecordJSAPICall(string)                                  {}
func (o *otelRecorder) RecordAFTNValidationError(context.Context, string)       {}
func (o *otelRecorder) RecordDBQuery(string, string, time.Duration)             {}
func (o *otelRecorder) RecordNATSConsumerPending(string, string, uint64)        {}
func (o *otelRecorder) RecordMessageGap(string, string, float64)                {}
func (o *otelRecorder) RecordSequenceGap(string, string, uint64)                {}
func (o *otelRecorder) RecordSerialReaderHealth(string, string, bool)           {}
