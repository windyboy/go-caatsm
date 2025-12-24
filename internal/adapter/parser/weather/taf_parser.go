package weather

import (
	"caatsm/internal/domain/weather"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// parseTaf parses a TAF report
func parseTaf(raw string) (*weather.Taf, error) {
	tokens := Tokenize(raw)
	if len(tokens) < 4 {
		return nil, weather.ErrInvalidFormat
	}

	taf, pos, err := parseTafHeader(tokens, raw)
	if err != nil {
		return nil, err
	}

	// Parse main forecast period (before any FM/TEMPO/BECMG)
	mainPeriod := weather.TafPeriod{
		Type:      "MAIN",
		ValidFrom: taf.ValidFrom,
		ValidTo:   taf.ValidTo,
	}

	// Find first special section
	firstSpecialIdx := len(tokens)
	for i := pos; i < len(tokens); i++ {
		upperToken := strings.ToUpper(tokens[i])
		if strings.HasPrefix(upperToken, "FM") ||
			strings.HasPrefix(upperToken, "TEMPO") ||
			strings.HasPrefix(upperToken, "BECMG") ||
			strings.HasPrefix(upperToken, "PROB") ||
			strings.HasPrefix(upperToken, "RMK") {
			firstSpecialIdx = i
			break
		}
	}

	// Parse main period tokens
	if firstSpecialIdx > pos {
		if firstSpecialIdx < len(tokens) {
			upperToken := strings.ToUpper(tokens[firstSpecialIdx])
			if strings.HasPrefix(upperToken, "FM") {
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
		}

		mainTokens := tokens[pos:firstSpecialIdx]
		parsePeriodElements(mainTokens, &mainPeriod, &taf.Warnings)
		taf.Periods = append(taf.Periods, mainPeriod)
		pos = firstSpecialIdx
	}

	// Parse special sections (FM, TEMPO, BECMG)
	var pendingProb int
	for pos < len(tokens) {
		upperToken := strings.ToUpper(tokens[pos])

		switch {
		case strings.HasPrefix(upperToken, "PROB"):
			// PROB30 or PROB40 - probability for next period
			probStr := strings.TrimPrefix(upperToken, "PROB")
			pendingProb, _ = strconv.Atoi(probStr)
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
			period, newPos, err := parseTEMPOPeriod(tokens, pos, taf.ValidFrom, &taf.Warnings)
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
			period, newPos, err := parseBECMGPeriod(tokens, pos, taf.ValidFrom, &taf.Warnings)
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
			// Remarks section - include RMK token and all following tokens
			taf.Remarks = strings.Join(tokens[pos:], " ")
			// Set pos to exit the loop
			pos = len(tokens)

		default:
			// Unrecognized token
			taf.Warnings = append(taf.Warnings, fmt.Sprintf("unrecognized token: %s", tokens[pos]))
			pos++
		}
	}

	return taf, nil
}

func parseTafHeader(tokens []string, raw string) (*weather.Taf, int, error) {
	taf := &weather.Taf{
		ReportType: weather.ReportTypeTAF,
		RawTextVal: raw,
		Warnings:   []string{},
		Periods:    []weather.TafPeriod{},
	}

	pos := 0
	if pos >= len(tokens) {
		return nil, 0, weather.ErrMissingStation
	}
	pos++

	if pos >= len(tokens) {
		return nil, 0, weather.ErrMissingStation
	}
	taf.StationID = tokens[pos]
	pos++

	if pos >= len(tokens) {
		return nil, 0, weather.ErrMissingTime
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
	t := time.Date(base.Year(), base.Month(), day, hour, min, 0, 0, time.UTC)
	if day < base.Day() {
		t = t.AddDate(0, 1, 0)
	}
	return t
}

// parseFMPeriod parses an FM (from) period
func parseFMPeriod(tokens []string, startPos int, validityFrom, validityTo time.Time, warnings *[]string) (weather.TafPeriod, int, error) {
	period := weather.TafPeriod{
		Type: "FM",
	}

	match := tafFMPattern.FindStringSubmatch(tokens[startPos])
	if len(match) == 0 {
		return period, startPos + 1, fmt.Errorf("invalid FM format")
	}

	day, _ := strconv.Atoi(match[1])
	hour, _ := strconv.Atoi(match[2])
	min, _ := strconv.Atoi(match[3])

	period.ValidFrom = resolveDayTime(validityFrom, day, hour, min)
	period.ValidTo = validityTo

	// Find end of this period (next FM, TEMPO, BECMG, or end)
	endPos := len(tokens)
	for i := startPos + 1; i < len(tokens); i++ {
		upperToken := strings.ToUpper(tokens[i])
		if strings.HasPrefix(upperToken, "FM") ||
			strings.HasPrefix(upperToken, "TEMPO") ||
			strings.HasPrefix(upperToken, "BECMG") ||
			strings.HasPrefix(upperToken, "RMK") {
			endPos = i
			break
		}
	}

	// Parse period elements
	periodTokens := tokens[startPos+1 : endPos]
	parsePeriodElements(periodTokens, &period, warnings)

	// Set valid_to to start of next period or end of validity
	if endPos < len(tokens) {
		upperToken := strings.ToUpper(tokens[endPos])
		if strings.HasPrefix(upperToken, "FM") {
			// Next period starts here
			if match := tafFMPattern.FindStringSubmatch(tokens[endPos]); match != nil {
				nextDay, _ := strconv.Atoi(match[1])
				nextHour, _ := strconv.Atoi(match[2])
				nextMin, _ := strconv.Atoi(match[3])
				nextStart := resolveDayTime(period.ValidFrom, nextDay, nextHour, nextMin)
				if nextStart.Before(period.ValidTo) {
					period.ValidTo = nextStart
				}
			}
		}
	}

	return period, endPos, nil
}

// parseTEMPOPeriod parses a TEMPO (temporary) period
func parseTEMPOPeriod(tokens []string, startPos int, validityFrom time.Time, warnings *[]string) (weather.TafPeriod, int, error) {
	period := weather.TafPeriod{
		Type: "TEMPO",
	}

	match := tafTEMPOPattern.FindStringSubmatch(tokens[startPos])
	if len(match) == 0 {
		return period, startPos + 1, fmt.Errorf("invalid TEMPO format")
	}

	fromDay, _ := strconv.Atoi(match[1])
	fromHour, _ := strconv.Atoi(match[2])
	toDay, _ := strconv.Atoi(match[3])
	toHour, _ := strconv.Atoi(match[4])

	period.ValidFrom = resolveDayTime(validityFrom, fromDay, fromHour, 0)
	period.ValidTo = resolveDayTime(period.ValidFrom, toDay, toHour, 0)

	// Find end of this period
	endPos := startPos + 1
	for endPos < len(tokens) {
		upperToken := strings.ToUpper(tokens[endPos])
		if strings.HasPrefix(upperToken, "FM") ||
			strings.HasPrefix(upperToken, "TEMPO") ||
			strings.HasPrefix(upperToken, "BECMG") ||
			strings.HasPrefix(upperToken, "RMK") {
			break
		}
		endPos++
	}

	// Parse period elements
	periodTokens := tokens[startPos+1 : endPos]
	parsePeriodElements(periodTokens, &period, warnings)

	return period, endPos, nil
}

// parseBECMGPeriod parses a BECMG (becoming) period
func parseBECMGPeriod(tokens []string, startPos int, validityFrom time.Time, warnings *[]string) (weather.TafPeriod, int, error) {
	period := weather.TafPeriod{
		Type: "BECMG",
	}

	match := tafBECMGPattern.FindStringSubmatch(tokens[startPos])
	if len(match) == 0 {
		return period, startPos + 1, fmt.Errorf("invalid BECMG format")
	}

	fromDay, _ := strconv.Atoi(match[1])
	fromHour, _ := strconv.Atoi(match[2])
	toDay, _ := strconv.Atoi(match[3])
	toHour, _ := strconv.Atoi(match[4])

	period.ValidFrom = resolveDayTime(validityFrom, fromDay, fromHour, 0)
	period.ValidTo = resolveDayTime(period.ValidFrom, toDay, toHour, 0)

	// Find end of this period
	endPos := startPos + 1
	for endPos < len(tokens) {
		upperToken := strings.ToUpper(tokens[endPos])
		if strings.HasPrefix(upperToken, "FM") ||
			strings.HasPrefix(upperToken, "TEMPO") ||
			strings.HasPrefix(upperToken, "BECMG") ||
			strings.HasPrefix(upperToken, "RMK") {
			break
		}
		endPos++
	}

	// Parse period elements
	periodTokens := tokens[startPos+1 : endPos]
	parsePeriodElements(periodTokens, &period, warnings)

	return period, endPos, nil
}

// parsePeriodElements parses common elements (wind, visibility, clouds, phenomena) for a TAF period
func parsePeriodElements(tokens []string, period *weather.TafPeriod, warnings *[]string) {
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
