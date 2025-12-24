package parser

import (
	"caatsm/internal/adapter/dto"
	"caatsm/internal/port"
)

// AviationParser implements the Parser interface
type AviationParser struct{}

// Parse parses a raw message string and returns a ParsedTelegram
func (p *AviationParser) Parse(rawText string) (*dto.ParsedTelegram, error) {
	return Parse(rawText)
}

// ProvideParser creates a composite parser instance that combines weather and aviation parsers
func ProvideParser(weatherParser port.WeatherParser) Parser {
	aviation := &AviationParser{}
	return NewCompositeParser(aviation, weatherParser)
}

