package pipeline

import (
	"caatsm/internal/telegram"
	"context"
	"errors"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

var _ = Describe("Processor", func() {
	var (
		store      *stubStore
		pub        *stubPublisher
		rec        *stubRecorder
		parserStub *stubParser
		proc       *Processor
		ctx        context.Context
	)

	BeforeEach(func() {
		store = &stubStore{}
		pub = &stubPublisher{}
		rec = &stubRecorder{}
		parserStub = &stubParser{}
		proc = NewProcessor(parserStub, telegram.NewAFTNValidator(), store, pub, rec, zap.NewNop(), Options{})
		ctx = context.Background()
	})

	Describe("Handle", func() {
		It("returns a permanent error when payload is empty", func() {
			err := proc.Handle(ctx, nil, "id-1")
			Expect(err).To(HaveOccurred())
			Expect(IsPermanent(err)).To(BeTrue())
		})

		It("records raw messages when parser returns nil", func() {
			parserStub.value = nil
			err := proc.Handle(ctx, []byte("payload"), "id-2")
			Expect(err).To(HaveOccurred())
			Expect(IsPermanent(err)).To(BeTrue())
			Expect(store.rawCount()).To(Equal(1))
		})

		It("does not panic when parser returns Parsed=false with nil error", func() {
			parserStub.value = &telegram.Message{Parsed: false, Status: telegram.StatusBodyError}
			parserStub.err = nil
			err := proc.Handle(ctx, []byte("payload"), "id-nil-err")
			Expect(err).To(HaveOccurred())
			Expect(IsPermanent(err)).To(BeTrue())
			Expect(store.rawCount()).To(Equal(1))
			Expect(store.lastRaw().ErrorReason).To(Equal("message marked as unparsed"))
		})

		It("preserves UUIDs, appends nats message id, and passes context to publisher", func() {
			originalUUID := telegram.NewUUID()
			parserStub.value = &telegram.Message{UUID: originalUUID, Parsed: true, Status: telegram.StatusParsed}

			Expect(proc.Handle(ctx, []byte("payload"), "msg-123")).To(Succeed())

			Expect(store.last()).NotTo(BeNil())
			Expect(store.last().UUID).To(Equal(originalUUID))
			Expect(store.last().Comments).To(ContainSubstring("nats_msg_id=msg-123"))
			Expect(pub.last).NotTo(BeNil())
			Expect(pub.lastCtx).NotTo(BeNil())
		})

		It("propagates permanent vs transient publish error semantics and counts publisher failure once", func() {
			parserStub.value = &telegram.Message{Parsed: true, Status: telegram.StatusParsed, Category: "ARR"}
			pub.err = Permanent(errors.New("bad topic"))

			err := proc.Handle(ctx, []byte("payload"), "id-3")
			Expect(err).To(HaveOccurred())
			Expect(IsPermanent(err)).To(BeTrue())
			Expect(store.rawCount()).To(Equal(1))
			Expect(rec.publishFailures).To(Equal(1))
			Expect(rec.failureStages["publisher"]).To(Equal(1))

			pub.err = errors.New("temporary nats blip")
			err2 := proc.Handle(ctx, []byte("payload"), "id-4")
			Expect(err2).To(HaveOccurred())
			Expect(IsPermanent(err2)).To(BeFalse())
			Expect(store.rawCount()).To(Equal(1)) // No duplicate raw write for transient publish failure
		})

		It("validates AFTN messages when ValidateAFTN=true and skips weather messages", func() {
			proc = NewProcessor(parserStub, telegram.NewAFTNValidator(), store, pub, rec, zap.NewNop(), Options{ValidateAFTN: true})

			parserStub.value = &telegram.Message{
				Protocol:       telegram.ProtocolWeather,
				Category:       "METAR",
				DateTime:       "261125120000",
				PrimaryAddress: "",
				Parsed:         true,
				Status:         telegram.StatusParsed,
			}
			Expect(proc.Handle(ctx, []byte("METAR KJFK 251200Z="), "id-wx")).To(Succeed())

			parserStub.value = &telegram.Message{
				Protocol:          telegram.ProtocolAFTN,
				PriorityIndicator: "XX",
				PrimaryAddress:    "ZBBBZPZX",
				DateTime:          "151430",
				Parsed:            true,
				Status:            telegram.StatusParsed,
			}
			err := proc.Handle(ctx, []byte("bad aftn"), "id-aftn")
			Expect(err).To(HaveOccurred())
			Expect(IsPermanent(err)).To(BeTrue())
			Expect(rec.aftnErrorTypes).To(ContainElement("priority_indicator"))
		})

		It("sets missing timestamps and preserves provided timestamps", func() {
			received := time.Now().Add(-2 * time.Minute)
			parsedAt := time.Now().Add(-1 * time.Minute)
			parserStub.value = &telegram.Message{
				UUID:       telegram.NewUUID(),
				Parsed:     true,
				Status:     telegram.StatusParsed,
				ReceivedAt: received,
				ParsedAt:   parsedAt,
			}

			Expect(proc.Handle(ctx, []byte("payload"), "msg-5")).To(Succeed())
			Expect(store.last().ReceivedAt).To(Equal(received))
			Expect(store.last().ParsedAt).To(Equal(parsedAt))
		})

		It("logs truncated previews when parsing fails", func() {
			core, logs := observer.New(zap.WarnLevel)
			logger := zap.New(core)
			parserStub = &stubParser{
				value: &telegram.Message{
					Content:     strings.Repeat("x", 1024),
					Parsed:      false,
					Status:      telegram.StatusBodyError,
					ErrorReason: "parse failure",
				},
				err: errors.New("parse failure"),
			}
			proc = NewProcessor(parserStub, nil, store, pub, rec, logger, Options{})

			err := proc.Handle(ctx, []byte("raw"), "msg-6")
			Expect(err).To(HaveOccurred())
			Expect(IsPermanent(err)).To(BeTrue())

			entries := logs.FilterMessage("Message failed to parse").All()
			Expect(entries).NotTo(BeEmpty())
			preview, ok := entries[0].ContextMap()["content_preview"].(string)
			Expect(ok).To(BeTrue())
			Expect(len(preview)).To(BeNumerically("<=", 256))
		})
	})

	Describe("truncateContent", func() {
		It("keeps length at limit with ellipsis", func() {
			longContent := strings.Repeat("a", 1024)
			Expect(truncateContent(longContent, 256)).To(HaveLen(256))
			Expect(truncateContent(longContent, 256)).To(HaveSuffix("..."))
		})
	})
})

type stubParser struct {
	value *telegram.Message
	err   error
}

func (s *stubParser) Parse(string) (*telegram.Message, error) {
	return s.value, s.err
}

type stubStore struct {
	inserted []*telegram.Message
	raw      []*telegram.Message
	err      error
	rawErr   error
}

func (s *stubStore) InsertOne(_ context.Context, msg *telegram.Message) error {
	if s.err != nil {
		return s.err
	}
	s.inserted = append(s.inserted, msg)
	return nil
}

func (s *stubStore) InsertBatch(context.Context, []*telegram.Message) error {
	return errors.New("not implemented")
}

func (s *stubStore) InsertRaw(_ context.Context, msg *telegram.Message) error {
	if s.rawErr != nil {
		return s.rawErr
	}
	s.raw = append(s.raw, msg)
	return nil
}

func (s *stubStore) last() *telegram.Message {
	if len(s.inserted) == 0 {
		return nil
	}
	return s.inserted[len(s.inserted)-1]
}

func (s *stubStore) lastRaw() *telegram.Message {
	if len(s.raw) == 0 {
		return nil
	}
	return s.raw[len(s.raw)-1]
}

func (s *stubStore) rawCount() int {
	return len(s.raw)
}

type stubPublisher struct {
	last    *telegram.Message
	lastCtx context.Context
	err     error
}

func (s *stubPublisher) Publish(ctx context.Context, msg *telegram.Message) error {
	s.lastCtx = ctx
	s.last = msg
	return s.err
}

type stubRecorder struct {
	publishFailures int
	failureStages   map[string]int
	aftnErrorTypes  []string
}

func (r *stubRecorder) RecordProcessingResult(context.Context, string, string, time.Duration) {}

func (r *stubRecorder) RecordPublishFailure(context.Context, string) {
	r.publishFailures++
}

func (r *stubRecorder) RecordFailure(stage string) {
	if r.failureStages == nil {
		r.failureStages = make(map[string]int)
	}
	r.failureStages[stage]++
}

func (r *stubRecorder) RecordAFTNValidationError(_ context.Context, errorType string) {
	r.aftnErrorTypes = append(r.aftnErrorTypes, errorType)
}
