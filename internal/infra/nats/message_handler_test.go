package nats

import (
	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zaptest"
)

var _ = Describe("MessageHandler", func() {
	var (
		processor *defaultBatchProcessor
	)

	BeforeEach(func() {
		processor = &defaultBatchProcessor{
			logger:       zaptest.NewLogger(GinkgoT()),
			streamName:   "TEST_STREAM",
			consumerName: "test-consumer",
			mode:         "jetstream",
		}
	})

	Describe("resolveMsgID", func() {
		It("extracts message ID from header", func() {
			msg := &nats.Msg{
				Header: nats.Header{},
			}
			msg.Header.Set("Nats-Msg-Id", "msg-123")

			id, source, err := processor.resolveMsgID(msg)
			Expect(err).NotTo(HaveOccurred())
			Expect(id).To(Equal("msg-123"))
			Expect(source).To(Equal("header"))
		})

		It("generates UUID for core mode when header is missing", func() {
			processor.mode = "core"
			msg := &nats.Msg{
				Header: nats.Header{},
			}

			id, source, err := processor.resolveMsgID(msg)
			Expect(err).NotTo(HaveOccurred())
			Expect(id).NotTo(BeEmpty())
			Expect(source).To(Equal("generated"))
		})

		It("returns error for JetStream mode when header and metadata are missing", func() {
			processor.mode = "jetstream"
			msg := &nats.Msg{
				Header: nats.Header{},
			}

			// Without metadata, this should return an error
			_, _, err := processor.resolveMsgID(msg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fetch metadata"))
		})
	})
})
