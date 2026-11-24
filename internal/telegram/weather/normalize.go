package weather

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseTime parses a time string in DDHHmmZ format to time.Time.
func ParseTime(timeStr string) (time.Time, error) {
	match := timePattern.FindStringSubmatch(timeStr)
	if len(match) == 0 {
		return time.Time{}, fmt.Errorf("invalid time format: %s", timeStr)
	}

	day, err := strconv.Atoi(match[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid day: %w", err)
	}
	hour, err := strconv.Atoi(match[2])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid hour: %w", err)
	}
	min, err := strconv.Atoi(match[3])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid minute: %w", err)
	}

	now := time.Now()
	t := time.Date(now.Year(), now.Month(), day, hour, min, 0, 0, time.UTC)
	if day < now.Day()-15 {
		t = t.AddDate(0, 1, 0)
	}
	return t, nil
}

// ParseTAFValidity parses TAF validity period in DDHH/DDHH format.
func ParseTAFValidity(validityStr string, issueTime time.Time) (time.Time, time.Time, error) {
	match := tafValidityPattern.FindStringSubmatch(validityStr)
	if len(match) == 0 {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid TAF validity format: %s", validityStr)
	}

	fromDay, _ := strconv.Atoi(match[1])
	fromHour, _ := strconv.Atoi(match[2])
	toDay, _ := strconv.Atoi(match[3])
	toHour, _ := strconv.Atoi(match[4])

	year := issueTime.Year()
	month := issueTime.Month()

	fromTime := time.Date(year, month, fromDay, fromHour, 0, 0, 0, time.UTC)
	toTime := time.Date(year, month, toDay, toHour, 0, 0, 0, time.UTC)

	if toDay < fromDay {
		toTime = toTime.AddDate(0, 1, 0)
	}
	return fromTime, toTime, nil
}

// ParseFraction parses a fraction string like "1/4" or "1 1/2".
func ParseFraction(fracStr string) (float64, error) {
	fracStr = strings.TrimSpace(fracStr)
	if strings.Contains(fracStr, " ") {
		parts := strings.Fields(fracStr)
		if len(parts) != 2 {
			return 0, fmt.Errorf("invalid fraction format: %s", fracStr)
		}
		whole, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid whole number: %w", err)
		}
		frac, err := parseSimpleFraction(parts[1])
		if err != nil {
			return 0, err
		}
		return whole + frac, nil
	}
	return parseSimpleFraction(fracStr)
}

func parseSimpleFraction(fracStr string) (float64, error) {
	parts := strings.Split(fracStr, "/")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid fraction format: %s", fracStr)
	}
	numerator, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numerator: %w", err)
	}
	denominator, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid denominator: %w", err)
	}
	if denominator == 0 {
		return 0, fmt.Errorf("division by zero")
	}
	return numerator / denominator, nil
}
