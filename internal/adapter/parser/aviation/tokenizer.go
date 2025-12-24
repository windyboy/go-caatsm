package aviation

import "strings"

// Token represents a lexeme in the body with its byte offsets.
type Token struct {
	Text  string
	Start int
	End   int
}

// Tokenizer splits text into tokens using a whitespace set.
// A forward slash is treated as whitespace but is emitted as its own token.
type Tokenizer struct {
	Whitespace string
}

// Tokenize tokenizes input and returns tokens with byte offsets.
func (t Tokenizer) Tokenize(input string) []Token {
	if t.Whitespace == "" {
		t.Whitespace = " \n\t\r"
	}

	var tokens []Token
	start := -1

	for idx, r := range input {
		if strings.ContainsRune(t.Whitespace, r) {
			if start != -1 {
				tokens = append(tokens, Token{
					Text:  input[start:idx],
					Start: start,
					End:   idx,
				})
				start = -1
			}
			if r == '/' {
				tokens = append(tokens, Token{
					Text:  "/",
					Start: idx,
					End:   idx + 1,
				})
			}
			continue
		}

		if start == -1 {
			start = idx
		}
	}

	if start != -1 {
		tokens = append(tokens, Token{
			Text:  input[start:],
			Start: start,
			End:   len(input),
		})
	}

	return tokens
}
