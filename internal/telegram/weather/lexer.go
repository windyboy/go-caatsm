package weather

import "strings"

// Tokenize splits raw text into whitespace-delimited tokens after stripping trailing '=',
// normalizing tokens before RMK to uppercase while preserving free-text remarks casing.
func Tokenize(raw string) []string {
	raw = strings.TrimSpace(raw)
	if strings.HasSuffix(raw, "=") {
		raw = strings.TrimSpace(raw[:len(raw)-1])
	}
	tokens := strings.Fields(raw)
	for i, tok := range tokens {
		upper := strings.ToUpper(tok)
		if strings.HasPrefix(upper, "RMK") {
			break
		}
		tokens[i] = upper
	}
	return tokens
}
