package parser

// ProvideParser creates a parser instance
func ProvideParser() Parser {
	return NewAviationParser()
}

