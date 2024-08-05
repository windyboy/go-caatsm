package nats

import (
	"caatsm/internal/config"
	"caatsm/internal/handlers"
	"context"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	nc "github.com/nats-io/nats.go"
)

func Subscribe(config *config.Config, marshaler *handlers.PlainTextMarshaler, handler *handlers.MessageHandler) {
	logger := watermill.NewStdLogger(false, false)
	options := []nc.Option{
		nc.RetryOnFailedConnect(true),
		nc.Timeout(config.Timeouts.Server),
		nc.ReconnectWait(config.Timeouts.ReconnectWait),
	}
	jsConfig := nats.JetStreamConfig{Disabled: true}

	subscriber, err := nats.NewSubscriber(
		nats.SubscriberConfig{
			URL:            config.Nats.URL,
			CloseTimeout:   config.Timeouts.Close,
			AckWaitTimeout: config.Timeouts.AckWait,
			NatsOptions:    options,
			Unmarshaler:    marshaler,
			JetStream:      jsConfig,
		},
		logger,
	)
	if err != nil {
		panic(err)
	}

	logger.Info("NATS server connected", map[string]interface{}{"url": config.Nats.URL})
	logger.Info("Subscribing to NATS topic", map[string]interface{}{"topic": config.Subscription.Topic})

	defer subscriber.Close()
	messages, err := subscriber.Subscribe(context.Background(), config.Subscription.Topic)
	if err != nil {
		logger.Error("Failed to subscribe to NATS topic", err, map[string]interface{}{"topic": config.Subscription.Topic})
		return
	}

	for msg := range messages {
		handler.HandleMessage(msg)
		msg.Ack()
	}
}
