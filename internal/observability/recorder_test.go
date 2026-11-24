package observability

import (
	"caatsm/internal/config"
	"context"

	"github.com/prometheus/client_golang/prometheus/testutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Recorder", func() {
	BeforeEach(func() {
		resetMetricsForTest()
	})

	It("does not double-count publisher failure stage when RecordPublishFailure is called", func() {
		rec := NewRecorder(&config.Config{
			Monitoring: config.MonitoringConfig{Disabled: false, EnableMetrics: true},
		})
		rec.RecordPublishFailure(context.Background(), "ARR")
		rec.RecordFailure("publisher")

		Expect(testutil.ToFloat64(publishFailuresTotal.WithLabelValues("arr"))).To(BeNumerically("==", 1))
		Expect(testutil.ToFloat64(failureCounter.WithLabelValues("publisher"))).To(BeNumerically("==", 1))
	})

	It("returns noop recorder when telemetry and metrics are disabled", func() {
		rec := NewRecorder(&config.Config{
			Monitoring: config.MonitoringConfig{Disabled: true},
			Telemetry:  config.TelemetryConfig{Enabled: false},
		})
		_, isNoop := rec.(*noopRecorder)
		Expect(isNoop).To(BeTrue())
	})
})
