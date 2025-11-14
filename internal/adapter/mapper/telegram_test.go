package mapper

import (
	"testing"
	"time"

	"caatsm/internal/domain"

	"github.com/google/uuid"
)

func TestTelegramMapper_ToDBRow_GeneratesUUIDWhenEmpty(t *testing.T) {
	mapper := NewTelegramMapper()
	msg := &domain.ParsedMessage{}

	row, err := mapper.ToDBRow(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	value, ok := row[0].(uuid.UUID)
	if !ok {
		t.Fatalf("expected first column to be uuid.UUID, got %T", row[0])
	}
	if value == uuid.Nil {
		t.Fatalf("expected generated uuid to be non-nil")
	}
}

func TestTelegramMapper_FromDBRow_RoundTrip(t *testing.T) {
	mapper := NewTelegramMapper()
	now := time.Now().UTC()

	original := &domain.ParsedMessage{
		Uuid:               uuid.NewString(),
		MessageID:          "TMQ1324",
		DateTime:           "150631",
		PriorityIndicator:  "FF",
		PrimaryAddress:     "ZBTJZPZX",
		SecondaryAddresses: "150630 ZBACZQZX",
		Originator:         "ORIGIN",
		OriginatorDateTime: "150630",
		Category:           "FPL",
		Content:            "raw telegram",
		BodyData:           map[string]string{"key": "value"},
		ReceivedAt:         now,
		ParsedAt:           now,
		DispatchedAt:       now,
		NeedDispatch:       true,
	}

	row, err := mapper.ToDBRow(original)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	roundTrip, err := mapper.FromDBRow(row)
	if err != nil {
		t.Fatalf("unexpected error reading row: %v", err)
	}

	if roundTrip.Uuid != original.Uuid {
		t.Fatalf("expected uuid %s, got %s", original.Uuid, roundTrip.Uuid)
	}
	if roundTrip.MessageID != original.MessageID {
		t.Fatalf("expected message_id %s, got %s", original.MessageID, roundTrip.MessageID)
	}
	if roundTrip.NeedDispatch != original.NeedDispatch {
		t.Fatalf("expected need_dispatch %v, got %v", original.NeedDispatch, roundTrip.NeedDispatch)
	}
}
