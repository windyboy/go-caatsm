package iface

import (
	"caatsm/internal/config"
	"caatsm/internal/domain"
)

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
	CreateNew(message *domain.ParsedMessage) error
}
