package nats

import (
	"caatsm/internal/config"
	"encoding/json"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
)

func getLogger() watermill.LoggerAdapter {
	return watermill.NewStdLogger(false, false)
}

func getPublisher(config *config.Config, logger watermill.LoggerAdapter) (*nats.Publisher, error) {
	options := []nc.Option{
		nc.RetryOnFailedConnect(true),
		nc.Timeout(config.Timeouts.Server),
		nc.ReconnectWait(config.Timeouts.ReconnectWait),
	}
	jsConfig := nats.JetStreamConfig{Disabled: true}

	return nats.NewPublisher(
		nats.PublisherConfig{
			URL:         config.Nats.URL,
			NatsOptions: options,
			JetStream:   jsConfig,
		},
		logger,
	)
}

func publishMessage(publisher *nats.Publisher, topic string, parsedMessage interface{}, logger watermill.LoggerAdapter) error {
	messageText, err := json.Marshal(parsedMessage)
	if err != nil {
		logger.Error("Failed to marshal message", err, map[string]interface{}{"message": parsedMessage})
		return err
	}

	msg := message.NewMessage(watermill.NewUUID(), []byte(messageText))
	err = publisher.Publish(topic, msg)
	if err != nil {
		logger.Error("Failed to publish message to NATS topic", err, map[string]interface{}{"topic": topic})
		return err
	}

	return nil
}

func Publish(config *config.Config, parsedMessage interface{}) error {
	logger := getLogger()

	publisher, err := getPublisher(config, logger)
	if err != nil {
		logger.Error("Failed to create publisher", err, nil)
		return err
	}

	// logger.Info("NATS server connected", map[string]interface{}{"url": config.Nats.URL})
	// logger.Info("Publishing message to NATS topic", map[string]interface{}{"topic": config.Publisher.Topic})

	err = publishMessage(publisher, config.Publisher.Topic, parsedMessage, logger)
	if err != nil {
		logger.Error("Failed to publish message", err, map[string]interface{}{"message": parsedMessage})
		return err
	}

	return nil
}
