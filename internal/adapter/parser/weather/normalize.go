package weather

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ConvertKTToMPS converts knots to meters per second
func ConvertKTToMPS(knots int) int {
	// 1 knot = 0.514444 m/s
	return int(float64(knots) * 0.514444)
}

// ConvertMPSToKT converts meters per second to knots
func ConvertMPSToKT(mps int) int {
	// 1 m/s = 1.94384 knots
	return int(float64(mps) * 1.94384)
}

// ConvertSMToMeters converts statute miles to meters
func ConvertSMToMeters(sm float64) float64 {
	// 1 SM = 1609.34 meters
	return sm * 1609.34
}

// ConvertMetersToSM converts meters to statute miles
func ConvertMetersToSM(meters float64) float64 {
	// 1 meter = 0.000621371 SM
	return meters * 0.000621371
}

// ConvertInHgToHPa converts inches of mercury to hectopascals
func ConvertInHgToHPa(inHg float64) float64 {
	// 1 inHg = 33.8639 hPa
	return inHg * 33.8639
}

// ConvertHPatoInHg converts hectopascals to inches of mercury
func ConvertHPatoInHg(hPa float64) float64 {
	// 1 hPa = 0.0295299 inHg
	return hPa * 0.0295299
}

// ParseTime parses a time string in DDHHmmZ format to time.Time
// Uses the current year/month as reference
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
	// Use current year and month, but adjust if day is in the future (likely next month)
	t := time.Date(now.Year(), now.Month(), day, hour, min, 0, 0, time.UTC)

	// If the day is significantly in the past (more than 15 days), assume next month
	if day < now.Day()-15 {
		t = t.AddDate(0, 1, 0)
	}

	return t, nil
}

// ParseTAFValidity parses TAF validity period in DDHH/DDHH format
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

	// If toDay is less than fromDay, assume next month
	if toDay < fromDay {
		toTime = toTime.AddDate(0, 1, 0)
	}

	return fromTime, toTime, nil
}

// ParseFraction parses a fraction string like "1/4" or "1 1/2"
func ParseFraction(fracStr string) (float64, error) {
	fracStr = strings.TrimSpace(fracStr)

	// Handle whole number with fraction: "1 1/2"
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

	// Handle simple fraction: "1/4"
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

