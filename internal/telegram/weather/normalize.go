package weather

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseTime parses a time string in DDHHmmZ format relative to current UTC time.
func ParseTime(timeStr string) (time.Time, error) {
	return parseTimeAt(timeStr, time.Now().UTC())
}

func parseTimeAt(timeStr string, now time.Time) (time.Time, error) {
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

	if day < 1 || day > 31 {
		return time.Time{}, fmt.Errorf("day out of range: %d", day)
	}
	if hour < 0 || hour > 23 {
		return time.Time{}, fmt.Errorf("hour out of range: %d", hour)
	}
	if min < 0 || min > 59 {
		return time.Time{}, fmt.Errorf("minute out of range: %d", min)
	}

	now = now.UTC()
	year, month := now.Year(), now.Month()
	switch {
	case day > now.Day()+15:
		prev := time.Date(year, month-1, 1, 0, 0, 0, 0, time.UTC)
		year, month = prev.Year(), prev.Month()
	case day < now.Day()-15:
		next := time.Date(year, month+1, 1, 0, 0, 0, 0, time.UTC)
		year, month = next.Year(), next.Month()
	}

	t := time.Date(year, month, day, hour, min, 0, 0, time.UTC)
	if t.Day() != day {
		return time.Time{}, fmt.Errorf("invalid day %d for %s %d", day, month, year)
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

	if fromDay < 1 || fromDay > 31 || toDay < 1 || toDay > 31 {
		return time.Time{}, time.Time{}, fmt.Errorf("TAF validity day out of range: %s", validityStr)
	}
	if fromHour < 0 || fromHour > 24 || toHour < 0 || toHour > 24 {
		return time.Time{}, time.Time{}, fmt.Errorf("TAF validity hour out of range: %s", validityStr)
	}

	issueUTC := issueTime.UTC()
	fromYear, fromMonth := issueUTC.Year(), issueUTC.Month()
	if fromDay < issueUTC.Day()-15 {
		next := time.Date(fromYear, fromMonth+1, 1, 0, 0, 0, 0, time.UTC)
		fromYear, fromMonth = next.Year(), next.Month()
	}
	if time.Date(fromYear, fromMonth, fromDay, 0, 0, 0, 0, time.UTC).Day() != fromDay {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid TAF validity start day %d for %s %d", fromDay, fromMonth, fromYear)
	}
	fromTime := time.Date(fromYear, fromMonth, fromDay, fromHour, 0, 0, 0, time.UTC)

	toYear, toMonth := fromYear, fromMonth
	if toDay < fromDay {
		next := time.Date(toYear, toMonth+1, 1, 0, 0, 0, 0, time.UTC)
		toYear, toMonth = next.Year(), next.Month()
	}
	if time.Date(toYear, toMonth, toDay, 0, 0, 0, 0, time.UTC).Day() != toDay {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid TAF validity end day %d for %s %d", toDay, toMonth, toYear)
	}
	toTime := time.Date(toYear, toMonth, toDay, toHour, 0, 0, 0, time.UTC)

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
