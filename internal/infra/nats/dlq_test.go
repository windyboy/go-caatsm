package nats

import (
	"caatsm/internal/infra/telemetry"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

var _ = Describe("DLQHandler", func() {
	var (
		handler *defaultDLQHandler
		logger  *zap.Logger
	)

	BeforeEach(func() {
		logger = zaptest.NewLogger(GinkgoT())
		handler = &defaultDLQHandler{
			logger:       logger,
			streamName:   "TEST_STREAM",
			consumerName: "test-consumer",
			dlqSubject:   "caatsm.dlq",
			telemetry:    telemetry.NewNoop(),
		}
	})

	Describe("ValidateDLQ", func() {
		It("returns error when JetStream context is nil", func() {
			handler.js = nil
			err := handler.ValidateDLQ()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("JetStream context is nil"))
		})
	})

})
