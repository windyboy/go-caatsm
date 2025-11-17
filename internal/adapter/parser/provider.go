package parser

import "caatsm/internal/adapter/dto"

// AviationParser implements the Parser interface
type AviationParser struct{}

// Parse parses a raw message string and returns a ParsedTelegram
func (p *AviationParser) Parse(rawText string) (*dto.ParsedTelegram, error) {
	return Parse(rawText)
}

// ProvideParser creates a parser instance
func ProvideParser() Parser {
	return &AviationParser{}
}

