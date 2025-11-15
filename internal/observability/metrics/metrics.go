package metrics

import (
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	once             sync.Once
	registry         *prometheus.Registry
	processedCounter *prometheus.CounterVec
	failureCounter   *prometheus.CounterVec
	parseLatency     *prometheus.HistogramVec
)

func initCollectors() {
	registry = prometheus.NewRegistry()
	processedCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "caatsm_processed_total",
		Help: "Count of telegrams processed by status and category.",
	}, []string{"status", "category"})
	failureCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "caatsm_failures_total",
		Help: "Count of processor failures by stage (parser, repository, publisher).",
	}, []string{"stage"})
	parseLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "caatsm_parse_latency_seconds",
		Help:    "Latency between reception and parse completion.",
		Buckets: prometheus.DefBuckets,
	}, []string{"status", "category"})

	registry.MustRegister(processedCounter, failureCounter, parseLatency)
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

func labelValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	return strings.ToLower(value)
}
