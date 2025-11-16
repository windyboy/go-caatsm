package parser

import (
	"caatsm/internal/model"
	"caatsm/internal/parsers"
)

// AviationParser implements the Parser interface using the existing parsers package
type AviationParser struct{}

// NewAviationParser creates a new aviation parser
func NewAviationParser() *AviationParser {
	return &AviationParser{}
}

// Parse parses a raw message string and returns a ParsedTelegram
func (p *AviationParser) Parse(rawText string) (*model.ParsedTelegram, error) {
	// Use the existing Parse function from internal/parsers
	return parsers.Parse(rawText)
}
