package observability

import (
	"caatsm/internal/config"
	"context"
	"time"

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

	It("delegates DB and NATS monitor metrics through promRecorder", func() {
		rec := NewRecorder(&config.Config{
			Monitoring: config.MonitoringConfig{Disabled: false, EnableMetrics: true},
		})
		rec.RecordDBQuery("insert_one", DBResultOK, 5*time.Millisecond)
		rec.RecordNATSConsumerPending("TELEGRAM", "c1", 7)
		rec.RecordMessageGap("TELEGRAM", "c1", 1.5)
		rec.RecordSequenceGap("TELEGRAM", "c1", 2)
		rec.RecordSerialReaderHealth("TELEGRAM", "c1", true)

		Expect(testutil.ToFloat64(dbQueriesTotal.WithLabelValues("insert_one", "ok"))).To(BeNumerically("==", 1))
		Expect(testutil.ToFloat64(natsConsumerPending.WithLabelValues("telegram", "c1"))).To(BeNumerically("==", 7))
		Expect(testutil.ToFloat64(messageGapSeconds.WithLabelValues("telegram", "c1"))).To(BeNumerically("==", 1.5))
		Expect(testutil.ToFloat64(messageSequenceGapTotal.WithLabelValues("telegram", "c1"))).To(BeNumerically("==", 2))
		Expect(testutil.ToFloat64(serialReaderHealthy.WithLabelValues("telegram", "c1"))).To(BeNumerically("==", 1))
	})

	It("returns noop recorder when telemetry and metrics are disabled", func() {
		rec := NewRecorder(&config.Config{
			Monitoring: config.MonitoringConfig{Disabled: true},
			Telemetry:  config.TelemetryConfig{Enabled: false},
		})
		_, isNoop := rec.(*noopRecorder)
		Expect(isNoop).To(BeTrue())
		rec.RecordDBQuery("insert_one", DBResultOK, time.Millisecond)
		rec.RecordNATSConsumerPending("s", "c", 1)
	})
})
