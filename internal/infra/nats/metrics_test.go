package nats

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap/zaptest"
)

var _ = Describe("Metrics", func() {
	var (
		c   *Consumer
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		c = &Consumer{
			streamName:   "TEST_STREAM",
			consumerName: "test-consumer",
			logger:       zaptest.NewLogger(GinkgoT()),
		}
	})

	Describe("initMetrics", func() {
		It("initializes metrics without error", func() {
			c.initMetrics()
			Expect(c.meter).NotTo(BeNil())
		})
	})

	Describe("recordConsumerMetrics", func() {
		It("handles nil ConsumerInfo gracefully", func() {
			c.initMetrics()
			c.recordConsumerMetrics(ctx, nil)
			// Should not panic
		})

		It("records metrics when ConsumerInfo is provided", func() {
			c.initMetrics()
			info := &nats.ConsumerInfo{
				Config: nats.ConsumerConfig{},
				Delivered: nats.SequenceInfo{
					Consumer: 50,
					Stream:   100,
				},
			}
			// Set fields directly (they are exported)
			// Note: ConsumerInfo fields may not all be exported, so we test what we can
			c.recordConsumerMetrics(ctx, info)
			// Should not panic
		})
	})
})

