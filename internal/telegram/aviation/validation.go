package aviation

import "fmt"

// MaxTelegramSize is the maximum size for an AFTN telegram (ICAO standard).
const MaxTelegramSize = 1800

// ValidationError represents a validation failure with field context.
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error [%s]: %s", e.Field, e.Message)
}

// ValidateInputSize checks if the input telegram is non-empty and within MaxTelegramSize.
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

// GetRequiredField safely extracts a required non-empty field from parsed data.
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
func GetOptionalField(data map[string]string, field string) string {
	return data[field]
}
