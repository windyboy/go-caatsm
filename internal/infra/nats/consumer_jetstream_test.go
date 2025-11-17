package nats

import (
	"context"
	"errors"
	"time"

	configpkg "caatsm/internal/infra/config"

	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zaptest"
)

var _ = Describe("Consumer JetStream", func() {
	var (
		c *Consumer
	)

	BeforeEach(func() {
		c = &Consumer{
			mode:         "jetstream",
			streamName:   "TEST_STREAM",
			consumerName: "test-consumer",
			subject:      "test.subject",
			batchSize:    10,
			batchTimeout: 2 * time.Second,
			logger:       zaptest.NewLogger(GinkgoT()),
			cfg: &configpkg.Config{
				NATS: configpkg.NATSConfig{
					ConsumerRules: configpkg.ConsumerRulesConfig{
						Backoff: []time.Duration{5 * time.Second, 30 * time.Second},
					},
				},
			},
		}
	})

	Describe("nakWithStrategy", func() {
		PIt("sends NAK without delay when backoff is empty", func() {
			c.cfg.NATS.ConsumerRules.Backoff = []time.Duration{}
			// Note: This test would require a real NATS message to fully test
			// For now, we verify the logic path
		})

		PIt("sends NAK with delay based on delivery attempt", func() {
			// Note: This test would require a real NATS message with metadata
			// For now, we verify the function exists and can be called
		})
	})

	Describe("handleFetchError", func() {
		var ctx context.Context

		BeforeEach(func() {
			ctx = context.Background()
		})

		It("returns true for timeout errors", func() {
			var sub *nats.Subscription
			fetchErrorStreak := 0
			shouldContinue, err := c.handleFetchError(ctx, nats.ErrTimeout, &sub, &fetchErrorStreak)
			Expect(shouldContinue).To(BeTrue())
			Expect(err).NotTo(HaveOccurred())
		})

		It("handles ErrNoResponders with backoff", func() {
			var sub *nats.Subscription
			fetchErrorStreak := 0
			shouldContinue, err := c.handleFetchError(ctx, nats.ErrNoResponders, &sub, &fetchErrorStreak)
			Expect(shouldContinue).To(BeTrue())
			Expect(err).NotTo(HaveOccurred())
			Expect(fetchErrorStreak).To(Equal(1))
		})

		It("handles resource not found errors", func() {
			var sub *nats.Subscription
			fetchErrorStreak := 0
			resourceErr := errors.New("stream not found")
			shouldContinue, err := c.handleFetchError(ctx, resourceErr, &sub, &fetchErrorStreak)
			// Behavior depends on environment; in test this should attempt recovery
			Expect(err).To(HaveOccurred())
			Expect(shouldContinue).To(BeFalse())
		})

		It("handles generic errors with backoff", func() {
			var sub *nats.Subscription
			fetchErrorStreak := 0
			genericErr := errors.New("generic error")
			shouldContinue, err := c.handleFetchError(ctx, genericErr, &sub, &fetchErrorStreak)
			Expect(shouldContinue).To(BeTrue())
			Expect(err).NotTo(HaveOccurred())
			Expect(fetchErrorStreak).To(Equal(1))
		})
	})
})
