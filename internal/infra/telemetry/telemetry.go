package telemetry

import (
	"caatsm/internal/infra/config"
	obsmetrics "caatsm/internal/infra/metrics"
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Recorder provides a thin abstraction over telemetry backends (OpenTelemetry,
// Prometheus, etc.) so that application code does not need to import concrete
// metric libraries directly.
type Recorder interface {
	// RecordProcessingResult captures the final processing status of a telegram
	// along with the parser latency.
	RecordProcessingResult(ctx context.Context, status, category string, parseLatency time.Duration)

	// RecordPublishFailure increments the publish failure counter for the given
	// category.
	RecordPublishFailure(ctx context.Context, category string)

	// RecordFailure records a high-level failure bucket (parser, repository,
	// publisher, etc.).
	RecordFailure(stage string)

	// RecordMessageHandled tracks end-to-end message handling for a particular
	// stream/consumer pair.
	RecordMessageHandled(ctx context.Context, stream, consumer, result string, elapsed time.Duration)

	// RecordRetry records a retry (negative acknowledgement) reason.
	RecordRetry(ctx context.Context, stream, consumer, reason string)

	// RecordDLQMessage records a successfully routed DLQ message.
	RecordDLQMessage(ctx context.Context, stream, consumer string)

	// RecordDLQPublishFailure records a DLQ publish failure.
	RecordDLQPublishFailure(ctx context.Context, stream, consumer string)

	// RecordJSAPICall records a JetStream API call.
	RecordJSAPICall(operation string)

	// RecordAFTNValidationError records an AFTN protocol validation failure.
	RecordAFTNValidationError(ctx context.Context, errorType string)
}

// ProvideRecorder wires a composite Recorder based on configuration flags.
// - When telemetry is enabled, an OpenTelemetry-backed recorder is included.
// - When metrics are enabled, a Prometheus-backed recorder is included.
// - When neither is enabled, a noop recorder is returned.
func ProvideRecorder(cfg *config.Config) Recorder {
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

// noopRecorder implements Recorder but performs no operations.
type noopRecorder struct{}

func NewNoop() Recorder {
	return &noopRecorder{}
}

func (n *noopRecorder) RecordProcessingResult(ctx context.Context, status, category string, parseLatency time.Duration) {
}

func (n *noopRecorder) RecordPublishFailure(ctx context.Context, category string) {
}

func (n *noopRecorder) RecordFailure(stage string) {
}

func (n *noopRecorder) RecordMessageHandled(ctx context.Context, stream, consumer, result string, elapsed time.Duration) {
}

func (n *noopRecorder) RecordRetry(ctx context.Context, stream, consumer, reason string) {
}

func (n *noopRecorder) RecordDLQMessage(ctx context.Context, stream, consumer string) {
}

func (n *noopRecorder) RecordDLQPublishFailure(ctx context.Context, stream, consumer string) {
}

func (n *noopRecorder) RecordJSAPICall(operation string) {
}

func (n *noopRecorder) RecordAFTNValidationError(ctx context.Context, errorType string) {
}

// compositeRecorder fans out all calls to a slice of underlying recorders.
type compositeRecorder struct {
	recorders []Recorder
}

func NewComposite(recorders ...Recorder) Recorder {
	// Filter out nils defensively.
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

func (c *compositeRecorder) RecordProcessingResult(ctx context.Context, status, category string, parseLatency time.Duration) {
	for _, r := range c.recorders {
		r.RecordProcessingResult(ctx, status, category, parseLatency)
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

// promRecorder delegates to the Prometheus metrics helpers in the
// internal/infra/metrics package.
type promRecorder struct{}

func newPromRecorder() Recorder {
	return &promRecorder{}
}

func (p *promRecorder) RecordProcessingResult(ctx context.Context, status, category string, parseLatency time.Duration) {
	if parseLatency < 0 {
		parseLatency = 0
	}
	obsmetrics.RecordProcessed(status, category, parseLatency)
}

func (p *promRecorder) RecordPublishFailure(ctx context.Context, category string) {
	// Record publish failure with category label for detailed tracking.
	obsmetrics.RecordPublishFailure(category)
	// Also record to legacy failure counter for backward compatibility.
	obsmetrics.RecordFailure("publisher")
}

func (p *promRecorder) RecordFailure(stage string) {
	obsmetrics.RecordFailure(stage)
}

func (p *promRecorder) RecordMessageHandled(ctx context.Context, stream, consumer, result string, elapsed time.Duration) {
	obsmetrics.RecordMessageHandled(stream, consumer, result, elapsed)
}

func (p *promRecorder) RecordRetry(ctx context.Context, stream, consumer, reason string) {
	obsmetrics.RecordRetry(stream, consumer, reason)
}

func (p *promRecorder) RecordDLQMessage(ctx context.Context, stream, consumer string) {
	obsmetrics.RecordDLQMessage(stream, consumer)
}

func (p *promRecorder) RecordDLQPublishFailure(ctx context.Context, stream, consumer string) {
	obsmetrics.RecordDLQPublishFailure(stream, consumer)
}

func (p *promRecorder) RecordJSAPICall(operation string) {
	obsmetrics.RecordJSAPICall(operation)
}

func (p *promRecorder) RecordAFTNValidationError(ctx context.Context, errorType string) {
	obsmetrics.RecordAFTNValidationError(ctx, errorType)
}

// otelRecorder creates and records OpenTelemetry metrics for the CAATSM
// processor. It intentionally focuses on a small set of high-value metrics to
// avoid duplicating the full Prometheus surface.
type otelRecorder struct {
	meter metric.Meter

	messageStatusAttrKey   attribute.Key
	messageCategoryAttrKey attribute.Key

	messageProcessedCounter   metric.Int64Counter
	messagePublishFailCounter metric.Int64Counter
	parseLatencyHistogram     metric.Float64Histogram
}

func newOTelRecorder() Recorder {
	meter := otel.Meter("caatsm/app")

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

func (o *otelRecorder) RecordProcessingResult(ctx context.Context, status, category string, parseLatency time.Duration) {
	if parseLatency < 0 {
		parseLatency = 0
	}
	o.messageProcessedCounter.Add(ctx, 1,
		metric.WithAttributes(
			o.messageStatusAttrKey.String(status),
			o.messageCategoryAttrKey.String(category),
		),
	)
	o.parseLatencyHistogram.Record(ctx, parseLatency.Seconds(),
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

func (o *otelRecorder) RecordFailure(stage string) {
	// OpenTelemetry does not currently publish a dedicated failure counter; the
	// Prometheus surface captures this. This method is a no-op here.
}

func (o *otelRecorder) RecordMessageHandled(ctx context.Context, stream, consumer, result string, elapsed time.Duration) {
	// High-cardinality stream/consumer labels are exposed via Prometheus
	// metrics; OTEL can rely on traces and existing consumer metrics.
}

func (o *otelRecorder) RecordRetry(ctx context.Context, stream, consumer, reason string) {
}

func (o *otelRecorder) RecordDLQMessage(ctx context.Context, stream, consumer string) {
}

func (o *otelRecorder) RecordDLQPublishFailure(ctx context.Context, stream, consumer string) {
}

func (o *otelRecorder) RecordJSAPICall(operation string) {
}

func (o *otelRecorder) RecordAFTNValidationError(ctx context.Context, errorType string) {
	// AFTN validation metrics are primarily tracked via Prometheus.
	// This is a no-op for OTEL recorder.
}


