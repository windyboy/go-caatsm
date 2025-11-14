package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP metrics
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	// Message processing metrics
	MessagesProcessedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messages_processed_total",
			Help: "Total number of messages processed",
		},
		[]string{"status", "category"},
	)

	MessageProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "message_processing_duration_seconds",
			Help:    "Message processing duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"category"},
	)

	MessageProcessingFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "message_processing_failures_total",
			Help: "Number of message processing failures, labeled by stage",
		},
		[]string{"stage"},
	)

	MessageRetriesTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "message_processing_retries_total",
			Help: "Total number of message retry attempts",
		},
	)

	// Batch/worker metrics
	WorkerBatchDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "worker_batch_duration_seconds",
			Help:    "Duration of worker batch flush operations",
			Buckets: prometheus.DefBuckets,
		},
	)

	// Database metrics
	DatabaseQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "database_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "table"},
	)

	DatabaseConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "database_connections_active",
			Help: "Number of active database connections",
		},
	)

	// NATS metrics
	NATSMessagesPublished = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nats_messages_published_total",
			Help: "Total number of messages published to NATS",
		},
		[]string{"subject"},
	)

	NATSMessagesConsumed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nats_messages_consumed_total",
			Help: "Total number of messages consumed from NATS",
		},
		[]string{"subject"},
	)
)

