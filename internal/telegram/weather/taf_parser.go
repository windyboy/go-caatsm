package weather

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func isTafPeriodBoundary(token string) bool {
	upper := strings.ToUpper(token)
	return strings.HasPrefix(upper, "FM") ||
		strings.HasPrefix(upper, "TEMPO") ||
		strings.HasPrefix(upper, "BECMG") ||
		strings.HasPrefix(upper, "PROB") ||
		strings.HasPrefix(upper, "RMK")
}

func parseTaf(raw string) (*Taf, error) {
	tokens := Tokenize(raw)
	if len(tokens) < 4 {
		return nil, ErrInvalidFormat
	}

	taf, pos, err := parseTafHeader(tokens, raw)
	if err != nil {
		return nil, err
	}

	mainPeriod := TafPeriod{
		Type:      "MAIN",
		ValidFrom: taf.ValidFrom,
		ValidTo:   taf.ValidTo,
	}

	firstSpecialIdx := findNextBoundary(tokens, pos)
	if firstSpecialIdx > pos {
		if firstSpecialIdx < len(tokens) && strings.HasPrefix(strings.ToUpper(tokens[firstSpecialIdx]), "FM") {
			if match := tafFMPattern.FindStringSubmatch(tokens[firstSpecialIdx]); match != nil {
				day, _ := strconv.Atoi(match[1])
				hour, _ := strconv.Atoi(match[2])
				min, _ := strconv.Atoi(match[3])
				fmStart := resolveDayTime(taf.ValidFrom, day, hour, min)
				if fmStart.Before(mainPeriod.ValidTo) {
					mainPeriod.ValidTo = fmStart
				}
			}
		}

		parsePeriodElements(tokens[pos:firstSpecialIdx], &mainPeriod, &taf.Warnings)
		taf.Periods = append(taf.Periods, mainPeriod)
		pos = firstSpecialIdx
	}

	var pendingProb int
	for pos < len(tokens) {
		upperToken := strings.ToUpper(tokens[pos])

		switch {
		case strings.HasPrefix(upperToken, "PROB"):
			probStr := strings.TrimPrefix(upperToken, "PROB")
			probVal, err := strconv.Atoi(probStr)
			if err == nil && probVal > 0 && pos+1 < len(tokens) && tafValidityPattern.MatchString(tokens[pos+1]) {
				period, newPos, pErr := parseRangePeriod("PROB", nil, tokens, pos, taf.ValidFrom, &taf.Warnings)
				if pErr != nil {
					taf.Warnings = append(taf.Warnings, fmt.Sprintf("failed to parse PROB period: %v", pErr))
					pos++
					continue
				}
				period.Probability = probVal
				taf.Periods = append(taf.Periods, period)
				pendingProb = 0
				pos = newPos
				continue
			}
			pendingProb = probVal
			pos++

		case strings.HasPrefix(upperToken, "FM"):
			period, newPos, err := parseFMPeriod(tokens, pos, taf.ValidFrom, taf.ValidTo, &taf.Warnings)
			if err != nil {
				taf.Warnings = append(taf.Warnings, fmt.Sprintf("failed to parse FM period: %v", err))
				pos++
				continue
			}
			if pendingProb > 0 {
				period.Probability = pendingProb
				pendingProb = 0
			}
			taf.Periods = append(taf.Periods, period)
			pos = newPos

		case strings.HasPrefix(upperToken, "TEMPO"):
			period, newPos, err := parseRangePeriod("TEMPO", tafTEMPOPattern, tokens, pos, taf.ValidFrom, &taf.Warnings)
			if err != nil {
				taf.Warnings = append(taf.Warnings, fmt.Sprintf("failed to parse TEMPO period: %v", err))
				pos++
				continue
			}
			if pendingProb > 0 {
				period.Probability = pendingProb
				pendingProb = 0
			}
			taf.Periods = append(taf.Periods, period)
			pos = newPos

		case strings.HasPrefix(upperToken, "BECMG"):
			period, newPos, err := parseRangePeriod("BECMG", tafBECMGPattern, tokens, pos, taf.ValidFrom, &taf.Warnings)
			if err != nil {
				taf.Warnings = append(taf.Warnings, fmt.Sprintf("failed to parse BECMG period: %v", err))
				pos++
				continue
			}
			if pendingProb > 0 {
				period.Probability = pendingProb
				pendingProb = 0
			}
			taf.Periods = append(taf.Periods, period)
			pos = newPos

		case strings.HasPrefix(upperToken, "RMK"):
			taf.Remarks = strings.Join(tokens[pos:], " ")
			pos = len(tokens)

		default:
			taf.Warnings = append(taf.Warnings, fmt.Sprintf("unrecognized token: %s", tokens[pos]))
			pos++
		}
	}

	return taf, nil
}

func findNextBoundary(tokens []string, start int) int {
	for i := start; i < len(tokens); i++ {
		if isTafPeriodBoundary(tokens[i]) {
			return i
		}
	}
	return len(tokens)
}

