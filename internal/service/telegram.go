package service

import (
	"context"
	"fmt"
	"time"

	"caatsm/internal/domain"
	"caatsm/internal/iface"
)

type TelegramService struct {
	repo iface.TelegramRepository
}

func NewTelegramService(repo iface.TelegramRepository) *TelegramService {
	return &TelegramService{
		repo: repo,
	}
}

// CreateTelegram creates a new telegram message
func (s *TelegramService) CreateTelegram(ctx context.Context, msg *domain.ParsedMessage) error {
	if msg == nil {
		return fmt.Errorf("message cannot be nil")
	}

	if msg.Uuid == "" {
		return fmt.Errorf("message UUID is required")
	}

	return s.repo.Create(ctx, msg)
}

// GetTelegram retrieves a telegram by ID
func (s *TelegramService) GetTelegram(ctx context.Context, id string) (*domain.ParsedMessage, error) {
	if id == "" {
		return nil, fmt.Errorf("telegram ID is required")
	}

	return s.repo.GetByID(ctx, id)
}

// ListTelegrams lists telegrams with filters and pagination
func (s *TelegramService) ListTelegrams(ctx context.Context, filters *iface.TelegramFilters) ([]*domain.ParsedMessage, int64, error) {
	if filters == nil {
		filters = &iface.TelegramFilters{}
	}

	// Set default pagination
	if filters.Limit <= 0 {
		filters.Limit = 100
	}
	if filters.Limit > 1000 {
		filters.Limit = 1000 // Max limit
	}

	if filters.OrderBy == "" {
		filters.OrderBy = "received_at"
	}
	if filters.OrderDirection == "" {
		filters.OrderDirection = "DESC"
	}

	return s.repo.List(ctx, filters)
}

// GetTelegramsByTimeRange retrieves telegrams within a time range
func (s *TelegramService) GetTelegramsByTimeRange(ctx context.Context, start, end time.Time) ([]*domain.ParsedMessage, error) {
	if start.After(end) {
		return nil, fmt.Errorf("start time must be before end time")
	}

	// Limit time range to 30 days
	maxRange := 30 * 24 * time.Hour
	if end.Sub(start) > maxRange {
		return nil, fmt.Errorf("time range cannot exceed 30 days")
	}

	return s.repo.GetByTimeRange(ctx, start, end)
}

