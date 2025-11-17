package nats

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap/zaptest"
)

var _ = Describe("MessageHandler", func() {
	var (
		c *Consumer
	)

	BeforeEach(func() {
		c = &Consumer{
			mode:         "jetstream",
			streamName:   "TEST_STREAM",
			consumerName: "test-consumer",
			logger:       zaptest.NewLogger(GinkgoT()),
		}
	})

	Describe("resolveMsgID", func() {
		It("extracts message ID from header", func() {
			msg := &nats.Msg{
				Header: nats.Header{},
			}
			msg.Header.Set("Nats-Msg-Id", "msg-123")

			id, source, err := c.resolveMsgID(msg)
			Expect(err).NotTo(HaveOccurred())
			Expect(id).To(Equal("msg-123"))
			Expect(source).To(Equal("header"))
		})

		It("generates UUID for core mode when header is missing", func() {
			c.mode = "core"
			msg := &nats.Msg{
				Header: nats.Header{},
			}

			id, source, err := c.resolveMsgID(msg)
			Expect(err).NotTo(HaveOccurred())
			Expect(id).NotTo(BeEmpty())
			Expect(source).To(Equal("generated"))
		})

		It("returns error for JetStream mode when header and metadata are missing", func() {
			c.mode = "jetstream"
			msg := &nats.Msg{
				Header: nats.Header{},
			}

			// Without metadata, this should return an error
			_, _, err := c.resolveMsgID(msg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fetch metadata"))
		})
	})
})

