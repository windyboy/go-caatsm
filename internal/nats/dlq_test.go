package nats

import (
	"caatsm/internal/observability"
	"context"
	"errors"

	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

type stubDLQJS struct {
	nats.JetStreamContext
	pubErr   error
	lastSubj string
	lastData []byte
}

func (s *stubDLQJS) Publish(subj string, data []byte, _ ...nats.PubOpt) (*nats.PubAck, error) {
	s.lastSubj = subj
	s.lastData = data
	if s.pubErr != nil {
		return nil, s.pubErr
	}
	return &nats.PubAck{}, nil
}

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
			recorder:     observability.NewNoop(),
		}
	})

	It("returns error when JetStream context is nil during validation or routing", func() {
		Expect(handler.ValidateDLQ()).To(HaveOccurred())
		Expect(handler.RouteToDLQ(context.Background(), &nats.Msg{}, errors.New("boom"))).To(HaveOccurred())
	})

	It("publishes JSON payload to DLQ subject and propagates errors", func() {
		js := &stubDLQJS{}
		handler.js = js
		msg := &nats.Msg{Subject: "telegram.serial", Data: []byte("BAD")}

		Expect(handler.RouteToDLQ(context.Background(), msg, errors.New("parse fail"))).To(Succeed())
		Expect(js.lastSubj).To(Equal("caatsm.dlq"))
		Expect(string(js.lastData)).To(ContainSubstring("parse fail"))

		js.pubErr = errors.New("js unavailable")
		Expect(handler.RouteToDLQ(context.Background(), msg, errors.New("parse fail"))).To(HaveOccurred())
	})
})
