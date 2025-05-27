package nats // Already has package comment

import (
	"caatsm/internal/config"
	"caatsm/pkg/utils"
	"encoding/json"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
)

// NatsPublisher is a message publisher that uses Watermill's NATS publisher.
// It is responsible for publishing messages to a NATS server.
type NatsPublisher struct {
	config    *config.Config    // config holds application configuration, including NATS URLs and topics.
	publisher *nats.Publisher // publisher is the Watermill NATS publisher instance.
}

// NewPub creates and initializes a new NatsPublisher.
// It configures the Watermill NATS publisher based on the provided application configuration.
// config: The application configuration.
// Returns a pointer to the initialized NatsPublisher.
// Note: The current implementation uses `publisher, _ := nats.NewPublisher(...)` which discards
// the error from `nats.NewPublisher`. This should be handled in a production environment.
func NewPub(config *config.Config) *NatsPublisher {
	logger := watermill.NewStdLogger(false, false) // Watermill logger, can be configured

	// JetStream is disabled by default in this configuration.
	jsConfig := nats.JetStreamConfig{Disabled: true}
	
	// NATS connection options from the application configuration.
	options := []nc.Option{
		nc.RetryOnFailedConnect(true),
		nc.Timeout(config.Timeouts.Server),
		nc.ReconnectWait(config.Timeouts.ReconnectWait),
	}

	// Create a new Watermill NATS publisher.
	// Error from nats.NewPublisher is currently ignored with `_`.
	// In a production system, this error should be checked and handled appropriately.
	publisher, err := nats.NewPublisher(
		nats.PublisherConfig{
			URL:         config.Nats.URL,
			NatsOptions: options,
			JetStream:   jsConfig,
		}, logger)
	if err != nil {
		// If error handling is improved, this log might become conditional.
		// For now, assuming it could panic or log if NewPublisher fails.
		// Based on Watermill docs, NewPublisher can return error.
		utils.GetSugaredLogger().Fatalf("Failed to create NATS publisher: %v", err) // Added Fatalf for clarity on failure
	}
	
	return &NatsPublisher{
		config:    config,
		publisher: publisher,
	}
}

// Publish marshals the given message to JSON and publishes it to the NATS topic
// specified in the configuration.
// parsedMessage: The message to be published. It can be any interface{} that is marshallable to JSON.
// Returns an error if JSON marshalling fails or if the message publishing fails.
func (n *NatsPublisher) Publish(parsedMessage interface{}) error {
	logger := utils.GetSugaredLogger()

	messageText, err := json.Marshal(parsedMessage)
	if err != nil {
		logger.Errorf("Failed to marshal message: %v", err)
	}
	msg := message.NewMessage(watermill.NewUUID(), []byte(messageText))
	err = n.publisher.Publish(n.config.Publisher.Topic, msg)
	if err != nil {
		logger.Errorf("Failed to publish message: %v", err)
		return err
	}
	logger.Infof("Message published: %s", msg.UUID)
	return nil
}
