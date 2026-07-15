package metrics

import (
	"context"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metric and label key/value contracts for the CAATSM receiver. Centralising these
// names avoids scattering magic strings across the codebase and keeps PromQL and
// documentation aligned with the implementation.
const (
	// Metric names.
	MetricProcessedTotal        = "caatsm_processed_total"
	MetricFailuresTotal         = "caatsm_failures_total"
	MetricParseLatencySeconds   = "caatsm_parse_latency_seconds"
	MetricMessagesTotal         = "caatsm_messages_total"
	MetricHandleLatencySeconds  = "caatsm_handle_latency_seconds"
	MetricRetriesTotal          = "caatsm_retries_total"
	MetricJSAPICallsTotal       = "caatsm_js_api_calls_total"
	MetricDBQueriesTotal        = "caatsm_db_queries_total"
	MetricDBQueryLatencySeconds = "caatsm_db_query_latency_seconds"
	MetricDLQMessagesTotal            = "caatsm_dlq_messages_total"
	MetricDLQPublishFailures          = "caatsm_dlq_publish_failures_total"
	MetricPublishFailuresTotal        = "caatsm_publish_failures_total"
	MetricNATSConsumerPending         = "caatsm_nats_consumer_pending_messages"
	MetricAFTNValidationErrorsTotal   = "caatsm_aftn_validation_errors_total"
	MetricMessageGapSeconds           = "caatsm_message_gap_seconds"
	MetricMessageSequenceGapTotal     = "caatsm_message_sequence_gap_total"
	MetricConsumerHealthy             = "caatsm_consumer_healthy"

	// Common label keys.
	LabelStatus    = "status"
	LabelCategory  = "category"
	LabelStage     = "stage"
	LabelStream    = "stream"
	LabelConsumer  = "consumer"
	LabelResult    = "result"
	LabelReason    = "reason"
	LabelOperation = "operation"
	LabelErrorType = "error_type"

	// Standard result label values for caatsm_messages_total.
	ResultOK            = "ok"
	ResultFail          = "fail"
	ResultPermanentFail = "permanent_fail"

	// Standard result values for DB operations.
	DBResultOK    = "ok"
	DBResultError = "error"

	// Standard retry reasons.
	RetryReasonProcessorError = "processor_error"
)

// MetricSerialReaderHealthy is a deprecated alias for MetricConsumerHealthy.
// It is retained for backward compatibility and will be removed in a future release.
var MetricSerialReaderHealthy = MetricConsumerHealthy

var (
	once sync.Once

	registry *prometheus.Registry

	// Legacy metrics (kept for backward compatibility).
	processedCounter *prometheus.CounterVec
	failureCounter   *prometheus.CounterVec
	parseLatency     *prometheus.HistogramVec

	// Message handling metrics (per stream / consumer).
	messagesTotal      *prometheus.CounterVec
	handleLatency      *prometheus.HistogramVec
	retriesTotal       *prometheus.CounterVec
	jsAPICallsTotal    *prometheus.CounterVec
	dlqMessagesTotal   *prometheus.CounterVec
	dlqPublishFailures *prometheus.CounterVec
	publishFailuresTotal *prometheus.CounterVec

	// Database metrics.
	dbQueriesTotal *prometheus.CounterVec
	dbQueryLatency *prometheus.HistogramVec

	// NATS consumer lag metrics.
	natsConsumerPending *prometheus.GaugeVec

	// AFTN validation and health metrics.
	aftnValidationErrorsTotal *prometheus.CounterVec
	messageGapSeconds         *prometheus.GaugeVec
	messageSequenceGapTotal   *prometheus.CounterVec
	serialReaderHealthy       *prometheus.GaugeVec
)

func initCollectors() {
	registry = prometheus.NewRegistry()

	// Legacy metrics.
	processedCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: MetricProcessedTotal,
		Help: "Count of telegrams processed by status and category.",
	}, []string{LabelStatus, LabelCategory})

	failureCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: MetricFailuresTotal,
		Help: "Count of processor failures by stage (parser, repository, publisher).",
	}, []string{LabelStage})

	parseLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    MetricParseLatencySeconds,
		Help:    "Latency between reception and parse completion.",
		Buckets: prometheus.DefBuckets,
	}, []string{LabelStatus, LabelCategory})

	// New message handling metrics.
	messagesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: MetricMessagesTotal,
		Help: "Total number of messages handled by the receiver, labelled by stream, consumer and result.",
	}, []string{LabelStream, LabelConsumer, LabelResult})

	handleLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    MetricHandleLatencySeconds,
		Help:    "Latency of end-to-end message handling in seconds, from NATS receive to handler completion.",
		Buckets: prometheus.DefBuckets,
	}, []string{LabelStream, LabelConsumer})

	retriesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: MetricRetriesTotal,
		Help: "Total number of message retries (negative acknowledgements), labelled by stream, consumer and reason.",
	}, []string{LabelStream, LabelConsumer, LabelReason})

	dlqMessagesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: MetricDLQMessagesTotal,
		Help: "Total number of messages routed to the DLQ, labelled by stream and consumer.",
	}, []string{LabelStream, LabelConsumer})

	dlqPublishFailures = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: MetricDLQPublishFailures,
		Help: "Total number of failures when publishing to the DLQ, labelled by stream and consumer.",
	}, []string{LabelStream, LabelConsumer})

	publishFailuresTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: MetricPublishFailuresTotal,
		Help: "Total number of telegram publish failures, labelled by category.",
	}, []string{LabelCategory})

	jsAPICallsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: MetricJSAPICallsTotal,
		Help: "Count of JetStream API calls made by the receiver.",
	}, []string{LabelOperation})

	// Database metrics.
	dbQueriesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: MetricDBQueriesTotal,
		Help: "Total number of database operations, labelled by operation and result.",
	}, []string{LabelOperation, LabelResult})

	dbQueryLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    MetricDBQueryLatencySeconds,
		Help:    "Latency of database operations in seconds, labelled by operation.",
		Buckets: prometheus.DefBuckets,
	}, []string{LabelOperation})

	natsConsumerPending = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: MetricNATSConsumerPending,
		Help: "Approximate number of pending messages for a JetStream consumer, labelled by stream and consumer.",
	}, []string{LabelStream, LabelConsumer})

	// AFTN validation and health metrics.
	aftnValidationErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: MetricAFTNValidationErrorsTotal,
		Help: "Total number of AFTN protocol validation errors, labelled by error type.",
	}, []string{LabelErrorType})

	messageGapSeconds = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: MetricMessageGapSeconds,
		Help: "Time in seconds since the last message was received from the JetStream consumer.",
	}, []string{LabelStream, LabelConsumer})

	messageSequenceGapTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: MetricMessageSequenceGapTotal,
		Help: "Total number of message sequence gaps detected (missing sequence numbers).",
	}, []string{LabelStream, LabelConsumer})

	serialReaderHealthy = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: MetricSerialReaderHealthy,
		Help: "JetStream consumer health status: 1 = healthy (messages flowing), 0 = stalled (no messages).",
	}, []string{LabelStream, LabelConsumer})

	registry.MustRegister(
		processedCounter,
		failureCounter,
		parseLatency,
		messagesTotal,
		handleLatency,
		retriesTotal,
		jsAPICallsTotal,
		dlqMessagesTotal,
		dlqPublishFailures,
		publishFailuresTotal,
		dbQueriesTotal,
		dbQueryLatency,
		natsConsumerPending,
		aftnValidationErrorsTotal,
		messageGapSeconds,
		messageSequenceGapTotal,
		serialReaderHealthy,
	)
}

