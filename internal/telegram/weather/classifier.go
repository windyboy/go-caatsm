package weather

import (
	"regexp"
	"strings"
)

var (
	metarPattern = regexp.MustCompile(`^(METAR|SPECI)\s+[A-Z0-9]{4}`)
	tafPattern   = regexp.MustCompile(`^TAF\s+[A-Z0-9]{4}`)
)

// Classify identifies the weather report type from raw text.
func Classify(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	if metarPattern.MatchString(raw) {
		if strings.HasPrefix(raw, "SPECI") {
			return "SPECI", true
		}
		return "METAR", true
	}
	if tafPattern.MatchString(raw) {
		return "TAF", true
	}
	return "", false
}

// HasValidEnding checks if the report ends with '='.
func HasValidEnding(raw string) bool {
	return strings.HasSuffix(strings.TrimSpace(raw), "=")
}
