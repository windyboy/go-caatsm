//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	loginfra "caatsm/internal/infra/log"
	natsinfra "caatsm/internal/infra/nats"
	telemetryinfra "caatsm/internal/infra/telemetry"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
)

func TestDLQStreamIsIsolated(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	natsContainer, natsURL := startNATS(ctx, t)
	defer func() { _ = natsContainer.Terminate(context.Background()) }()

	cfg := buildTestConfig(natsURL, "postgres://unused")
	cfg.DLQ.Enabled = true
	cfg.DLQ.Subject = "integration.dlq"
	require.NoError(t, cfg.Validate())

	logger, err := loginfra.ProvideLogger(cfg)
	require.NoError(t, err)
	defer logger.Sync()
	nc, err := natsinfra.ProvideNATSConn(cfg, logger)
	require.NoError(t, err)
	defer nc.Close()
	js, err := nc.JetStream()
	require.NoError(t, err)

	_, err = natsinfra.ProvideConsumer(nc, js, nil, cfg, telemetryinfra.NewNoop(), logger)
	require.NoError(t, err)

	business, err := js.StreamInfo(cfg.NATS.Stream)
	require.NoError(t, err)
	dlq, err := js.StreamInfo(cfg.NATS.Stream + "_DLQ")
	require.NoError(t, err)
	if dlq.Config.Retention != nats.LimitsPolicy {
		t.Fatalf("DLQ retention = %v, want LimitsPolicy", dlq.Config.Retention)
	}
	require.NotContains(t, business.Config.Subjects, cfg.DLQ.Subject)
	require.Equal(t, []string{cfg.DLQ.Subject}, dlq.Config.Subjects)
}
