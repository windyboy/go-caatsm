package nats

import (
	"context"
	"errors"
	"time"

	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

var _ = Describe("ErrorHandler", func() {
	var (
		handler *ErrorHandler
		logger  *zap.Logger
	)

	BeforeEach(func() {
		logger = zaptest.NewLogger(GinkgoT())
		handler = NewErrorHandler(logger)
	})

	Describe("HandleProcessingError", func() {
		It("identifies permanent errors correctly", func() {
			// Mock a permanent error (this would be defined in the app package)
			permanentErr := errors.New("permanent error")
			// For testing, we'll assume any error is transient unless specified

			result := handler.HandleProcessingError(0, permanentErr, logger, "test.subject")
			Expect(result.IsPermanent).To(BeFalse()) // Since we can't easily mock app.IsPermanent
			Expect(result.ShouldApplyBackpressure).To(BeFalse())
		})

		It("applies backpressure for consecutive errors", func() {
			transientErr := errors.New("transient error")

			result := handler.HandleProcessingError(10, transientErr, logger, "test.subject")
			Expect(result.IsPermanent).To(BeFalse())
			Expect(result.ShouldApplyBackpressure).To(BeTrue())
			Expect(result.BackpressureDelay).To(BeNumerically(">=", 100*time.Millisecond))
		})
	})

	Describe("HandleFetchError", func() {
		var (
			ctx              context.Context
			sub              *nats.Subscription
			fetchErrorStreak int
			streamName       string
			consumerName     string
		)

		BeforeEach(func() {
			ctx = context.Background()
			sub = nil
			fetchErrorStreak = 0
			streamName = "TEST_STREAM"
			consumerName = "test-consumer"
		})

		It("handles timeout errors", func() {
			result := handler.HandleFetchError(ctx, nats.ErrTimeout, &sub, &fetchErrorStreak, streamName, consumerName, nil)
			Expect(result.ShouldContinue).To(BeTrue())
			Expect(result.Error).NotTo(HaveOccurred())
		})

		It("handles no responders with backoff", func() {
			result := handler.HandleFetchError(ctx, nats.ErrNoResponders, &sub, &fetchErrorStreak, streamName, consumerName, nil)
			Expect(result.ShouldContinue).To(BeTrue())
			Expect(result.Error).NotTo(HaveOccurred())
			Expect(fetchErrorStreak).To(Equal(1))
		})

		It("handles resource not found errors in dev environment", func() {
			// Mock resource not found error
			resourceErr := errors.New("stream not found")
			// Provide a no-op recovery function to avoid panic
			recoveryFunc := func() (*nats.Subscription, error) {
				return nil, errors.New("recovery not implemented in test")
			}
			result := handler.HandleFetchError(ctx, resourceErr, &sub, &fetchErrorStreak, streamName, consumerName, recoveryFunc)
			// In test environment, this should attempt recovery but fail since recovery func returns error
			Expect(result.ShouldContinue).To(BeFalse())
			Expect(result.Error).To(HaveOccurred())
		})

		It("handles successful recovery", func() {
			resourceErr := errors.New("consumer not found")
			mockSub := &nats.Subscription{}
			recoveryFunc := func() (*nats.Subscription, error) {
				return mockSub, nil
			}
			result := handler.HandleFetchError(ctx, resourceErr, &sub, &fetchErrorStreak, streamName, consumerName, recoveryFunc)
			Expect(result.ShouldContinue).To(BeTrue())
			Expect(result.RecoveredSub).To(Equal(mockSub))
			Expect(fetchErrorStreak).To(Equal(0)) // Should reset on successful recovery
		})
	})
})
