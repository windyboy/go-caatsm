package weather

import (
	"regexp"
	"strings"
)

var (
	// metarPattern matches METAR or SPECI at the start followed by station code
	metarPattern = regexp.MustCompile(`^(METAR|SPECI)\s+[A-Z0-9]{4}`)
	// tafPattern matches TAF at the start followed by station code
	tafPattern = regexp.MustCompile(`^TAF\s+[A-Z0-9]{4}`)
)

// Classify identifies the weather report type from raw text
// Returns the report type (METAR, SPECI, or TAF) and true if it's a weather report
func Classify(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}

	// Check for METAR/SPECI
	if metarPattern.MatchString(raw) {
		if strings.HasPrefix(raw, "SPECI") {
			return "SPECI", true
		}
		return "METAR", true
	}

	// Check for TAF
	if tafPattern.MatchString(raw) {
		return "TAF", true
	}

	return "", false
}

// HasValidEnding checks if the report ends with '='
func HasValidEnding(raw string) bool {
	raw = strings.TrimSpace(raw)
	return strings.HasSuffix(raw, "=")
}

