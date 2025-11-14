package app

import (
	"errors"
	"testing"
)

func TestPermanentWrapsError(t *testing.T) {
	base := errors.New("boom")
	perr := Permanent(base)

	if perr == nil {
		t.Fatalf("expected wrapped error, got nil")
	}
	if !IsPermanent(perr) {
		t.Fatalf("expected IsPermanent to be true")
	}
	if !errors.Is(perr, base) {
		t.Fatalf("expected wrapped error to unwrap to base")
	}
	if errors.Is(base, perr) {
		t.Fatalf("expected base not to consider wrapper as same")
	}
}

func TestPermanentNil(t *testing.T) {
	if Permanent(nil) != nil {
		t.Fatalf("Permanent(nil) should return nil")
	}
	if IsPermanent(nil) {
		t.Fatalf("IsPermanent(nil) should be false")
	}
}
