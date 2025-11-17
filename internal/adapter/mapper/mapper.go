package mapper

import "caatsm/internal/adapter/dto"

// Mapper defines the interface for mapping between pipeline models and database models
type Mapper interface {
	// ToDBRow converts a ParsedTelegram to a database row representation
	ToDBRow(msg *dto.ParsedTelegram) ([]interface{}, error)

	// FromDBRow converts a database row to a ParsedTelegram
	FromDBRow(row []interface{}) (*dto.ParsedTelegram, error)
}