func parseTafHeader(tokens []string, raw string) (*Taf, int, error) {
	taf := &Taf{
		ReportType: ReportTypeTAF,
		RawTextVal: raw,
		Warnings:   []string{},
		Periods:    []TafPeriod{},
	}

	pos := 0
	if pos >= len(tokens) {
		return nil, 0, ErrMissingStation
	}
	pos++

	if pos < len(tokens) && (strings.EqualFold(tokens[pos], "AMD") || strings.EqualFold(tokens[pos], "COR")) {
		pos++
	}

	if pos >= len(tokens) {
		return nil, 0, ErrMissingStation
	}
	taf.StationID = tokens[pos]
	pos++

	if pos >= len(tokens) {
		return nil, 0, ErrMissingTime
	}
	issueTime, err := ParseTime(tokens[pos])
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse issue time: %w", err)
	}
	taf.IssueTimeVal = issueTime
	pos++

	if pos >= len(tokens) {
		return nil, 0, fmt.Errorf("missing validity period")
	}
	validFrom, validTo, err := ParseTAFValidity(tokens[pos], taf.IssueTimeVal)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse validity period: %w", err)
	}
	taf.ValidFrom = validFrom
	taf.ValidTo = validTo
	pos++

	return taf, pos, nil
}

func resolveDayTime(base time.Time, day, hour, min int) time.Time {
	year, month := base.Year(), base.Month()
	if day < base.Day() {
		nextMonth := time.Date(year, month+1, 1, 0, 0, 0, 0, time.UTC)
		year, month = nextMonth.Year(), nextMonth.Month()
	}
	return time.Date(year, month, day, hour, min, 0, 0, time.UTC)
}

func parseFMPeriod(tokens []string, startPos int, validityFrom, validityTo time.Time, warnings *[]string) (TafPeriod, int, error) {
	period := TafPeriod{Type: "FM"}

	match := tafFMPattern.FindStringSubmatch(tokens[startPos])
	if len(match) == 0 {
		return period, startPos + 1, fmt.Errorf("invalid FM format")
	}

	day, _ := strconv.Atoi(match[1])
	hour, _ := strconv.Atoi(match[2])
	min, _ := strconv.Atoi(match[3])

	period.ValidFrom = resolveDayTime(validityFrom, day, hour, min)
	period.ValidTo = validityTo

	endPos := findNextBoundary(tokens, startPos+1)
	parsePeriodElements(tokens[startPos+1:endPos], &period, warnings)

	if endPos < len(tokens) && strings.HasPrefix(strings.ToUpper(tokens[endPos]), "FM") {
		if nextMatch := tafFMPattern.FindStringSubmatch(tokens[endPos]); nextMatch != nil {
			nextDay, _ := strconv.Atoi(nextMatch[1])
			nextHour, _ := strconv.Atoi(nextMatch[2])
			nextMin, _ := strconv.Atoi(nextMatch[3])
			nextStart := resolveDayTime(period.ValidFrom, nextDay, nextHour, nextMin)
			if nextStart.Before(period.ValidTo) {
				period.ValidTo = nextStart
			}
		}
	}

	return period, endPos, nil
}

// parseRangePeriod handles both compact (e.g. TEMPO2512/2515) and spaced (e.g. TEMPO 2512/2515 or PROB30 2512/2515) periods.
func parseRangePeriod(
	periodType string,
	compactPattern *regexp.Regexp,
	tokens []string,
	startPos int,
	validityFrom time.Time,
	warnings *[]string,
) (TafPeriod, int, error) {
	period := TafPeriod{Type: periodType}

	var match []string
	elemStart := startPos + 1

	if compactPattern != nil {
		if m := compactPattern.FindStringSubmatch(tokens[startPos]); len(m) > 0 {
			match = m
		}
	}
	if len(match) == 0 && startPos+1 < len(tokens) {
		upper := strings.ToUpper(tokens[startPos])
		if upper == periodType || (periodType == "PROB" && strings.HasPrefix(upper, "PROB")) {
			if m := tafValidityPattern.FindStringSubmatch(tokens[startPos+1]); len(m) > 0 {
				match = m
				elemStart = startPos + 2
			}
		}
	}

	if len(match) == 0 {
		return period, startPos + 1, fmt.Errorf("invalid %s format", periodType)
	}

	fromDay, _ := strconv.Atoi(match[1])
	fromHour, _ := strconv.Atoi(match[2])
	toDay, _ := strconv.Atoi(match[3])
	toHour, _ := strconv.Atoi(match[4])

	period.ValidFrom = resolveDayTime(validityFrom, fromDay, fromHour, 0)
	period.ValidTo = resolveDayTime(period.ValidFrom, toDay, toHour, 0)

	endPos := findNextBoundary(tokens, elemStart)
	parsePeriodElements(tokens[elemStart:endPos], &period, warnings)

	return period, endPos, nil
}

func parsePeriodElements(tokens []string, period *TafPeriod, warnings *[]string) {
	pos, wind, visibility := parseWindAndVisibility(tokens)
	period.Wind = wind
	period.Visibility = visibility

	phenomConsumed, phenomena := parsePhenomena(tokens[pos:])
	period.Phenomena = append(period.Phenomena, phenomena...)
	pos += phenomConsumed

	cloudConsumed, clouds := parseClouds(tokens[pos:])
	period.Clouds = append(period.Clouds, clouds...)
	pos += cloudConsumed

	appendPeriodWarnings(warnings, tokens[pos:])
}
