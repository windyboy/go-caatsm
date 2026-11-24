package observability

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

// Metric and label key/value contracts for the CAATSM receiver.
const (
	MetricProcessedTotal            = "caatsm_processed_total"
	MetricFailuresTotal             = "caatsm_failures_total"
	MetricParseLatencySeconds       = "caatsm_parse_latency_seconds"
	MetricMessagesTotal             = "caatsm_messages_total"
	MetricHandleLatencySeconds      = "caatsm_handle_latency_seconds"
	MetricRetriesTotal              = "caatsm_retries_total"
	MetricJSAPICallsTotal           = "caatsm_js_api_calls_total"
	MetricDBQueriesTotal            = "caatsm_db_queries_total"
	MetricDBQueryLatencySeconds     = "caatsm_db_query_latency_seconds"
	MetricDLQMessagesTotal          = "caatsm_dlq_messages_total"
	MetricDLQPublishFailures        = "caatsm_dlq_publish_failures_total"
	MetricPublishFailuresTotal      = "caatsm_publish_failures_total"
	MetricNATSConsumerPending       = "caatsm_nats_consumer_pending_messages"
	MetricAFTNValidationErrorsTotal = "caatsm_aftn_validation_errors_total"
	MetricMessageGapSeconds         = "caatsm_message_gap_seconds"
	MetricMessageSequenceGapTotal   = "caatsm_message_sequence_gap_total"
	MetricSerialReaderHealthy       = "caatsm_serial_reader_healthy"

	LabelStatus    = "status"
	LabelCategory  = "category"
	LabelStage     = "stage"
	LabelStream    = "stream"
	LabelConsumer  = "consumer"
	LabelResult    = "result"
	LabelReason    = "reason"
	LabelOperation = "operation"
	LabelErrorType = "error_type"

	ResultOK            = "ok"
	ResultFail          = "fail"
	ResultPermanentFail = "permanent_fail"

	DBResultOK    = "ok"
	DBResultError = "error"

	RetryReasonProcessorError = "processor_error"
	RetryReasonDLQError       = "dlq_publish_error"
)

var (
	once     sync.Once
	registry *prometheus.Registry

	processedCounter          *prometheus.CounterVec
	failureCounter            *prometheus.CounterVec
	parseLatency              *prometheus.HistogramVec
	messagesTotal             *prometheus.CounterVec
	handleLatency             *prometheus.HistogramVec
	retriesTotal              *prometheus.CounterVec
	jsAPICallsTotal           *prometheus.CounterVec
	dlqMessagesTotal          *prometheus.CounterVec
	dlqPublishFailures        *prometheus.CounterVec
	publishFailuresTotal      *prometheus.CounterVec
	dbQueriesTotal            *prometheus.CounterVec
	dbQueryLatency            *prometheus.HistogramVec
	natsConsumerPending       *prometheus.GaugeVec
	aftnValidationErrorsTotal *prometheus.CounterVec
	messageGapSeconds         *prometheus.GaugeVec
	messageSequenceGapTotal   *prometheus.CounterVec
	serialReaderHealthy       *prometheus.GaugeVec
)

func initCollectors() {
	registry = prometheus.NewRegistry()

	processedCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: MetricProcessedTotal,
		Help: "Count of telegrams processed by status and category.",
	}, []string{LabelStatus, LabelCategory})

	failureCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: MetricFailuresTotal,
		Help: "Count of processor failures by stage (parser, aftn_validator, repository, publisher).",
	}, []string{LabelStage})

	parseLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    MetricParseLatencySeconds,
		Help:    "Latency between reception and processing completion.",
		Buckets: prometheus.DefBuckets,
	}, []string{LabelStatus, LabelCategory})

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

	aftnValidationErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: MetricAFTNValidationErrorsTotal,
		Help: "Total number of AFTN protocol validation errors, labelled by error type.",
	}, []string{LabelErrorType})

	messageGapSeconds = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: MetricMessageGapSeconds,
		Help: "Time in seconds since the last message was received from the serial reader.",
	}, []string{LabelStream, LabelConsumer})

	messageSequenceGapTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: MetricMessageSequenceGapTotal,
		Help: "Total number of message sequence gaps detected (missing sequence numbers).",
	}, []string{LabelStream, LabelConsumer})

	serialReaderHealthy = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: MetricSerialReaderHealthy,
		Help: "Serial reader health status: 1 = healthy (messages flowing), 0 = stalled (no messages).",
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

