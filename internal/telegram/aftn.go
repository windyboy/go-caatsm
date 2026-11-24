package telegram

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// AFTNError represents a single AFTN protocol field violation.
type AFTNError struct {
	Field   string
	Value   string
	Message string
}

func (e *AFTNError) Error() string {
	return fmt.Sprintf("AFTN validation error [%s]: %s (value: %q)", e.Field, e.Message, e.Value)
}

// AFTNValidationErrors wraps multiple AFTN validation errors.
type AFTNValidationErrors struct {
	Errors []error
}

func (e *AFTNValidationErrors) Error() string {
	messages := make([]string, len(e.Errors))
	for i, err := range e.Errors {
		messages[i] = err.Error()
	}
	return fmt.Sprintf("AFTN validation failed: %s", strings.Join(messages, "; "))
}

// Unwrap exposes the underlying validation errors for errors.Is / errors.As.
func (e *AFTNValidationErrors) Unwrap() []error {
	return e.Errors
}

var (
	validPriorities = map[string]bool{
		"FF": true, "GG": true, "QU": true,
		"DD": true, "SS": true, "KK": true,
	}

	// Accepts 4-character ICAO location indicators or 8-character AFTN routing addresses.
	aftnAddressPattern = regexp.MustCompile(`^([A-Z0-9]{4}|[A-Z0-9]{8})$`)
	dateTimePattern    = regexp.MustCompile(`^\d{6}$`)
)

// AFTNValidator validates AFTN header envelope fields on parsed messages.
type AFTNValidator struct{}

// NewAFTNValidator constructs an AFTNValidator.
func NewAFTNValidator() *AFTNValidator {
	return &AFTNValidator{}
}

// Validate validates AFTN envelope fields on msg.
func (v *AFTNValidator) Validate(msg *Message) error {
	return ValidateAFTN(msg)
}

// ValidatePriorityIndicator validates AFTN priority indicators (FF, GG, QU, DD, SS, KK).
func ValidatePriorityIndicator(priority string) error {
	priority = strings.TrimSpace(strings.ToUpper(priority))
	if priority == "" {
		return nil
	}
	if !validPriorities[priority] {
		return &AFTNError{
			Field:   "priority_indicator",
			Value:   priority,
			Message: "must be one of FF, GG, QU, DD, SS, KK",
		}
	}
	return nil
}

// ValidateAddress validates a 4-character ICAO code or 8-character AFTN address.
func ValidateAddress(address string) error {
	address = strings.TrimSpace(strings.ToUpper(address))
	if address == "" {
		return nil
	}
	if !aftnAddressPattern.MatchString(address) {
		return &AFTNError{
			Field:   "icao_address",
			Value:   address,
			Message: "must be 4 or 8 uppercase alphanumeric characters",
		}
	}
	return nil
}

// ValidateDateTime validates the 6-digit DDHHMM AFTN filing/originator timestamp.
func ValidateDateTime(dt string) error {
	dt = strings.TrimSpace(dt)
	if dt == "" {
		return nil
	}
	if !dateTimePattern.MatchString(dt) {
		return &AFTNError{
			Field:   "datetime",
			Value:   dt,
			Message: "must be 6 digits (DDHHMM format)",
		}
	}
	day := dt[0:2]
	hour := dt[2:4]
	minute := dt[4:6]
	if !isValidRange(day, 1, 31) || !isValidRange(hour, 0, 23) || !isValidRange(minute, 0, 59) {
		return &AFTNError{
			Field:   "datetime",
			Value:   dt,
			Message: "invalid date/time ranges (DD:01-31, HH:00-23, MM:00-59)",
		}
	}
	return nil
}

// ValidateAFTN validates AFTN envelope fields on non-weather messages.
// Weather reports (METAR, SPECI, TAF) do not carry AFTN headers and are skipped.
// A single violation is returned directly as *AFTNError so metrics preserve field granularity.
func ValidateAFTN(msg *Message) error {
	if msg == nil {
		return nil
	}
	if msg.Protocol == ProtocolWeather ||
		msg.Category == "METAR" ||
		msg.Category == "SPECI" ||
		msg.Category == "TAF" {
		return nil
	}

	var errs []error
	if err := ValidatePriorityIndicator(msg.PriorityIndicator); err != nil {
		errs = append(errs, err)
	}
	if err := ValidateAddress(msg.PrimaryAddress); err != nil {
		errs = append(errs, err)
	}
	if err := ValidateAddress(msg.Originator); err != nil {
		errs = append(errs, err)
	}
	if err := ValidateDateTime(msg.DateTime); err != nil {
		errs = append(errs, err)
	}
	if err := ValidateDateTime(msg.OriginatorDateTime); err != nil {
		errs = append(errs, err)
	}

	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		return &AFTNValidationErrors{Errors: errs}
	}
}

// IsAFTNError checks whether err (or any wrapped error) is an AFTN validation error.
func IsAFTNError(err error) bool {
	if err == nil {
		return false
	}
	var single *AFTNError
	var multi *AFTNValidationErrors
	return errors.As(err, &single) || errors.As(err, &multi)
}

// GetAFTNErrorType extracts the failing field label for observability metrics.
func GetAFTNErrorType(err error) string {
	var multi *AFTNValidationErrors
	if errors.As(err, &multi) {
		return "multiple_errors"
	}
	var single *AFTNError
	if errors.As(err, &single) {
		return single.Field
	}
	return "unknown"
}

func isValidRange(s string, minVal, maxVal int) bool {
	val, err := strconv.Atoi(s)
	return err == nil && val >= minVal && val <= maxVal
}
