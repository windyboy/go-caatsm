package main

import (
	"testing"
	"time"
)

// Test buildBody generates syntactically plausible bodies for each category.
func TestBuildBodyFormats(t *testing.T) {
	tests := []struct {
		category string
		prefix   string
		suffix   string
	}{
		{"ARR", "(ARR-", ")"},
		{"DEP", "(DEP-", ")"},
		{"CNL", "(CNL-", ")"},
		{"DLA", "(DLA-", ")"},
		{"FPL", "(FPL-", ")"},
	}

	for _, tt := range tests {
		body, invalid := buildBody(tt.category)
		if !invalid && body == "" {
			t.Fatalf("category %s: expected non-empty body", tt.category)
		}
		if body[0:len(tt.prefix)] != tt.prefix {
			t.Fatalf("category %s: expected prefix %q, got %q", tt.category, tt.prefix, body[0:len(tt.prefix)])
		}
		if body[len(body)-len(tt.suffix):] != tt.suffix {
			t.Fatalf("category %s: expected suffix %q, got %q", tt.category, tt.suffix, body[len(body)-len(tt.suffix):])
		}
	}
}

// Test buildTelegram wires header and body into a full telegram.
func TestBuildTelegramBasicFields(t *testing.T) {
	tg, _ := buildTelegram("ARR")
	if tg.Category != "ARR" {
		t.Fatalf("expected category ARR, got %s", tg.Category)
	}
	if tg.MessageID == "" {
		t.Fatalf("expected non-empty MessageID")
	}
	if tg.UUID == "" {
		t.Fatalf("expected non-empty UUID")
	}
	if tg.Content == "" {
		t.Fatalf("expected non-empty Content")
	}
	if tg.ReceivedAt.IsZero() {
		t.Fatalf("expected ReceivedAt to be set")
	}
	if tg.Metadata != nil {
		t.Fatalf("expected Metadata to be nil from buildTelegram")
	}
}

func TestBuildCategories(t *testing.T) {
	all := []string{"ARR", "DEP", "CNL", "DLA", "FPL"}

	got := buildCategories("mixed", all)
	if len(got) != len(all) {
		t.Fatalf("expected all categories for mixed, got %d", len(got))
	}

	got = buildCategories("", all)
	if len(got) != len(all) {
		t.Fatalf("expected all categories for empty flag, got %d", len(got))
	}

	got = buildCategories("arr", all)
	if len(got) != 1 || got[0] != "ARR" {
		t.Fatalf("expected single category ARR, got %#v", got)
	}
}

func TestBuildStatuses(t *testing.T) {
	all := []string{"parsed", "header_error", "body_error"}

	got := buildStatuses("random", all)
	if len(got) != len(all) {
		t.Fatalf("expected all statuses for random, got %d", len(got))
	}

	got = buildStatuses("", all)
	if len(got) != len(all) {
		t.Fatalf("expected all statuses for empty flag, got %d", len(got))
	}

	got = buildStatuses("parsed", all)
	if len(got) != 1 || got[0] != "parsed" {
		t.Fatalf("expected single status parsed, got %#v", got)
	}
}

func TestChooseStatusNonRandomUsesProvidedList(t *testing.T) {
	statuses := []string{"parsed"}
	for i := 0; i < 10; i++ {
		got := chooseStatus(false, "parsed", statuses)
		if got != "parsed" {
			t.Fatalf("expected parsed, got %s", got)
		}
	}
}

func TestChooseStatusRandomOnlyReturnsKnownStatuses(t *testing.T) {
	statuses := []string{"parsed", "header_error", "body_error"}

	for i := 0; i < 100; i++ {
		got := chooseStatus(false, "random", statuses)
		if !contains(statuses, got) && got != "body_error" && got != "parsed" {
			t.Fatalf("unexpected status from chooseStatus: %s", got)
		}
	}
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

// Test RunSeed burst mode publishes Count messages.
func TestRunSeedBurstPublishesCount(t *testing.T) {
	cfg := SeedConfig{
		Count:  5,
		Mode:   "burst",
		DryRun: false,
	}
	categories := []string{"ARR"}
	statuses := []string{"parsed"}

	var published int
	pub := func(*telegram) error {
		published++
		return nil
	}

	if err := RunSeed(cfg, categories, statuses, pub); err != nil {
		t.Fatalf("RunSeed burst returned error: %v", err)
	}
	if published != cfg.Count {
		t.Fatalf("expected %d published messages, got %d", cfg.Count, published)
	}
}

// Test RunSeed interval mode respects Count and does not sleep in DryRun.
func TestRunSeedIntervalRespectsCount(t *testing.T) {
	origSleep := sleepFunc
	defer func() { sleepFunc = origSleep }()
	sleepFunc = func(time.Duration) {}

	cfg := SeedConfig{
		Count:       3,
		Mode:        "interval",
		IntervalMin: 10 * time.Millisecond,
		IntervalMax: 20 * time.Millisecond,
		DryRun:      true,
	}
	categories := []string{"ARR"}
	statuses := []string{"parsed"}

	var published int
	pub := func(*telegram) error {
		published++
		return nil
	}

	if err := RunSeed(cfg, categories, statuses, pub); err != nil {
		t.Fatalf("RunSeed interval returned error: %v", err)
	}
	if published != cfg.Count {
		t.Fatalf("expected %d published messages, got %d", cfg.Count, published)
	}
}

// Test RunSeed mixed mode still results in exactly Count messages when Count>0.
func TestRunSeedMixedPublishesCount(t *testing.T) {
	origSleep := sleepFunc
	defer func() { sleepFunc = origSleep }()
	sleepFunc = func(time.Duration) {}

	cfg := SeedConfig{
		Count:       6,
		Mode:        "mixed",
		IntervalMin: 1 * time.Millisecond,
		IntervalMax: 2 * time.Millisecond,
		DryRun:      true,
	}
	categories := []string{"ARR"}
	statuses := []string{"parsed"}

	var published int
	pub := func(*telegram) error {
		published++
		return nil
	}

	if err := RunSeed(cfg, categories, statuses, pub); err != nil {
		t.Fatalf("RunSeed mixed returned error: %v", err)
	}
	if published != cfg.Count {
		t.Fatalf("expected %d published messages, got %d", cfg.Count, published)
	}
}


