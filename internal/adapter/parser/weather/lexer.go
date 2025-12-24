package weather

import "strings"

// Tokenize splits the raw text into tokens by whitespace
func Tokenize(raw string) []string {
	raw = strings.TrimSpace(raw)
	// Remove trailing '=' if present
	if strings.HasSuffix(raw, "=") {
		raw = raw[:len(raw)-1]
		raw = strings.TrimSpace(raw)
	}

	tokens := strings.Fields(raw)
	return tokens
}
