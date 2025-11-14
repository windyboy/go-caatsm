package nats

import (
	"fmt"
	"time"

	"caatsm/internal/config"
	"caatsm/pkg/utils"

	"github.com/nats-io/nats.go"
)

// NewJetStream creates a new JetStream context
func NewJetStream(cfg *config.NatsConfig) (nats.JetStreamContext, error) {
	nc, err := nats.Connect(cfg.URL, nats.RetryOnFailedConnect(true))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to get JetStream context: %w", err)
	}

	return js, nil
}

// EnsureStream ensures a JetStream stream exists with the given configuration
func EnsureStream(js nats.JetStreamContext, streamConfig *nats.StreamConfig) error {
	log := utils.GetSugaredLogger()

	stream, err := js.StreamInfo(streamConfig.Name)
	if err != nil && err != nats.ErrStreamNotFound {
		return fmt.Errorf("failed to check stream info: %w", err)
	}

	if stream != nil {
		log.Infof("Stream %s already exists", streamConfig.Name)
		return nil
	}

	_, err = js.AddStream(streamConfig)
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}

	log.Infof("Created JetStream stream: %s", streamConfig.Name)
	return nil
}

// CreateStreamConfig creates a StreamConfig from application config
func CreateStreamConfig(cfg *config.NatsConfig) *nats.StreamConfig {
	jsCfg := cfg.JetStream

	streamConfig := &nats.StreamConfig{
		Name:      jsCfg.StreamName,
		Subjects:  []string{jsCfg.Subject},
		Retention: nats.LimitsPolicy,
		MaxAge:    7 * 24 * time.Hour, // 7 days retention
		Storage:   nats.FileStorage,
		Replicas:  1,
	}

	return streamConfig
}
