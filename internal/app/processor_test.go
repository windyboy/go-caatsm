package app

import (
	"caatsm/internal/adapter"
	"caatsm/internal/adapter/parser"
	"caatsm/internal/domain"
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"
)

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

type stubPublisher struct {
	last interface{}
	err  error
}

func (s *stubPublisher) Publish(message interface{}) error {
	s.last = message
	return s.err
}

func newTestProcessor(p parser.Parser, repo adapter.Repository, pub adapter.Publisher) *MessageProcessor {
	return NewMessageProcessor(p, repo, pub, zap.NewNop())
}

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

func TestHandleSuccessSetsUuidAndPublishes(t *testing.T) {
	parsed := domain.NewParsedMessage()
	parsed.Parsed = true

	repo := &stubRepository{}
	pub := &stubPublisher{}
	proc := newTestProcessor(&stubParser{value: parsed}, repo, pub)

	const msgID = "uuid-123"
	err := proc.Handle(context.Background(), []byte("payload"), msgID)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	if len(repo.inserted) != 1 {
		t.Fatalf("expected one inserted message, got %d", len(repo.inserted))
	}
	if repo.inserted[0].Uuid != msgID {
		t.Fatalf("expected message uuid to be %s, got %s", msgID, repo.inserted[0].Uuid)
	}
	if pub.last == nil {
		t.Fatalf("expected publisher to receive message")
	}
	if pub.last != repo.inserted[0] {
		t.Fatalf("publisher received unexpected message pointer")
	}
}

func TestHandlePublisherErrorIsPermanent(t *testing.T) {
	parsed := domain.NewParsedMessage()
	parsed.Parsed = true

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
	if len(repo.inserted) != 1 {
		t.Fatalf("expected message to insert before publish failure")
	}
}
