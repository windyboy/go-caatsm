package weather

import (
	"caatsm/internal/domain/weather"
	"fmt"
	"strconv"
)

func parseWindAndVisibility(tokens []string) (int, *weather.Wind, *weather.Visibility) {
	pos := 0
	var wind *weather.Wind
	var visibility *weather.Visibility

	if pos < len(tokens) {
		if parsed := parseWind(tokens[pos]); parsed != nil {
			wind = parsed
			pos++

			if pos < len(tokens) {
				if match := variableWindPattern.FindStringSubmatch(tokens[pos]); match != nil {
					from, _ := strconv.Atoi(match[1])
					to, _ := strconv.Atoi(match[2])
					wind.Variable = true
					wind.VariableFrom = from
					wind.VariableTo = to
					pos++
				}
			}
		}
	}

	if pos < len(tokens) {
		if parsed := parseVisibility(tokens[pos]); parsed != nil {
			visibility = parsed
			pos++
		}
	}

	return pos, wind, visibility
}

func parsePhenomena(tokens []string) (int, []weather.Phenomenon) {
	pos := 0
	var phenomena []weather.Phenomenon

	for pos < len(tokens) {
		if match := phenomenonPattern.FindStringSubmatch(tokens[pos]); match != nil {
			phenomena = append(phenomena, weather.Phenomenon{
				Intensity:  match[1],
				Descriptor: match[2],
				Weather:    match[3],
			})
			pos++
			continue
		}
		break
	}

	return pos, phenomena
}

func parseClouds(tokens []string) (int, []weather.Cloud) {
	pos := 0
	var clouds []weather.Cloud

	if pos < len(tokens) {
		if skyClearPattern.MatchString(tokens[pos]) {
			return 1, clouds
		}
	}

	for pos < len(tokens) {
		if match := cloudPattern.FindStringSubmatch(tokens[pos]); match != nil {
			alt, _ := strconv.Atoi(match[2])
			clouds = append(clouds, weather.Cloud{
				Type:     match[1],
				Altitude: alt * 100,
				Modifier: match[3],
			})
			pos++
			continue
		}
		break
	}

	return pos, clouds
}

func appendPeriodWarnings(warnings *[]string, tokens []string) {
	if warnings == nil {
		return
	}
	for _, token := range tokens {
		*warnings = append(*warnings, fmt.Sprintf("unrecognized token in period: %s", token))
	}
}
