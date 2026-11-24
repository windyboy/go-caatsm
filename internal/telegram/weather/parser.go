package weather

import "fmt"

// Parser parses METAR, SPECI, and TAF aviation weather reports.
type Parser struct{}

// NewParser creates a new weather parser instance.
func NewParser() *Parser {
	return &Parser{}
}

// CanParse reports whether raw text starts with a supported weather report header
// and terminates with '='.
func (p *Parser) CanParse(raw string) bool {
	if _, ok := Classify(raw); !ok {
		return false
	}
	return HasValidEnding(raw)
}

// Parse parses a raw weather report string and returns a typed weather Message.
func (p *Parser) Parse(raw string) (Message, error) {
	reportType, ok := Classify(raw)
	if !ok {
		return nil, ErrInvalidFormat
	}

	switch reportType {
	case "METAR", "SPECI":
		return parseMetar(raw, reportType)
	case "TAF":
		return parseTaf(raw)
	default:
		return nil, fmt.Errorf("unsupported report type: %s", reportType)
	}
}
