package service

import (
	"context"
	"fmt"

	"caatsm/internal/config"
	"caatsm/internal/iface"
	"caatsm/internal/parsers"
	"caatsm/pkg/utils"
)

const MaxMessageSize = 10 * 1024 * 1024 // 10MB

type MessageService struct {
	cfg       *config.Config
	repo      iface.TelegramRepository
	publisher iface.MessagePublisher
}

func NewMessageService(cfg *config.Config, repo iface.TelegramRepository, publisher iface.MessagePublisher) *MessageService {
	return &MessageService{
		cfg:       cfg,
		repo:      repo,
		publisher: publisher,
	}
}

// ProcessMessage parses, validates, stores, and publishes a message payload.
func (s *MessageService) ProcessMessage(ctx context.Context, rawMessage []byte, messageID string) error {
	log := utils.GetSugaredLogger()

	if len(rawMessage) == 0 {
		log.Error("empty message")
		return fmt.Errorf("empty message")
	}

	if len(rawMessage) > MaxMessageSize {
		return fmt.Errorf("message size %d exceeds maximum %d", len(rawMessage), MaxMessageSize)
	}

	payload := string(rawMessage)
	parsed := parsers.Parse(payload)
	if parsed == nil {
		return fmt.Errorf("parser returned nil message")
	}

	if !parsed.Parsed {
		log.Infof("not parsed: [%s] (message length: %d)", messageID, len(payload))
	} else {
		parsed.Uuid = messageID
		log.Infof("parsed [%s]: %v", messageID, parsed.ToString())
	}

	if err := s.repo.Create(ctx, parsed); err != nil {
		log.Errorf("Failed to save message [%s]: %v", messageID, err)
		return fmt.Errorf("repository error: %w", err)
	}

	subject := s.cfg.Publisher.Topic
	if subject == "" {
		return fmt.Errorf("publisher topic is not configured")
	}

	if err := s.publisher.Publish(ctx, subject, parsed); err != nil {
		log.Errorf("Failed to publish message [%s]: %v", messageID, err)
		return fmt.Errorf("publish error: %w", err)
	}

	return nil
}
