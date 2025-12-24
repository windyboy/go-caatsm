package validator

import (
	"caatsm/internal/adapter/dto"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// AFTNError represents an AFTN protocol violation
type AFTNError struct {
	Field   string // e.g., "priority_indicator", "icao_address"
	Value   string
	Message string
}

func (e *AFTNError) Error() string {
	return fmt.Sprintf("AFTN validation error [%s]: %s (value: %q)", e.Field, e.Message, e.Value)
}

// AFTN field validators
var (
	// Priority indicators: FF (Flash), GG (Immediate), QU (Distress), DD (Delay), SS (Service), KK (Correction)
	validPriorities = map[string]bool{
		"FF": true, "GG": true, "QU": true,
		"DD": true, "SS": true, "KK": true,
	}

	// ICAO address: 4 uppercase alphanumeric characters
	icaoAddressPattern = regexp.MustCompile(`^[A-Z0-9]{4}$`)

	// DateTime: DDHHMM (6 digits)
	dateTimePattern = regexp.MustCompile(`^\d{6}$`)
)

// ValidatePriorityIndicator validates AFTN priority indicator
func ValidatePriorityIndicator(priority string) error {
	priority = strings.TrimSpace(strings.ToUpper(priority))
	if priority == "" {
		return nil // Optional field
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

// ValidateICAOAddress validates 4-character ICAO address
func ValidateICAOAddress(address string) error {
	address = strings.TrimSpace(strings.ToUpper(address))
	if address == "" {
		return nil // Optional field
	}
	if !icaoAddressPattern.MatchString(address) {
		return &AFTNError{
			Field:   "icao_address",
			Value:   address,
			Message: "must be 4 uppercase alphanumeric characters",
		}
	}
	return nil
}

// ValidateDateTime validates DDHHMM format
func ValidateDateTime(dt string) error {
	dt = strings.TrimSpace(dt)
	if dt == "" {
		return nil // Optional field
	}
	if !dateTimePattern.MatchString(dt) {
		return &AFTNError{
			Field:   "datetime",
			Value:   dt,
			Message: "must be 6 digits (DDHHMM format)",
		}
	}
	// Additional semantic validation
	if len(dt) == 6 {
		day := dt[0:2]
		hour := dt[2:4]
		minute := dt[4:6]
		// Basic range checks
		if !isValidRange(day, 1, 31) || !isValidRange(hour, 0, 23) || !isValidRange(minute, 0, 59) {
			return &AFTNError{
				Field:   "datetime",
				Value:   dt,
				Message: "invalid date/time ranges (DD:01-31, HH:00-23, MM:00-59)",
			}
		}
	}
	return nil
}

// ValidateTelegram validates all AFTN fields in ParsedTelegram
func ValidateTelegram(telegram *dto.ParsedTelegram) error {
	if telegram == nil {
		return nil
	}

	var errors []error

	// Validate priority indicator
	if err := ValidatePriorityIndicator(telegram.PriorityIndicator); err != nil {
		errors = append(errors, err)
	}

	// Validate primary address (ICAO)
	if err := ValidateICAOAddress(telegram.PrimaryAddress); err != nil {
		errors = append(errors, err)
	}

	// Validate originator (ICAO)
	if err := ValidateICAOAddress(telegram.Originator); err != nil {
		errors = append(errors, err)
	}

	// Validate datetime
	if err := ValidateDateTime(telegram.DateTime); err != nil {
		errors = append(errors, err)
	}

	// Validate originator datetime
	if err := ValidateDateTime(telegram.OriginatorDateTime); err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return &AFTNValidationErrors{Errors: errors}
	}
	return nil
}

// AFTNValidationErrors wraps multiple validation errors
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

// IsAFTNError checks if error is an AFTN validation error
func IsAFTNError(err error) bool {
	if err == nil {
		return false
	}
	_, ok1 := err.(*AFTNError)
	_, ok2 := err.(*AFTNValidationErrors)
	return ok1 || ok2
}

// GetAFTNErrorType extracts the error type for metrics labeling
func GetAFTNErrorType(err error) string {
	if aftnErr, ok := err.(*AFTNError); ok {
		return aftnErr.Field
	}
	if _, ok := err.(*AFTNValidationErrors); ok {
		return "multiple_errors"
	}
	return "unknown"
}

// isValidRange checks if a numeric string is within the specified range
func isValidRange(s string, min, max int) bool {
	val, err := strconv.Atoi(s)
	return err == nil && val >= min && val <= max
}
