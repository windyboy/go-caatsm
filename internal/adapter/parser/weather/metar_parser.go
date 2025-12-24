package weather

import (
	"caatsm/internal/domain/weather"
	"fmt"
	"strconv"
	"strings"
)

// parseMetar parses a METAR or SPECI report
func parseMetar(raw string, reportType string) (*weather.Metar, error) {
	tokens := Tokenize(raw)
	if len(tokens) < 3 {
		return nil, weather.ErrInvalidFormat
	}

	metar, pos, err := parseMetarHeader(tokens, raw, reportType)
	if err != nil {
		return nil, err
	}

	// Parse runway visual range (RVR) - skip for now, add to warnings
	for pos < len(tokens) && strings.HasPrefix(tokens[pos], "R") {
		metar.Warnings = append(metar.Warnings, fmt.Sprintf("RVR not parsed: %s", tokens[pos]))
		pos++
	}

	phenomConsumed, phenomena := parsePhenomena(tokens[pos:])
	metar.Phenomena = append(metar.Phenomena, phenomena...)
	pos += phenomConsumed

	cloudConsumed, clouds := parseClouds(tokens[pos:])
	metar.Clouds = append(metar.Clouds, clouds...)
	pos += cloudConsumed

	// Parse temperature/dewpoint
	if pos < len(tokens) {
		if match := tempPattern.FindStringSubmatch(tokens[pos]); match != nil {
			tempVal, _ := strconv.ParseFloat(match[2], 64)
			if match[1] == "M" {
				tempVal = -tempVal
			}

			dewVal, _ := strconv.ParseFloat(match[4], 64)
			if match[3] == "M" {
				dewVal = -dewVal
			}

			metar.Temperature = &weather.Temperature{
				Value: tempVal,
				Unit:  "C",
			}
			metar.Dewpoint = &weather.Temperature{
				Value: dewVal,
				Unit:  "C",
			}
			pos++
		}
	}

	// Parse altimeter
	if pos < len(tokens) {
		if match := altimeterPattern.FindStringSubmatch(tokens[pos]); match != nil {
			value, _ := strconv.ParseFloat(match[2], 64)
			unit := match[1]

			switch unit {
			case "Q":
				// QNH in hPa
				metar.Altimeter = &weather.Altimeter{
					Value: value,
					Unit:  "QNH",
				}
			case "A":
				// Altimeter in inHg
				metar.Altimeter = &weather.Altimeter{
					Value: value / 100.0, // A2992 means 29.92 inHg
					Unit:  "A",
				}
			}
			pos++
		}
	}

	// Parse remarks (everything after RMK)
	remarksStart := -1
	for i := pos; i < len(tokens); i++ {
		if strings.HasPrefix(strings.ToUpper(tokens[i]), "RMK") {
			remarksStart = i
			break
		}
	}

	if remarksStart >= 0 {
		metar.Remarks = strings.Join(tokens[remarksStart:], " ")
		pos = len(tokens) // Skip remaining tokens
	}

	// Collect any remaining unrecognized tokens as warnings
	for pos < len(tokens) {
		metar.Warnings = append(metar.Warnings, fmt.Sprintf("unrecognized token: %s", tokens[pos]))
		pos++
	}

	return metar, nil
}

func parseMetarHeader(tokens []string, raw string, reportType string) (*weather.Metar, int, error) {
	metar := &weather.Metar{
		ReportType: weather.ReportType(reportType),
		RawTextVal: raw,
		Warnings:   []string{},
		Clouds:     []weather.Cloud{},
		Phenomena:  []weather.Phenomenon{},
	}

	pos := 0
	if pos >= len(tokens) {
		return nil, 0, weather.ErrMissingStation
	}
	pos++

	if pos >= len(tokens) {
		return nil, 0, weather.ErrMissingStation
	}
	metar.StationID = tokens[pos]
	pos++

	if pos >= len(tokens) {
		return nil, 0, weather.ErrMissingTime
	}
	issueTime, err := ParseTime(tokens[pos])
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse issue time: %w", err)
	}
	metar.IssueTimeVal = issueTime
	metar.ObsTime = issueTime
	pos++

	if pos < len(tokens) {
		if match := modifierPattern.FindStringSubmatch(tokens[pos]); match != nil {
			metar.Modifier = match[1]
			pos++
		}
	}

	consumed, wind, visibility := parseWindAndVisibility(tokens[pos:])
	metar.Wind = wind
	metar.Visibility = visibility
	pos += consumed

	return metar, pos, nil
}

// parseWind parses wind information
func parseWind(token string) *weather.Wind {
	match := windPattern.FindStringSubmatch(token)
	if len(match) == 0 {
		return nil
	}

	wind := &weather.Wind{}

	// Parse direction
	if match[1] == "VRB" {
		wind.Variable = true
		wind.Direction = 0
	} else {
		dir, _ := strconv.Atoi(match[1])
		wind.Direction = dir
	}

	// Parse speed
	speed, _ := strconv.Atoi(match[2])
	wind.Speed = speed

	// Parse gust
	if match[3] != "" {
		gust, _ := strconv.Atoi(match[4])
		wind.Gust = gust
	}

	// Parse unit
	wind.Unit = match[5]
	if wind.Unit == "" {
		wind.Unit = "KT" // Default to knots
	}

	return wind
}

// parseVisibility parses visibility information
func parseVisibility(token string) *weather.Visibility {
	// Try directional visibility first
	if match := directionalVisibilityPattern.FindStringSubmatch(token); match != nil {
		dist, _ := strconv.ParseFloat(match[1], 64)
		return &weather.Visibility{
			Distance:  dist,
			Unit:      "M",
			Direction: match[2],
		}
	}

	// Try standard visibility pattern
	match := visibilityPattern.FindStringSubmatch(token)
	if len(match) == 0 {
		return nil
	}

	vis := &weather.Visibility{
		Modifier: match[1],
		Unit:     match[3],
	}

	// Parse distance
	distStr := match[2]
	if strings.Contains(distStr, "/") {
		// Fractional visibility (e.g., "1/4SM")
		dist, err := ParseFraction(distStr)
		if err == nil {
			vis.Distance = dist
		} else {
			return nil
		}
	} else {
		dist, err := strconv.ParseFloat(distStr, 64)
		if err != nil {
			return nil
		}
		vis.Distance = dist
	}

	// Default unit
	if vis.Unit == "" {
		if vis.Distance >= 10 {
			vis.Unit = "M" // Meters (e.g., 9999)
		} else {
			vis.Unit = "SM" // Statute miles
		}
	}

	return vis
}
