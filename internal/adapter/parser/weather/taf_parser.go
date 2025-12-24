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

	taf := &weather.Taf{
		ReportType: weather.ReportTypeTAF,
		RawTextVal: raw,
		Warnings:   []string{},
		Periods:    []weather.TafPeriod{},
	}

	pos := 0

	// Skip TAF - first token
	if pos >= len(tokens) {
		return nil, weather.ErrMissingStation
	}
	pos++

	// Parse station (second token)
	if pos >= len(tokens) {
		return nil, weather.ErrMissingStation
	}
	taf.StationID = tokens[pos]
	pos++

	// Parse issue time (DDHHmmZ) - third token
	if pos >= len(tokens) {
		return nil, weather.ErrMissingTime
	}
	issueTime, err := ParseTime(tokens[pos])
	if err != nil {
		return nil, fmt.Errorf("failed to parse issue time: %w", err)
	}
	taf.IssueTimeVal = issueTime
	pos++

	// Parse validity period (DDHH/DDHH)
	if pos >= len(tokens) {
		return nil, fmt.Errorf("missing validity period")
	}
	validFrom, validTo, err := ParseTAFValidity(tokens[pos], taf.IssueTimeVal)
	if err != nil {
		return nil, fmt.Errorf("failed to parse validity period: %w", err)
	}
	taf.ValidFrom = validFrom
	taf.ValidTo = validTo
	pos++

	// Parse main forecast period (before any FM/TEMPO/BECMG)
	mainPeriod := weather.TafPeriod{
		Type:      "MAIN",
		ValidFrom: validFrom,
		ValidTo:   validTo,
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
		mainTokens := tokens[pos:firstSpecialIdx]
		parsePeriodElements(mainTokens, &mainPeriod, taf)
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
			period, newPos, err := parseFMPeriod(tokens, pos, taf.IssueTimeVal)
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
			period, newPos, err := parseTEMPOPeriod(tokens, pos, taf.IssueTimeVal)
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
			period, newPos, err := parseBECMGPeriod(tokens, pos, taf.IssueTimeVal)
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

// parseFMPeriod parses an FM (from) period
func parseFMPeriod(tokens []string, startPos int, issueTimeVal time.Time) (weather.TafPeriod, int, error) {
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

	year := issueTimeVal.Year()
	month := issueTimeVal.Month()
	period.ValidFrom = time.Date(year, month, day, hour, min, 0, 0, time.UTC)

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
	parsePeriodElements(periodTokens, &period, nil)

	// Set valid_to to start of next period or end of validity
	if endPos < len(tokens) {
		upperToken := strings.ToUpper(tokens[endPos])
		if strings.HasPrefix(upperToken, "FM") {
			// Next period starts here
			if match := tafFMPattern.FindStringSubmatch(tokens[endPos]); match != nil {
				nextDay, _ := strconv.Atoi(match[1])
				nextHour, _ := strconv.Atoi(match[2])
				nextMin, _ := strconv.Atoi(match[3])
				period.ValidTo = time.Date(year, month, nextDay, nextHour, nextMin, 0, 0, time.UTC)
			}
		}
	}

	return period, endPos, nil
}

// parseTEMPOPeriod parses a TEMPO (temporary) period
func parseTEMPOPeriod(tokens []string, startPos int, issueTimeVal time.Time) (weather.TafPeriod, int, error) {
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

	year := issueTimeVal.Year()
	month := issueTimeVal.Month()

	period.ValidFrom = time.Date(year, month, fromDay, fromHour, 0, 0, 0, time.UTC)
	period.ValidTo = time.Date(year, month, toDay, toHour, 0, 0, 0, time.UTC)

	// If toDay < fromDay, assume next month
	if toDay < fromDay {
		period.ValidTo = period.ValidTo.AddDate(0, 1, 0)
	}

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
	parsePeriodElements(periodTokens, &period, nil)

	return period, endPos, nil
}

// parseBECMGPeriod parses a BECMG (becoming) period
func parseBECMGPeriod(tokens []string, startPos int, issueTimeVal time.Time) (weather.TafPeriod, int, error) {
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

	year := issueTimeVal.Year()
	month := issueTimeVal.Month()

	period.ValidFrom = time.Date(year, month, fromDay, fromHour, 0, 0, 0, time.UTC)
	period.ValidTo = time.Date(year, month, toDay, toHour, 0, 0, 0, time.UTC)

	// If toDay < fromDay, assume next month
	if toDay < fromDay {
		period.ValidTo = period.ValidTo.AddDate(0, 1, 0)
	}

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
	parsePeriodElements(periodTokens, &period, nil)

	return period, endPos, nil
}

// parsePeriodElements parses common elements (wind, visibility, clouds, phenomena) for a TAF period
func parsePeriodElements(tokens []string, period *weather.TafPeriod, taf *weather.Taf) {
	pos := 0

	// Parse wind
	if pos < len(tokens) {
		if wind := parseWind(tokens[pos]); wind != nil {
			period.Wind = wind
			pos++

			// Check for variable wind
			if pos < len(tokens) {
				if match := variableWindPattern.FindStringSubmatch(tokens[pos]); match != nil {
					from, _ := strconv.Atoi(match[1])
					to, _ := strconv.Atoi(match[2])
					period.Wind.Variable = true
					period.Wind.VariableFrom = from
					period.Wind.VariableTo = to
					pos++
				}
			}
		}
	}

	// Parse visibility
	if pos < len(tokens) {
		if vis := parseVisibility(tokens[pos]); vis != nil {
			period.Visibility = vis
			pos++
		}
	}

	// Parse weather phenomena
	for pos < len(tokens) {
		if match := phenomenonPattern.FindStringSubmatch(tokens[pos]); match != nil {
			phenom := weather.Phenomenon{
				Intensity:  match[1],
				Descriptor: match[2],
				Weather:    match[3],
			}
			period.Phenomena = append(period.Phenomena, phenom)
			pos++
		} else {
			break
		}
	}

	// Parse clouds
	for pos < len(tokens) {
		if match := skyClearPattern.FindStringSubmatch(tokens[pos]); match != nil {
			// SKC, CLR, NSC - no clouds
			pos++
			break
		}

		if match := cloudPattern.FindStringSubmatch(tokens[pos]); match != nil {
			alt, _ := strconv.Atoi(match[2])
			cloud := weather.Cloud{
				Type:     match[1],
				Altitude: alt * 100, // Convert to feet
				Modifier: match[3],
			}
			period.Clouds = append(period.Clouds, cloud)
			pos++
		} else {
			break
		}
	}

	// Collect any remaining unrecognized tokens as warnings
	if taf != nil {
		for pos < len(tokens) {
			taf.Warnings = append(taf.Warnings, fmt.Sprintf("unrecognized token in period: %s", tokens[pos]))
			pos++
		}
	}
}

