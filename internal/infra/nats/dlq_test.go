package nats

import (
	"context"
	"errors"

	configpkg "caatsm/internal/infra/config"
	"caatsm/internal/infra/telemetry"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

var _ = Describe("DLQ", func() {
	var (
		logger *zap.Logger
	)

	BeforeEach(func() {
		logger = zaptest.NewLogger(GinkgoT())
	})

	Describe("validateDLQ", func() {
		It("returns nil when consumer is nil", func() {
			var c *Consumer
			Expect(c.validateDLQ()).To(Succeed())
		})

		It("returns nil when mode is not jetstream", func() {
			c := &Consumer{
				mode: "core",
				cfg: &configpkg.Config{
					DLQ: configpkg.DLQConfig{
						Enabled: true,
						Subject: "caatsm.dlq",
					},
				},
				logger: logger,
			}
			Expect(c.validateDLQ()).To(Succeed())
		})

		It("clears dlqSubject when DLQ is disabled", func() {
			c := &Consumer{
				mode:      "jetstream",
				dlqSubject: "caatsm.dlq",
				cfg: &configpkg.Config{
					DLQ: configpkg.DLQConfig{
						Enabled: false,
						Subject: "caatsm.dlq",
					},
				},
				logger: logger,
			}
			Expect(c.validateDLQ()).To(Succeed())
			Expect(c.dlqSubject).To(Equal(""))
		})

		It("returns error when DLQ is enabled but subject is empty", func() {
			c := &Consumer{
				mode:      "jetstream",
				dlqSubject: "",
				cfg: &configpkg.Config{
					DLQ: configpkg.DLQConfig{
						Enabled: true,
						Subject: "",
					},
				},
				logger: logger,
			}
			err := c.validateDLQ()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("DLQ enabled but dlq.subject is empty"))
		})

		It("returns error when DLQ is enabled but JetStream context is nil", func() {
			c := &Consumer{
				mode:      "jetstream",
				dlqSubject: "caatsm.dlq",
				js:        nil,
				cfg: &configpkg.Config{
					DLQ: configpkg.DLQConfig{
						Enabled: true,
						Subject: "caatsm.dlq",
					},
				},
				logger:    logger,
				telemetry: telemetry.NewNoop(),
			}
			err := c.validateDLQ()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("JetStream context is nil"))
		})
	})

	Describe("routeToDLQ", func() {
		It("returns nil when consumer is nil", func() {
			var c *Consumer
			ctx := context.Background()
			msg := &nats.Msg{}
			err := errors.New("test error")
			Expect(c.routeToDLQ(ctx, msg, err)).To(Succeed())
		})

		It("returns nil when mode is not jetstream", func() {
			c := &Consumer{
				mode: "core",
			}
			ctx := context.Background()
			msg := &nats.Msg{}
			err := errors.New("test error")
			Expect(c.routeToDLQ(ctx, msg, err)).To(Succeed())
		})

		It("returns nil when dlqSubject is empty", func() {
			c := &Consumer{
				mode:      "jetstream",
				dlqSubject: "",
				js:        nil, // Can be nil for this test
			}
			ctx := context.Background()
			msg := &nats.Msg{}
			err := errors.New("test error")
			Expect(c.routeToDLQ(ctx, msg, err)).To(Succeed())
		})
	})
})

