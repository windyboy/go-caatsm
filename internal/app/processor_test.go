package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"caatsm/internal/adapter"
	"caatsm/internal/adapter/parser"
	"caatsm/internal/domain"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestHandleEmptyMessageIsPermanent(t *testing.T) {
	proc := newTestProcessor(&stubParser{}, &stubRepository{}, &stubPublisher{})
	err := proc.Handle(context.Background(), nil, "id-1")
	if err == nil || !IsPermanent(err) {
		t.Fatalf("expected permanent error for empty message, got %v", err)
	}
}

func TestHandleNilParserResultIsPermanent(t *testing.T) {
	proc := newTestProcessor(&stubParser{value: nil}, &stubRepository{}, &stubPublisher{})
	err := proc.Handle(context.Background(), []byte("payload"), "id-2")
	if err == nil || !IsPermanent(err) {
		t.Fatalf("expected permanent error for nil parser result, got %v", err)
	}
}

func TestHandleSuccessDoesNotOverwriteUuid(t *testing.T) {
	originalUUID := uuid.NewString()
	parsed := &domain.ParsedMessage{Uuid: originalUUID, Parsed: true}

	repo := &stubRepository{}
	pub := &stubPublisher{}
	proc := newTestProcessor(&stubParser{value: parsed}, repo, pub)

	const msgID = "msg-123"
	if err := proc.Handle(context.Background(), []byte("payload"), msgID); err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	if repo.last() == nil {
		t.Fatalf("expected message to be inserted")
	}
	if repo.last().Uuid != originalUUID {
		t.Fatalf("expected uuid to remain %s, got %s", originalUUID, repo.last().Uuid)
	}
	if !strings.Contains(repo.last().Comments, "nats_msg_id=msg-123") {
		t.Fatalf("expected comments to contain msg id, got %q", repo.last().Comments)
	}
	if pub.last == nil {
		t.Fatalf("expected publisher to receive message")
	}
}

func TestHandlePublisherErrorIsPermanent(t *testing.T) {
	parsed := &domain.ParsedMessage{Parsed: true}

	repo := &stubRepository{}
	pub := &stubPublisher{err: errors.New("publish failed")}
	proc := newTestProcessor(&stubParser{value: parsed}, repo, pub)

	err := proc.Handle(context.Background(), []byte("payload"), "id-3")
	if err == nil {
		t.Fatalf("expected error when publisher fails")
	}
	if !IsPermanent(err) {
		t.Fatalf("publisher failure should be permanent")
	}
	if repo.last() == nil {
		t.Fatalf("expected message to insert before publish failure")
	}
}

func TestMessageProcessor_Handle_SetsReceivedAndParsedAtWhenZero(t *testing.T) {
	parsed := &domain.ParsedMessage{
		Uuid:   uuid.NewString(),
		Parsed: true,
	}
	repo := &stubRepository{}
	publisher := &stubPublisher{}
	logger := zap.NewNop()
	processor := NewMessageProcessor(&stubParser{value: parsed}, repo, publisher, logger)

	start := time.Now()
	if err := processor.Handle(context.Background(), []byte("raw"), "msg-4"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	saved := repo.last()
	if saved == nil {
		t.Fatal("expected repository to receive a message")
	}
	if saved.ReceivedAt.IsZero() || saved.ParsedAt.IsZero() {
		t.Fatalf("expected timestamps to be set, got received=%v parsed=%v", saved.ReceivedAt, saved.ParsedAt)
	}
	if saved.ReceivedAt.Before(start.Add(-time.Second)) || saved.ParsedAt.Before(start.Add(-time.Second)) {
		t.Fatalf("timestamps look stale: received=%v parsed=%v", saved.ReceivedAt, saved.ParsedAt)
	}
}

func TestMessageProcessor_Handle_DoesNotOverrideExistingTimestamps(t *testing.T) {
	received := time.Now().Add(-2 * time.Minute)
	parsedAt := time.Now().Add(-time.Minute)

	parsed := &domain.ParsedMessage{
		Uuid:       uuid.NewString(),
		Parsed:     true,
		ReceivedAt: received,
		ParsedAt:   parsedAt,
	}
	repo := &stubRepository{}
	publisher := &stubPublisher{}
	logger := zap.NewNop()
	processor := NewMessageProcessor(&stubParser{value: parsed}, repo, publisher, logger)

	if err := processor.Handle(context.Background(), []byte("raw"), "msg-5"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	saved := repo.last()
	if saved.ReceivedAt != received {
		t.Fatalf("expected received_at to remain %v, got %v", received, saved.ReceivedAt)
	}
	if saved.ParsedAt != parsedAt {
		t.Fatalf("expected parsed_at to remain %v, got %v", parsedAt, saved.ParsedAt)
	}
}

func TestContentPreview_TruncatesLongContent(t *testing.T) {
	longContent := strings.Repeat("a", 1024)
	preview := truncateContent(longContent, 256)
	if len(preview) != 256 {
		t.Fatalf("expected preview length 256, got %d", len(preview))
	}
	if !strings.HasSuffix(preview, "...") {
		t.Fatalf("expected preview to end with ellipsis, got %q", preview[len(preview)-10:])
	}
}

func TestMessageProcessor_Handle_NotParsedLogsPreviewOnly(t *testing.T) {
	core, logs := observer.New(zap.WarnLevel)
	logger := zap.New(core)

	parser := &stubParser{
		value: &domain.ParsedMessage{
			Content: strings.Repeat("x", 1024),
			Parsed:  false,
		},
	}
	repo := &stubRepository{}
	publisher := &stubPublisher{}
	processor := NewMessageProcessor(parser, repo, publisher, logger)

	if err := processor.Handle(context.Background(), []byte("raw"), "msg-6"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := logs.FilterMessage("Message not parsed").All()
	if len(entries) == 0 {
		t.Fatal("expected a warning log for unparsed message")
	}
	preview, ok := entries[0].ContextMap()["content_preview"].(string)
	if !ok {
		t.Fatal("expected content_preview field in log")
	}
	if len(preview) > 256 {
		t.Fatalf("expected preview <= 256 chars, got %d", len(preview))
	}
}

func newTestProcessor(p parser.Parser, repo adapter.Repository, pub adapter.Publisher) *MessageProcessor {
	return NewMessageProcessor(p, repo, pub, zap.NewNop())
}

type stubParser struct {
	value *domain.ParsedMessage
}

func (s *stubParser) Parse(rawText string) *domain.ParsedMessage {
	return s.value
}

type stubRepository struct {
	inserted []*domain.ParsedMessage
	err      error
}

func (s *stubRepository) InsertOne(ctx context.Context, msg *domain.ParsedMessage) error {
	if s.err != nil {
		return s.err
	}
	s.inserted = append(s.inserted, msg)
	return nil
}

func (s *stubRepository) InsertBatch(ctx context.Context, msgs []*domain.ParsedMessage) error {
	return errors.New("not implemented")
}

func (s *stubRepository) last() *domain.ParsedMessage {
	if len(s.inserted) == 0 {
		return nil
	}
	return s.inserted[len(s.inserted)-1]
}

type stubPublisher struct {
	last interface{}
	err  error
}

func (s *stubPublisher) Publish(message interface{}) error {
	s.last = message
	return s.err
}
