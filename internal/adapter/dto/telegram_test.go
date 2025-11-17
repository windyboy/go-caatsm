package dto

import "testing"

func TestNewParsedTelegramDefaults(t *testing.T) {
	pt := NewParsedTelegram()

	if pt == nil {
		t.Fatal("expected NewParsedTelegram to return a non-nil pointer")
	}

	if pt.Parsed {
		t.Fatalf("expected Parsed to be false, got %v", pt.Parsed)
	}

	if pt.Status != MessageStatusUnknown {
		t.Fatalf("expected default status %q, got %q", MessageStatusUnknown, pt.Status)
	}
}

func TestMessageStatusStringValues(t *testing.T) {
	cases := map[MessageStatus]string{
		MessageStatusUnknown:     "unknown",
		MessageStatusParsed:      "parsed",
		MessageStatusHeaderError: "header_error",
		MessageStatusBodyError:   "body_error",
	}

	for status, want := range cases {
		if string(status) != want {
			t.Fatalf("expected %q for status %v, got %q", want, status, status)
		}
	}
}
