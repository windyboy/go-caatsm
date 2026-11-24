package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Metrics", func() {
	BeforeEach(func() {
		resetMetricsForTest()
	})

	It("serves Prometheus metrics via HTTP handler", func() {
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		resp := httptest.NewRecorder()
		MetricsHandler().ServeHTTP(resp, req)
		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Header().Get("Content-Type")).To(ContainSubstring("text/plain"))
	})

	It("normalizes label values", func() {
		Expect(labelValue("   ")).To(Equal("unknown"))
		Expect(labelValue("SOME_VALUE")).To(Equal("some_value"))
	})

	It("records counters, histograms, and gauges", func() {
		RecordProcessed("PARSED", "ARR", 150*time.Millisecond)
		RecordFailure("parser")
		RecordMessageHandled("stream1", "consumer1", ResultOK, 80*time.Millisecond)
		RecordRetry("stream1", "consumer1", RetryReasonProcessorError)
		RecordDLQMessage("stream1", "consumer1")
		RecordDLQPublishFailure("stream1", "consumer1")
		RecordPublishFailure("flightPlan")
		RecordDBQuery("insert_one", DBResultOK, 10*time.Millisecond)
		RecordJSAPICall("publish")
		RecordAFTNValidationError(context.Background(), "priority_indicator")
		RecordNATSConsumerPending("STREAM_A", "consumerA", 42)
		RecordMessageGap("STREAM_A", "consumerA", 12.5)
		RecordSequenceGap("STREAM_A", "consumerA", 2)
		RecordSerialReaderHealth("STREAM_A", "consumerA", true)

		Expect(testutil.ToFloat64(processedCounter.WithLabelValues("parsed", "arr"))).To(BeNumerically("==", 1))
		Expect(testutil.CollectAndCount(parseLatency, MetricParseLatencySeconds)).To(Equal(1))
		Expect(testutil.ToFloat64(failureCounter.WithLabelValues("parser"))).To(BeNumerically("==", 1))
		Expect(testutil.ToFloat64(messagesTotal.WithLabelValues("stream1", "consumer1", ResultOK))).To(BeNumerically("==", 1))
		Expect(testutil.CollectAndCount(handleLatency, MetricHandleLatencySeconds)).To(Equal(1))
		Expect(testutil.ToFloat64(retriesTotal.WithLabelValues("stream1", "consumer1", RetryReasonProcessorError))).To(BeNumerically("==", 1))
		Expect(testutil.ToFloat64(dlqMessagesTotal.WithLabelValues("stream1", "consumer1"))).To(BeNumerically("==", 1))
		Expect(testutil.ToFloat64(dlqPublishFailures.WithLabelValues("stream1", "consumer1"))).To(BeNumerically("==", 1))
		Expect(testutil.ToFloat64(publishFailuresTotal.WithLabelValues("flightplan"))).To(BeNumerically("==", 1))
		Expect(testutil.ToFloat64(dbQueriesTotal.WithLabelValues("insert_one", DBResultOK))).To(BeNumerically("==", 1))
		Expect(testutil.CollectAndCount(dbQueryLatency, MetricDBQueryLatencySeconds)).To(Equal(1))
		Expect(testutil.ToFloat64(jsAPICallsTotal.WithLabelValues("publish"))).To(BeNumerically("==", 1))
		Expect(testutil.ToFloat64(aftnValidationErrorsTotal.WithLabelValues("priority_indicator"))).To(BeNumerically("==", 1))
		Expect(testutil.ToFloat64(natsConsumerPending.WithLabelValues("stream_a", "consumera"))).To(BeNumerically("==", 42))
		Expect(testutil.ToFloat64(messageGapSeconds.WithLabelValues("stream_a", "consumera"))).To(BeNumerically("==", 12.5))
		Expect(testutil.ToFloat64(messageSequenceGapTotal.WithLabelValues("stream_a", "consumera"))).To(BeNumerically("==", 2))
		Expect(testutil.ToFloat64(serialReaderHealthy.WithLabelValues("stream_a", "consumera"))).To(BeNumerically("==", 1))
	})
})

func resetMetricsForTest() {
	once = sync.Once{}
	registry = nil
	processedCounter = nil
	failureCounter = nil
	parseLatency = nil
	messagesTotal = nil
	handleLatency = nil
	retriesTotal = nil
	jsAPICallsTotal = nil
	dlqMessagesTotal = nil
	dlqPublishFailures = nil
	publishFailuresTotal = nil
	dbQueriesTotal = nil
	dbQueryLatency = nil
	natsConsumerPending = nil
	aftnValidationErrorsTotal = nil
	messageGapSeconds = nil
	messageSequenceGapTotal = nil
	serialReaderHealthy = nil
}
