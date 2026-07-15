package nats

import (
	"caatsm/internal/adapter/dto"
	"caatsm/internal/infra/config"
	"testing"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// fakeJetStream implements only PublishMsg for testing
type fakeJetStream struct {
	nats.JetStreamContext
	publishedMsg *nats.Msg
}

func (f *fakeJetStream) PublishMsg(msg *nats.Msg, opts ...nats.PubOpt) (*nats.PubAck, error) {
	f.publishedMsg = msg
	return &nats.PubAck{}, nil
}

func TestPublish_NatsMsgId_FromBusinessKey(t *testing.T) {
	js := &fakeJetStream{}
	cfg := &config.Config{
		Publisher: config.PublisherConfig{
			Topic: "test.topic",
		},
	}
	logger := zap.NewNop()
	publisher := &Publisher{js: js, cfg: cfg, logger: logger}

	msg := &dto.ParsedTelegram{
		Uuid:      uuid.NewString(),
		MessageID: "TMQ1234",
		DateTime:  "150631",
		Category:  "ARR",
	}

	if err := publisher.Publish(msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedMsgID := "TMQ1234|150631|ARR"
	actualMsgID := js.publishedMsg.Header.Get("Nats-Msg-Id")
	if actualMsgID != expectedMsgID {
		t.Errorf("expected Nats-Msg-Id %q, got %q", expectedMsgID, actualMsgID)
	}
}

func TestPublish_NatsMsgId_WithoutCategory(t *testing.T) {
	js := &fakeJetStream{}
	cfg := &config.Config{
		Publisher: config.PublisherConfig{
			Topic: "test.topic",
		},
	}
	logger := zap.NewNop()
	publisher := &Publisher{js: js, cfg: cfg, logger: logger}

	msg := &dto.ParsedTelegram{
		Uuid:      uuid.NewString(),
		MessageID: "TMQ1234",
		DateTime:  "150631",
		Category:  "",
	}

	if err := publisher.Publish(msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedMsgID := "TMQ1234|150631"
	actualMsgID := js.publishedMsg.Header.Get("Nats-Msg-Id")
	if actualMsgID != expectedMsgID {
		t.Errorf("expected Nats-Msg-Id %q, got %q", expectedMsgID, actualMsgID)
	}
}

func TestPublish_NatsMsgId_EmptyBusinessKey_FallsBackToUUID(t *testing.T) {
	js := &fakeJetStream{}
	cfg := &config.Config{
		Publisher: config.PublisherConfig{
			Topic: "test.topic",
		},
	}
	logger := zap.NewNop()
	publisher := &Publisher{js: js, cfg: cfg, logger: logger}

	msg := &dto.ParsedTelegram{
		Uuid:      uuid.NewString(),
		MessageID: "",
		DateTime:  "",
	}

	if err := publisher.Publish(msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	actualMsgID := js.publishedMsg.Header.Get("Nats-Msg-Id")
	if actualMsgID == "" {
		t.Error("expected non-empty Nats-Msg-Id")
	}
	// Should be a UUID format (contains dashes)
	if len(actualMsgID) != 36 {
		t.Errorf("expected UUID format, got %q", actualMsgID)
	}
}
