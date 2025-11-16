package iface

import "caatsm/internal/infra/config"

type MessageHandler interface {
	HandleMessage(msg []byte, id string) error
}

type MessagePublisher interface {
	Publish(message interface{}) error
}

type MessageSubscriber interface {
	Subscribe(config *config.Config) error
}

type MessageRepository interface {
	CreateNew(message interface{}) error
}
