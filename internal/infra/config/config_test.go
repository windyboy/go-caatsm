package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfig_DefaultAckWait(t *testing.T) {
	t.Setenv("GO_ENV", "testdefaults")

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("failed to chdir to repo root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	want := 30 * time.Second
	if cfg.Timeouts.AckWait != want {
		t.Fatalf("expected timeouts.ack_wait to default to %v, got %v", want, cfg.Timeouts.AckWait)
	}
	if cfg.NATS.ConsumerRules.AckWait != want {
		t.Fatalf("expected consumer ack_wait to default to %v, got %v", want, cfg.NATS.ConsumerRules.AckWait)
	}
}