// MetricsHandler exposes the Prometheus metrics registry.
func MetricsHandler() http.Handler {
	ensureCollectors()
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
}

func RecordProcessed(status, category string, elapsed time.Duration) {
	ensureCollectors()
	processedCounter.WithLabelValues(labelValue(status), labelValue(category)).Inc()
	seconds := math.Max(elapsed.Seconds(), 0)
	parseLatency.WithLabelValues(labelValue(status), labelValue(category)).Observe(seconds)
}

func RecordFailure(stage string) {
	ensureCollectors()
	failureCounter.WithLabelValues(labelValue(stage)).Inc()
}

func RecordMessageHandled(stream, consumer, result string, elapsed time.Duration) {
	ensureCollectors()
	if elapsed < 0 {
		elapsed = 0
	}
	messagesTotal.WithLabelValues(labelValue(stream), labelValue(consumer), labelValue(result)).Inc()
	handleLatency.WithLabelValues(labelValue(stream), labelValue(consumer)).Observe(elapsed.Seconds())
}

func RecordRetry(stream, consumer, reason string) {
	ensureCollectors()
	retriesTotal.WithLabelValues(labelValue(stream), labelValue(consumer), labelValue(reason)).Inc()
}

func RecordDLQMessage(stream, consumer string) {
	ensureCollectors()
	dlqMessagesTotal.WithLabelValues(labelValue(stream), labelValue(consumer)).Inc()
}

func RecordDLQPublishFailure(stream, consumer string) {
	ensureCollectors()
	dlqPublishFailures.WithLabelValues(labelValue(stream), labelValue(consumer)).Inc()
}

func RecordPublishFailure(category string) {
	ensureCollectors()
	publishFailuresTotal.WithLabelValues(labelValue(category)).Inc()
}

func RecordDBQuery(operation, result string, elapsed time.Duration) {
	ensureCollectors()
	if elapsed < 0 {
		elapsed = 0
	}
	dbQueriesTotal.WithLabelValues(labelValue(operation), labelValue(result)).Inc()
	dbQueryLatency.WithLabelValues(labelValue(operation)).Observe(elapsed.Seconds())
}

func RecordJSAPICall(operation string) {
	ensureCollectors()
	jsAPICallsTotal.WithLabelValues(labelValue(operation)).Inc()
}

func RecordNATSConsumerPending(stream, consumer string, pending uint64) {
	ensureCollectors()
	natsConsumerPending.WithLabelValues(labelValue(stream), labelValue(consumer)).Set(float64(pending))
}

func RecordAFTNValidationError(_ context.Context, errorType string) {
	ensureCollectors()
	aftnValidationErrorsTotal.WithLabelValues(labelValue(errorType)).Inc()
}

func RecordMessageGap(stream, consumer string, gapSeconds float64) {
	ensureCollectors()
	messageGapSeconds.WithLabelValues(labelValue(stream), labelValue(consumer)).Set(gapSeconds)
}

func RecordSequenceGap(stream, consumer string, gapSize uint64) {
	ensureCollectors()
	messageSequenceGapTotal.WithLabelValues(labelValue(stream), labelValue(consumer)).Add(float64(gapSize))
}

func RecordSerialReaderHealth(stream, consumer string, healthy bool) {
	ensureCollectors()
	value := 0.0
	if healthy {
		value = 1.0
	}
	serialReaderHealthy.WithLabelValues(labelValue(stream), labelValue(consumer)).Set(value)
}

func labelValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	return strings.ToLower(value)
}
