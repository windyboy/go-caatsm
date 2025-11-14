package parser

import "caatsm/internal/domain"

// Parser defines the interface for parsing raw telegram messages
type Parser interface {
	// Parse parses a raw message string and returns a ParsedMessage
	Parse(rawText string) *domain.ParsedMessage

	// TODO: consider returning (*domain.ParsedMessage, error) to surface parse failures explicitly.
}