func ensureCollectors() {
	once.Do(initCollectors)
}

// Handler exposes the Prometheus metrics registry.
func Handler() http.Handler {
	ensureCollectors()
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
}

// RecordProcessed tracks the final status of a telegram along with the parse latency.
func RecordProcessed(status, category string, elapsed time.Duration) {
	ensureCollectors()
	processedCounter.WithLabelValues(labelValue(status), labelValue(category)).Inc()
	seconds := math.Max(elapsed.Seconds(), 0)
	parseLatency.WithLabelValues(labelValue(status), labelValue(category)).Observe(seconds)
}

// RecordFailure increments the failure counter for the supplied stage.
func RecordFailure(stage string) {
	ensureCollectors()
	failureCounter.WithLabelValues(labelValue(stage)).Inc()
}

// RecordMessageHandled records end-to-end message handling metrics (per stream / consumer).
// Result is expected to be values such as "ok", "fail", or "retry".
func RecordMessageHandled(stream, consumer, result string, elapsed time.Duration) {
	ensureCollectors()
	if elapsed < 0 {
		elapsed = 0
	}
	messagesTotal.WithLabelValues(labelValue(stream), labelValue(consumer), labelValue(result)).Inc()
	handleLatency.WithLabelValues(labelValue(stream), labelValue(consumer)).Observe(elapsed.Seconds())
}

