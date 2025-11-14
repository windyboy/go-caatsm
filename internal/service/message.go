package service

import (
	"context"
	"fmt"
	"time"

	"caatsm/internal/config"
	"caatsm/internal/iface"
	applog "caatsm/internal/infra/log"
	"caatsm/internal/observability/metrics"
	"caatsm/internal/parsers"
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
	log := applog.Sugared()
	start := time.Now()
	status := "success"
	categoryLabel := "unknown"

	defer func() {
		duration := time.Since(start).Seconds()
		metrics.MessageProcessingDuration.WithLabelValues(categoryLabel).Observe(duration)
		metrics.MessagesProcessedTotal.WithLabelValues(status, categoryLabel).Inc()
	}()

	if len(rawMessage) == 0 {
		log.Error("empty message")
		metrics.MessageProcessingFailures.WithLabelValues("validation").Inc()
		status = "failure"
		return fmt.Errorf("empty message")
	}

	if len(rawMessage) > MaxMessageSize {
		metrics.MessageProcessingFailures.WithLabelValues("validation").Inc()
		status = "failure"
		return fmt.Errorf("message size %d exceeds maximum %d", len(rawMessage), MaxMessageSize)
	}

	payload := string(rawMessage)
	parsed := parsers.Parse(payload)
	if parsed == nil {
		metrics.MessageProcessingFailures.WithLabelValues("parser").Inc()
		status = "failure"
		return fmt.Errorf("parser returned nil message")
	}
	if parsed.Category != "" {
		categoryLabel = parsed.Category
	}

	if !parsed.Parsed {
		log.Infof("not parsed: [%s] (message length: %d)", messageID, len(payload))
	} else {
		parsed.Uuid = messageID
		log.Infof("parsed [%s]: %v", messageID, parsed.ToString())
	}

	if err := s.repo.Create(ctx, parsed); err != nil {
		log.Errorf("Failed to save message [%s]: %v", messageID, err)
		metrics.MessageProcessingFailures.WithLabelValues("repository").Inc()
		status = "failure"
		return fmt.Errorf("repository error: %w", err)
	}

	subject := s.cfg.Publisher.Topic
	if subject == "" {
		metrics.MessageProcessingFailures.WithLabelValues("publisher").Inc()
		status = "failure"
		return fmt.Errorf("publisher topic is not configured")
	}

	if err := s.publisher.Publish(ctx, subject, parsed); err != nil {
		log.Errorf("Failed to publish message [%s]: %v", messageID, err)
		metrics.MessageProcessingFailures.WithLabelValues("publisher").Inc()
		status = "failure"
		return fmt.Errorf("publish error: %w", err)
	}

	metrics.NATSMessagesPublished.WithLabelValues(subject).Inc()

	return nil
}

// Topic returns the configured publisher topic.
func (s *MessageService) Topic() string {
	if s.cfg == nil {
		return ""
	}
	return s.cfg.Publisher.Topic
}
