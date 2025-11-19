package nats

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
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
			config: consumerConfig{
				streamName:      "TEST_STREAM",
				consumerName:    "test-consumer",
				monitorInterval: 30 * time.Second, // Set a valid interval
			},
			logger: zaptest.NewLogger(GinkgoT()),
		}
	})

	Describe("emitConsumerStats", func() {
		It("handles context cancellation", func() {
			ctx, cancel := context.WithCancel(ctx)
			cancel()
			c.emitConsumerStats(ctx)
			// Should return without panic
		})
	})
})