// RecordRetry increments the retry counter for a message that is being negatively acknowledged.
// Reason can capture the high-level cause, e.g. "processor_error" or "nats_timeout".
func RecordRetry(stream, consumer, reason string) {
	ensureCollectors()
	retriesTotal.WithLabelValues(labelValue(stream), labelValue(consumer), labelValue(reason)).Inc()
}

// RecordDLQMessage increments the DLQ message counter for a successfully routed message.
func RecordDLQMessage(stream, consumer string) {
	ensureCollectors()
	dlqMessagesTotal.WithLabelValues(labelValue(stream), labelValue(consumer)).Inc()
}

// RecordDLQPublishFailure increments the DLQ publish failure counter when a DLQ
// publish attempt fails.
func RecordDLQPublishFailure(stream, consumer string) {
	ensureCollectors()
	dlqPublishFailures.WithLabelValues(labelValue(stream), labelValue(consumer)).Inc()
}

// RecordPublishFailure increments the publish failure counter for the given category.
// This tracks general publish failures (not DLQ-specific).
func RecordPublishFailure(category string) {
	ensureCollectors()
	publishFailuresTotal.WithLabelValues(labelValue(category)).Inc()
}

// RecordDBQuery records metrics for a single database operation.
// Operation examples: "insert_one", "insert_batch", "insert_raw".
// Result is usually "ok" or "error".
func RecordDBQuery(operation, result string, elapsed time.Duration) {
	ensureCollectors()
	if elapsed < 0 {
		elapsed = 0
	}
	dbQueriesTotal.WithLabelValues(labelValue(operation), labelValue(result)).Inc()
	dbQueryLatency.WithLabelValues(labelValue(operation)).Observe(elapsed.Seconds())
}

// RecordJSAPICall increments the JetStream API call counter for the given operation.
func RecordJSAPICall(operation string) {
	ensureCollectors()
	jsAPICallsTotal.WithLabelValues(labelValue(operation)).Inc()
}

// RecordNATSConsumerPending records the current pending message count for a
// JetStream consumer as a gauge, enabling backlog / lag alerts.
func RecordNATSConsumerPending(stream, consumer string, pending uint64) {
	ensureCollectors()
	
	// Validate inputs to ensure metric is recorded correctly
	streamLabel := labelValue(stream)
	consumerLabel := labelValue(consumer)
	
	// Ensure metric is always set, even with empty labels (will be "unknown")
	if streamLabel == "" {
		streamLabel = "unknown"
	}
	if consumerLabel == "" {
		consumerLabel = "unknown"
	}
	
	// Set the metric value
	natsConsumerPending.WithLabelValues(streamLabel, consumerLabel).Set(float64(pending))
}

// RecordAFTNValidationError increments the AFTN validation error counter for the given error type.
func RecordAFTNValidationError(ctx context.Context, errorType string) {
	ensureCollectors()
	aftnValidationErrorsTotal.WithLabelValues(labelValue(errorType)).Inc()
}

// RecordMessageGap records the time gap (in seconds) since the last message was received.
func RecordMessageGap(stream, consumer string, gapSeconds float64) {
	ensureCollectors()
	messageGapSeconds.WithLabelValues(labelValue(stream), labelValue(consumer)).Set(gapSeconds)
}

// RecordSequenceGap increments the sequence gap counter when missing sequence numbers are detected.
func RecordSequenceGap(stream, consumer string, gapSize uint64) {
	ensureCollectors()
	messageSequenceGapTotal.WithLabelValues(labelValue(stream), labelValue(consumer)).Add(float64(gapSize))
}

// RecordConsumerHealth sets the JetStream consumer health status.
// healthy=1 means messages are flowing normally, healthy=0 means the consumer has stalled.
func RecordConsumerHealth(stream, consumer string, healthy bool) {
	ensureCollectors()
	value := 0.0
	if healthy {
		value = 1.0
	}
	serialReaderHealthy.WithLabelValues(labelValue(stream), labelValue(consumer)).Set(value)
}

// RecordSerialReaderHealth is a deprecated alias for RecordConsumerHealth.
// It is retained for backward compatibility and will be removed in a future release.
func RecordSerialReaderHealth(stream, consumer string, healthy bool) {
	RecordConsumerHealth(stream, consumer, healthy)
}

func labelValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	return strings.ToLower(value)
}
