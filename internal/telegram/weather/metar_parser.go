package weather

import (
	"fmt"
	"strconv"
	"strings"
)

func parseMetar(raw string, reportType string) (*Metar, error) {
	tokens := Tokenize(raw)
	if len(tokens) < 3 {
		return nil, ErrInvalidFormat
	}

	metar, pos, err := parseMetarHeader(tokens, raw, reportType)
	if err != nil {
		return nil, err
	}

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
			metar.Temperature = &Temperature{Value: tempVal, Unit: "C"}
			metar.Dewpoint = &Temperature{Value: dewVal, Unit: "C"}
			pos++
		}
	}

	if pos < len(tokens) {
		if match := altimeterPattern.FindStringSubmatch(tokens[pos]); match != nil {
			value, _ := strconv.ParseFloat(match[2], 64)
			switch match[1] {
			case "Q":
				metar.Altimeter = &Altimeter{Value: value, Unit: "QNH"}
			case "A":
				metar.Altimeter = &Altimeter{Value: value / 100.0, Unit: "A"}
			}
			pos++
		}
	}

	for i := pos; i < len(tokens); i++ {
		if strings.HasPrefix(strings.ToUpper(tokens[i]), "RMK") {
			metar.Remarks = strings.Join(tokens[i:], " ")
			pos = len(tokens)
			break
		}
	}

	for pos < len(tokens) {
		metar.Warnings = append(metar.Warnings, fmt.Sprintf("unrecognized token: %s", tokens[pos]))
		pos++
	}

	return metar, nil
}

func parseMetarHeader(tokens []string, raw string, reportType string) (*Metar, int, error) {
	metar := &Metar{
		ReportType: ReportType(reportType),
		RawTextVal: raw,
		Warnings:   []string{},
		Clouds:     []Cloud{},
		Phenomena:  []Phenomenon{},
	}

	pos := 0
	if pos >= len(tokens) {
		return nil, 0, ErrMissingStation
	}
	pos++

	if pos >= len(tokens) {
		return nil, 0, ErrMissingStation
	}
	metar.StationID = tokens[pos]
	pos++

	if pos >= len(tokens) {
		return nil, 0, ErrMissingTime
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

func parseWind(token string) *Wind {
	match := windPattern.FindStringSubmatch(token)
	if len(match) == 0 {
		return nil
	}

	wind := &Wind{}
	if match[1] == "VRB" {
		wind.Variable = true
	} else {
		dir, _ := strconv.Atoi(match[1])
		wind.Direction = dir
	}

	speed, _ := strconv.Atoi(match[2])
	wind.Speed = speed

	if match[3] != "" {
		gust, _ := strconv.Atoi(match[4])
		wind.Gust = gust
	}

	wind.Unit = match[5]
	if wind.Unit == "" {
		wind.Unit = "KT"
	}
	return wind
}

func parseVisibility(token string) *Visibility {
	if match := directionalVisibilityPattern.FindStringSubmatch(token); match != nil {
		dist, _ := strconv.ParseFloat(match[1], 64)
		return &Visibility{
			Distance:  dist,
			Unit:      "M",
			Direction: match[2],
		}
	}

	match := visibilityPattern.FindStringSubmatch(token)
	if len(match) == 0 {
		return nil
	}

	vis := &Visibility{
		Modifier: match[1],
		Unit:     match[3],
	}

	distStr := match[2]
	if strings.Contains(distStr, "/") {
		dist, err := ParseFraction(distStr)
		if err != nil {
			return nil
		}
		vis.Distance = dist
	} else {
		dist, err := strconv.ParseFloat(distStr, 64)
		if err != nil {
			return nil
		}
		vis.Distance = dist
	}

	if vis.Unit == "" {
		if vis.Distance >= 10 {
			vis.Unit = "M"
		} else {
			vis.Unit = "SM"
		}
	}
	return vis
}
