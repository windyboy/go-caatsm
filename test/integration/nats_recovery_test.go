//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	natsinfra "caatsm/internal/infra/nats"

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

	// Test stream recovery
	streamManager := natsinfra.NewStreamManager(js, "TEST_STREAM", []string{"test.subject"}, nil)
	err = streamManager.EnsureStream()
	assert.NoError(t, err)

	// Test consumer recovery
	consumerManager := natsinfra.NewConsumerManager(js, "TEST_STREAM", "test-consumer", "test.subject", nil)
	consumerConfig := &nats.ConsumerConfig{
		Durable:   "test-consumer",
		AckPolicy: nats.AckExplicitPolicy,
	}
	err = consumerManager.EnsureConsumer(consumerConfig)
	assert.NoError(t, err)

	// Test pull subscription creation
	sub, err := consumerManager.CreatePullSubscription()
	assert.NoError(t, err)
	sub.Unsubscribe()

	// Test recovery when resources don't exist
	// Delete the consumer and try recovery
	err = js.DeleteConsumer("TEST_STREAM", "test-consumer")
	if err != nil && !errors.Is(err, nats.ErrConsumerNotFound) {
		require.NoError(t, err)
	}

	// This should recreate the consumer
	_, err = consumerManager.CreatePullSubscriptionWithRecovery(streamManager, consumerConfig)
	assert.NoError(t, err)
}
