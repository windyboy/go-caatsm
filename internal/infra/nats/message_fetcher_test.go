package nats

import (
	"context"
	"errors"
	"time"

	configpkg "caatsm/internal/infra/config"

	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

var _ = Describe("MessageFetcher", func() {
	var (
		fetcher *defaultMessageFetcher
		logger  *zap.Logger
	)

	BeforeEach(func() {
		logger = zaptest.NewLogger(GinkgoT())
		fetcher = &defaultMessageFetcher{
			logger: logger,
			config: &consumerConfig{
				streamName:   "TEST_STREAM",
				consumerName: "test-consumer",
			},
			cfg: &configpkg.Config{
				NATS: configpkg.NATSConfig{
					ConsumerRules: configpkg.ConsumerRulesConfig{
						Backoff: []time.Duration{5 * time.Second, 30 * time.Second},
					},
				},
			},
		}
	})

	Describe("HandleFetchError", func() {
		var ctx context.Context

		BeforeEach(func() {
			ctx = context.Background()
		})

		It("returns true for timeout errors", func() {
			var sub *nats.Subscription
			fetchErrorStreak := 0
			shouldContinue, err := fetcher.HandleFetchError(ctx, nats.ErrTimeout, &sub, &fetchErrorStreak)
			Expect(shouldContinue).To(BeTrue())
			Expect(err).NotTo(HaveOccurred())
		})

		It("handles ErrNoResponders with backoff", func() {
			var sub *nats.Subscription
			fetchErrorStreak := 0
			shouldContinue, err := fetcher.HandleFetchError(ctx, nats.ErrNoResponders, &sub, &fetchErrorStreak)
			Expect(shouldContinue).To(BeTrue())
			Expect(err).NotTo(HaveOccurred())
			Expect(fetchErrorStreak).To(Equal(1))
		})

		It("handles resource not found errors", func() {
			var sub *nats.Subscription
			fetchErrorStreak := 0
			resourceErr := errors.New("stream not found")
			shouldContinue, err := fetcher.HandleFetchError(ctx, resourceErr, &sub, &fetchErrorStreak)
			// Simplified error handling just applies backoff and continues
			Expect(err).NotTo(HaveOccurred())
			Expect(shouldContinue).To(BeTrue())
			Expect(fetchErrorStreak).To(Equal(1))
		})

		It("handles generic errors with backoff", func() {
			// We need a non-nil subscription to avoid recovery attempt
			dummySub := &nats.Subscription{}
			sub := dummySub

			fetchErrorStreak := 0
			genericErr := errors.New("generic error")
			shouldContinue, err := fetcher.HandleFetchError(ctx, genericErr, &sub, &fetchErrorStreak)
			Expect(shouldContinue).To(BeTrue())
			Expect(err).NotTo(HaveOccurred())
			Expect(fetchErrorStreak).To(Equal(1))
		})
	})
})
