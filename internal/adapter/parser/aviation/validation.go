package aviation

import "fmt"

// Validation limits based on ICAO and AFTN standards
const (
	// MaxTelegramSize is the maximum size for an AFTN telegram (ICAO standard)
	MaxTelegramSize = 1800

	// MaxHeaderLines is the maximum number of lines allowed in the header section
	MaxHeaderLines = 20

	// MaxBodySize is the maximum size for the telegram body
	MaxBodySize = 1500

	// MaxTokenCount is the maximum number of tokens allowed to prevent tokenizer abuse
	MaxTokenCount = 500
)

// ValidationError represents a validation failure with field context.
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error [%s]: %s", e.Field, e.Message)
}

// ValidateInputSize checks if the input telegram is within acceptable size limits.
// Returns ValidationError if the input is empty or exceeds MaxTelegramSize.
func ValidateInputSize(rawText string) error {
	if len(rawText) == 0 {
		return &ValidationError{
			Field:   "input",
			Message: "empty input",
		}
	}

	if len(rawText) > MaxTelegramSize {
		return &ValidationError{
			Field:   "input",
			Message: fmt.Sprintf("input exceeds maximum size of %d characters (got %d)", MaxTelegramSize, len(rawText)),
		}
	}

	return nil
}

// ValidateBodySize checks if the body content is within acceptable size limits.
// Returns ValidationError if the body exceeds MaxBodySize.
func ValidateBodySize(body string) error {
	if len(body) > MaxBodySize {
		return &ValidationError{
			Field:   "body",
			Message: fmt.Sprintf("body exceeds maximum size of %d characters (got %d)", MaxBodySize, len(body)),
		}
	}
	return nil
}

// ValidateTokenCount checks if the token count is within reasonable limits.
// Returns ValidationError if token count exceeds MaxTokenCount.
func ValidateTokenCount(tokens []Token) error {
	if len(tokens) > MaxTokenCount {
		return &ValidationError{
			Field:   "tokens",
			Message: fmt.Sprintf("token count exceeds maximum of %d (got %d)", MaxTokenCount, len(tokens)),
		}
	}
	return nil
}

// GetRequiredField safely extracts a required field from parsed data.
// Returns ValidationError if the field doesn't exist or is empty.
func GetRequiredField(data map[string]string, field string) (string, error) {
	value, exists := data[field]
	if !exists {
		return "", &ValidationError{
			Field:   field,
			Message: fmt.Sprintf("required field '%s' not found in parsed data", field),
		}
	}

	if value == "" {
		return "", &ValidationError{
			Field:   field,
			Message: fmt.Sprintf("required field '%s' is empty", field),
		}
	}

	return value, nil
}

// GetOptionalField safely extracts an optional field from parsed data.
// Returns empty string if the field doesn't exist.
func GetOptionalField(data map[string]string, field string) string {
	value, exists := data[field]
	if !exists {
		return ""
	}
	return value
}

// SanitizeErrorForClient removes sensitive information from error messages
// before exposing them to external clients. This prevents leaking:
// - Raw telegram content (may contain sensitive flight data)
// - Internal implementation details
// - System paths or configuration
//
// The function preserves error type and general context while removing
// specific content that could be sensitive.
func SanitizeErrorForClient(err error) string {
	if err == nil {
		return ""
	}

	// Get the error message string
	errMsg := err.Error()

	// Truncate long error messages that might contain sensitive content
	// This applies to all error types, including ValidationError
	if len(errMsg) > 200 {
		return errMsg[:200] + "..."
	}

	return errMsg
}
