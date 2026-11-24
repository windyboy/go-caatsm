package weather

import "strings"

// Tokenize splits the raw text into tokens by whitespace after stripping trailing '='.
func Tokenize(raw string) []string {
	raw = strings.TrimSpace(raw)
	if strings.HasSuffix(raw, "=") {
		raw = strings.TrimSpace(raw[:len(raw)-1])
	}
	return strings.Fields(raw)
}
