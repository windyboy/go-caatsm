package nats

import (
	"caatsm/internal/config"
	"context"
	"errors"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
)

func Subscribe(config *config.Config) {
	logger := watermill.NewStdLogger(false, false)
	marshaler := &PlainTextMarshaler{}
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

	handlers := New(config)
	for msg := range messages {
		if err := handlers.HandleMessage(msg); err == nil {
			msg.Ack()
		} else {
			logger.Error("Failed to handle message", err, map[string]interface{}{"message": msg})
			msg.Nack()
		}
	}
}

type PlainTextMarshaler struct{}

func (m *PlainTextMarshaler) Marshal(topic string, msg nc.Msg) ([]byte, error) {
	return msg.Data, nil
}

func (m *PlainTextMarshaler) Unmarshal(newMsg *nc.Msg) (*message.Message, error) {
	if newMsg == nil {
		return nil, errors.New("empty message")
	}
	msg := message.NewMessage(watermill.NewUUID(), newMsg.Data)
	return msg, nil
}
