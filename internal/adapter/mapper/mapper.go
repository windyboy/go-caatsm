package mapper

import "caatsm/internal/domain"

// Mapper defines the interface for mapping between domain models and database models
type Mapper interface {
	// ToDBRow converts a domain.ParsedMessage to a database row representation
	ToDBRow(msg *domain.ParsedMessage) ([]interface{}, error)
	
	// FromDBRow converts a database row to a domain.ParsedMessage
	FromDBRow(row []interface{}) (*domain.ParsedMessage, error)
}

