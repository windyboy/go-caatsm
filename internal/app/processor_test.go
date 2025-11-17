package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"caatsm/internal/adapter/parser"
	"caatsm/internal/adapter/dto"
	"caatsm/internal/infra/telemetry"
	"caatsm/internal/port"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

var _ = Describe("MessageProcessor", func() {
	var (
		repo       *stubRepository
		pub        *stubPublisher
		proc       *MessageProcessor
		ctx        context.Context
		parserStub *stubParser
	)

	BeforeEach(func() {
		repo = &stubRepository{}
		pub = &stubPublisher{}
		parserStub = &stubParser{}
		proc = newTestProcessor(parserStub, repo, pub)
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
			Expect(repo.rawCount()).To(Equal(1))
		})

		It("preserves UUIDs and appends nats message id comment", func() {
			originalUUID := uuid.NewString()
			parserStub.value = &dto.ParsedTelegram{Uuid: originalUUID, Parsed: true, Status: dto.MessageStatusParsed}

			Expect(proc.Handle(ctx, []byte("payload"), "msg-123")).To(Succeed())

			Expect(repo.last()).NotTo(BeNil())
			Expect(repo.last().Uuid).To(Equal(originalUUID))
			Expect(repo.last().Comments).To(ContainSubstring("nats_msg_id=msg-123"))
			Expect(pub.last).NotTo(BeNil())
		})

		It("treats publisher failures as permanent and stores raw entries", func() {
			parserStub.value = &dto.ParsedTelegram{Parsed: true, Status: dto.MessageStatusParsed}
			pub.err = errors.New("publish failed")

			err := proc.Handle(ctx, []byte("payload"), "id-3")
			Expect(err).To(HaveOccurred())
			Expect(IsPermanent(err)).To(BeTrue())
			Expect(repo.last()).NotTo(BeNil())
			Expect(repo.rawCount()).To(Equal(1))
			Expect(repo.lastRaw().Status).To(Equal(dto.MessageStatusParsed))
			Expect(repo.lastRaw().ErrorReason).To(ContainSubstring("publish failed"))
		})

		It("sets timestamps when missing", func() {
			parserStub.value = &dto.ParsedTelegram{
				Uuid:   uuid.NewString(),
				Parsed: true,
				Status: dto.MessageStatusParsed,
			}
			pub.err = nil

			start := time.Now()
			Expect(proc.Handle(ctx, []byte("payload"), "msg-4")).To(Succeed())

			saved := repo.last()
			Expect(saved).NotTo(BeNil())
			Expect(saved.ReceivedAt).NotTo(BeZero())
			Expect(saved.ParsedAt).NotTo(BeZero())
			Expect(saved.ReceivedAt.After(start.Add(-time.Second))).To(BeTrue())
			Expect(saved.ParsedAt.After(start.Add(-time.Second))).To(BeTrue())
		})

		It("does not override provided timestamps", func() {
			received := time.Now().Add(-2 * time.Minute)
			parsedAt := time.Now().Add(-1 * time.Minute)
			parserStub.value = &dto.ParsedTelegram{
				Uuid:       uuid.NewString(),
				Parsed:     true,
				Status:     dto.MessageStatusParsed,
				ReceivedAt: received,
				ParsedAt:   parsedAt,
			}

			Expect(proc.Handle(ctx, []byte("payload"), "msg-5")).To(Succeed())
			Expect(repo.last().ReceivedAt).To(Equal(received))
			Expect(repo.last().ParsedAt).To(Equal(parsedAt))
		})

		It("logs truncated previews when parsing fails", func() {
			core, logs := observer.New(zap.WarnLevel)
			logger := zap.New(core)
			parserStub = &stubParser{
				value: &dto.ParsedTelegram{
					Content:     strings.Repeat("x", 1024),
					Parsed:      false,
					Status:      dto.MessageStatusBodyError,
					ErrorReason: "parse failure",
				},
				err: errors.New("parse failure"),
			}
			proc = NewMessageProcessor(parserStub, repo, pub, telemetry.NewNoop(), logger)

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

func newTestProcessor(p parser.Parser, repo port.Repository, pub port.Publisher) *MessageProcessor {
	return NewMessageProcessor(p, repo, pub, telemetry.NewNoop(), zap.NewNop())
}

type stubParser struct {
	value *dto.ParsedTelegram
	err   error
}

func (s *stubParser) Parse(rawText string) (*dto.ParsedTelegram, error) {
	return s.value, s.err
}

type stubRepository struct {
	inserted []*dto.ParsedTelegram
	raw      []*dto.ParsedTelegram
	err      error
	rawErr   error
}

func (s *stubRepository) InsertOne(ctx context.Context, msg *dto.ParsedTelegram) error {
	if s.err != nil {
		return s.err
	}
	s.inserted = append(s.inserted, msg)
	return nil
}

func (s *stubRepository) InsertBatch(ctx context.Context, msgs []*dto.ParsedTelegram) error {
	return errors.New("not implemented")
}

func (s *stubRepository) InsertRaw(ctx context.Context, msg *dto.ParsedTelegram) error {
	if s.rawErr != nil {
		return s.rawErr
	}
	s.raw = append(s.raw, msg)
	return nil
}

func (s *stubRepository) last() *dto.ParsedTelegram {
	if len(s.inserted) == 0 {
		return nil
	}
	return s.inserted[len(s.inserted)-1]
}

func (s *stubRepository) lastRaw() *dto.ParsedTelegram {
	if len(s.raw) == 0 {
		return nil
	}
	return s.raw[len(s.raw)-1]
}

func (s *stubRepository) rawCount() int {
	return len(s.raw)
}

type stubPublisher struct {
	last interface{}
	err  error
}

func (s *stubPublisher) Publish(message interface{}) error {
	s.last = message
	return s.err
}
