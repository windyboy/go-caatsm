package parser

import (
	"caatsm/internal/domain"
	"caatsm/internal/parsers"
)

// AviationParser implements the Parser interface using the existing parsers package
type AviationParser struct{}

// NewAviationParser creates a new aviation parser
func NewAviationParser() *AviationParser {
	return &AviationParser{}
}

// Parse parses a raw message string and returns a ParsedMessage
func (p *AviationParser) Parse(rawText string) (*domain.ParsedMessage, error) {
	// Use the existing Parse function from internal/parsers
	return parsers.Parse(rawText)
}
