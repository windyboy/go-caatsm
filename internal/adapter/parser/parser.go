package parser

import "caatsm/internal/adapter/dto"

// Parser defines the interface for parsing raw telegram messages
type Parser interface {
	// Parse parses a raw message string and returns a ParsedTelegram
	Parse(rawText string) (*dto.ParsedTelegram, error)
}
