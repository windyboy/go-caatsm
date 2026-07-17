//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	loginfra "caatsm/internal/infra/log"
	natsinfra "caatsm/internal/infra/nats"
	telemetryinfra "caatsm/internal/infra/telemetry"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNATSConsumerRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// Start NATS container
	natsContainer, natsURL := startNATS(ctx, t)
	defer func() {
		_ = natsContainer.Terminate(context.Background())
	}()

	// Connect to NATS
	nc, err := nats.Connect(natsURL)
	require.NoError(t, err)
	defer nc.Close()

	js, err := nc.JetStream()
	require.NoError(t, err)

	cfg := buildTestConfig(natsURL, "postgres://unused")
	logger, err := loginfra.ProvideLogger(cfg)
	require.NoError(t, err)
	defer logger.Sync()

	_, err = natsinfra.ProvideConsumer(nc, js, nil, cfg, telemetryinfra.NewNoop(), logger)
	require.NoError(t, err)

	err = js.DeleteConsumer(cfg.NATS.Stream, cfg.NATS.Consumer)
	if err != nil && !errors.Is(err, nats.ErrConsumerNotFound) {
		require.NoError(t, err)
	}

	_, err = natsinfra.ProvideConsumer(nc, js, nil, cfg, telemetryinfra.NewNoop(), logger)
	assert.NoError(t, err)
}
