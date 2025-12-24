package aviation

import "caatsm/internal/adapter/dto"

// AviationParser implements the Parser interface for aviation telegrams.
type AviationParser struct{}

// NewParser creates a new aviation parser instance.
func NewParser() *AviationParser {
	return &AviationParser{}
}

// Parse parses a raw message string and returns a ParsedTelegram.
func (p *AviationParser) Parse(rawText string) (*dto.ParsedTelegram, error) {
	return Parse(rawText)
}
