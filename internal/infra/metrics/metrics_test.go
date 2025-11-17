package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMetrics(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metrics Suite")
}

var _ = Describe("Metrics", func() {
	Describe("Handler", func() {
		It("serves metrics with the correct content type", func() {
			req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			resp := httptest.NewRecorder()

			Handler().ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Header().Get("Content-Type")).To(ContainSubstring("text/plain"))
		})
	})

	Describe("labelValue helper", func() {
		It("returns unknown for empty values and lowercases input", func() {
			Expect(labelValue("   ")).To(Equal("unknown"))
			Expect(labelValue("SOME_VALUE")).To(Equal("some_value"))
		})
	})

	Describe("record helpers", func() {
		It("can be invoked without panicking", func() {
			Expect(func() {
				RecordProcessed("parsed", "ARR", 150*time.Millisecond)
				RecordFailure("parser")
				RecordMessageHandled("TEST", "consumer", ResultOK, 205*time.Millisecond)
				RecordRetry("TEST", "consumer", RetryReasonProcessorError)
				RecordDLQMessage("TEST", "consumer")
				RecordDLQPublishFailure("TEST", "consumer")
				RecordDBQuery("insert", DBResultOK, 10*time.Millisecond)
				RecordJSAPICall("publish")
				RecordNATSConsumerPending("TEST", "consumer", 7)
			}).NotTo(Panic())
		})
	})
})
