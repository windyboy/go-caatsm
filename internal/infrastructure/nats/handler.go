package nats

import (
	"context"

	"caatsm/internal/config"
	"caatsm/internal/iface"
	"caatsm/internal/service"
)

// MessageHandler processes incoming NATS messages via the message service.
type MessageHandler struct {
	messageService *service.MessageService
}

func NewHandler(cfg *config.Config, publisher iface.MessagePublisher, repository iface.TelegramRepository) *MessageHandler {
	return &MessageHandler{
		messageService: service.NewMessageService(cfg, repository, publisher),
	}
}

func (h *MessageHandler) HandleMessage(ctx context.Context, msg []byte, id string) error {
	return h.messageService.ProcessMessage(ctx, msg, id)
}
