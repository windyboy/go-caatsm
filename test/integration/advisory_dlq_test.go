//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"caatsm/internal/infra/config"
	loginfra "caatsm/internal/infra/log"
	natsinfra "caatsm/internal/infra/nats"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdvisoryDLQHandler(t *testing.T) {
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

	streamName := "TEST_ADVISORY_STREAM"
	consumerName := "test-advisory-consumer"
	dlqSubject := "test.dlq"

	// Create stream
	streamConfig := &nats.StreamConfig{
		Name:     streamName,
		Subjects: []string{"test.orders.*"},
		Storage:  nats.FileStorage,
	}
	_, err = js.AddStream(streamConfig)
	require.NoError(t, err)
	defer func() {
		_ = js.DeleteStream(streamName)
	}()

	// Create DLQ stream
	dlqStreamConfig := &nats.StreamConfig{
		Name:     "TEST_DLQ_STREAM",
		Subjects: []string{dlqSubject},
		Storage:  nats.FileStorage,
	}
	_, err = js.AddStream(dlqStreamConfig)
	require.NoError(t, err)
	defer func() {
		_ = js.DeleteStream("TEST_DLQ_STREAM")
	}()

	// Create consumer with MaxDeliver = 2 for testing
	consumerConfig := &nats.ConsumerConfig{
		Durable:    consumerName,
		AckPolicy:  nats.AckExplicitPolicy,
		MaxDeliver: 2, // Low value for testing
		AckWait:    5 * time.Second,
	}
	_, err = js.AddConsumer(streamName, consumerConfig)
	require.NoError(t, err)

	// Create a mock telemetry recorder
	telemetry := &mockTelemetryRecorder{}

	// Create a simple logger for the test
	logger, err := loginfra.ProvideLogger(&config.Config{
		Log: config.LogConfig{
			Level:  "info",
			Format: "console",
		},
	})
	require.NoError(t, err)
	defer logger.Sync()

	// Create advisory DLQ handler
	handler, err := natsinfra.NewAdvisoryDLQHandler(
		js,
		nc,
		streamName,
		consumerName,
		dlqSubject,
		logger,
		telemetry,
	)
	require.NoError(t, err)

	// Start handler in background
	handlerCtx, handlerCancel := context.WithCancel(ctx)
	defer handlerCancel()
	go func() {
		_ = handler.Start(handlerCtx)
	}()

	// Give handler time to subscribe
	time.Sleep(100 * time.Millisecond)

	// Publish a message that will fail processing
	testSubject := "test.orders.1"
	testData := []byte("test message data")
	_, err = js.Publish(testSubject, testData)
	require.NoError(t, err)

	// Create pull subscription and fetch message
	sub, err := js.PullSubscribe(testSubject, consumerName, nats.Bind(streamName, consumerName))
	require.NoError(t, err)
	defer sub.Unsubscribe()

	// Fetch and NAK the message multiple times to exhaust MaxDeliver
	msgs, err := sub.Fetch(1, nats.MaxWait(2*time.Second))
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	msg := msgs[0]
	// NAK first time
	err = msg.Nak()
	require.NoError(t, err)

	// Wait for redelivery and NAK again to exhaust MaxDeliver
	time.Sleep(6 * time.Second) // Wait for ack_wait + some buffer

	msgs, err = sub.Fetch(1, nats.MaxWait(2*time.Second))
	if err == nil && len(msgs) > 0 {
		// NAK second time to exhaust MaxDeliver
		err = msgs[0].Nak()
		require.NoError(t, err)
	}

	// Wait for advisory message to be processed
	time.Sleep(2 * time.Second)

	// Verify message was published to DLQ
	dlqSub, err := js.SubscribeSync(dlqSubject)
	require.NoError(t, err)
	defer dlqSub.Unsubscribe()

	dlqMsg, err := dlqSub.NextMsg(5 * time.Second)
	if assert.NoError(t, err, "Expected message in DLQ") {
		var payload map[string]interface{}
		err = json.Unmarshal(dlqMsg.Data, &payload)
		require.NoError(t, err)

		// Verify payload structure
		assert.Equal(t, streamName, payload["stream"])
		assert.Equal(t, consumerName, payload["consumer"])
		assert.True(t, payload["advisory_source"].(bool))
		assert.Contains(t, payload["error"].(string), "exhausted max_deliver")
		assert.Equal(t, string(testData), payload["body"])
	}

	// Verify telemetry was called
	assert.Greater(t, telemetry.getDLQMessages(), 0, "Expected DLQ message to be recorded")
}

// mockTelemetryRecorder implements TelemetryRecorder for testing
type mockTelemetryRecorder struct {
	mu          sync.RWMutex
	dlqMessages int
	dlqFailures int
}

func (m *mockTelemetryRecorder) RecordDLQMessage(ctx context.Context, stream, consumer string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dlqMessages++
}

func (m *mockTelemetryRecorder) RecordDLQPublishFailure(ctx context.Context, stream, consumer string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dlqFailures++
}

func (m *mockTelemetryRecorder) getDLQMessages() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.dlqMessages
}
