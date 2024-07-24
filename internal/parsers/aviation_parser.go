package parsers

import (
	"caatsm/internal/config"
	"caatsm/internal/domain"
	"caatsm/pkg/utils"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	StartIndicatorPrefix = "ZCZC"
	EndHeaderMarker      = "."
	BeginPartMarker      = "BEGIN PART"
)

var (
	categoryRegex      = regexp.MustCompile(`\((?P<category>[A-Z]+)-`)
	emptyLineRemove    = regexp.MustCompile(`(?m)^\s*$`)
	bodyOnly           = regexp.MustCompile(`(.|\n)?(ZCZC(.|\n)*)NNNN(.|\n)?$`)
	originator         = regexp.MustCompile(`(?P<originatorDateTime>[0-9]+)\s(?P<originator>[A-Z]+)`)
	navPattern         = regexp.MustCompile(`(?m)NAV\/(?P<nav>.*)$`)
	remarkPattern      = regexp.MustCompile(`(?s)RMK\/(?P<remark>.*)`)
	selPattern         = regexp.MustCompile(`(?m)SEL\/(?P<sel>\w+)`)
	pbnPattern         = regexp.MustCompile(`(?m)PBN\/(?P<pbn>[A-Z0-9]+)`)
	eetPattern         = regexp.MustCompile(`(?s)(-?EET\/(?P<eet>(?:[A-Z]{4}\d{4}\s*)+))`)
	performancePattern = regexp.MustCompile(`(?s)-?PER\/(?P<per>\w)`)
	reroutePattern     = regexp.MustCompile(`(?m)RIF\/(?P<rif>.*)[A-Z]{3}\/`)
	otherPatterns      = []*regexp.Regexp{navPattern, remarkPattern, selPattern, pbnPattern, eetPattern, performancePattern, reroutePattern}
)

var mu sync.Mutex

type BodyParser struct {
	bodyMu       sync.Mutex
	body         string
	bodyPatterns map[string]config.BodyConfig
}

func NewBodyParser(body string) *BodyParser {
	return &BodyParser{bodyPatterns: config.GetBodyPatterns(), body: body}
}

func (bp *BodyParser) GetBodyPatterns() map[string]config.BodyConfig {
	return bp.bodyPatterns
}

func (bp *BodyParser) SetBodyPatterns(patterns map[string]config.BodyConfig) {
	bp.bodyPatterns = patterns
}

func (bp *BodyParser) Parse() (string, interface{}, error) {
	bp.bodyMu.Lock()
	defer bp.bodyMu.Unlock()
	bp.body = strings.TrimSpace(bp.body)
	category := findCategory(bp.body)
	if category == "" {
		return "", nil, fmt.Errorf("no category found in body text")
	}

	if patternConfig, exists := bp.bodyPatterns[category]; exists && patternConfig.Patterns != nil {
		for _, p := range patternConfig.Patterns {
			if match := p.Expression.FindStringSubmatch(bp.body); match != nil {
				data := extractData(match, p.Expression)
				return bp.createBodyData(data)
			}
		}
	}
	return "", nil, fmt.Errorf("no matching pattern found for body: %s", bp.body)
}

func findCategory(body string) string {
	if match := categoryRegex.FindStringSubmatch(body); match != nil {
		for i, name := range categoryRegex.SubexpNames() {
			if i != 0 && name == "category" {
				return match[i]
			}
		}
	}
	return ""
}

func extractData(match []string, re *regexp.Regexp) map[string]string {
	data := make(map[string]string)
	for i, name := range re.SubexpNames() {
		if i != 0 && name != "" {
			data[name] = strings.TrimSpace(match[i])
		}
	}
	return data
}

func (bp *BodyParser) createBodyData(data map[string]string) (string, interface{}, error) {
	switch category := data["category"]; category {
	case "ARR":
		return category, &domain.ARR{
			Category:         data["category"],
			AircraftID:       data["number"],
			SSRModeAndCode:   data["ssr"],
			DepartureAirport: data["departure"],
			ArrivalAirport:   data["arrival"],
			ArrivalTime:      data["time"],
		}, nil
	case "DEP":
		return category, &domain.DEP{
			Category:         data["category"],
			AircraftID:       data["number"],
			SSRModeAndCode:   data["ssr"],
			DepartureAirport: data["departure"],
			DepartureTime:    data["departure_time"],
			Destination:      data["arrival"],
		}, nil
	case "FPL":
		otherData := parseOther(data["other"])
		return category, &domain.FPL{
			Category:                data["category"],
			FlightNumber:            data["number"],
			ReferenceData:           data["reference_data"],
			AircraftID:              data["aircraft"],
			SSRModeAndCode:          data["surve"],
			FlightRulesAndType:      data["indicator"],
			CruisingSpeedAndLevel:   data["speed"] + data["level"],
			DepartureAirport:        data["departure"],
			DepartureTime:           data["departure_time"],
			Route:                   data["route"],
			DestinationAndTotalTime: data["destination"] + data["estt"],
			AlternateAirport:        data["alter"],
			OtherInfo:               data["other"],
			Register:                otherData["reg"],
			EstimatedArrivalTime:    data["estt"],
			PBN:                     otherData["pbn"],
			NavigationEquipment:     otherData["nav"],
			EstimatedElapsedTime:    otherData["eet"],
			SELCALCode:              otherData["sel"],
			PerformanceCategory:     otherData["per"],
			RerouteInformation:      otherData["rif"],
			Remarks:                 otherData["remark"],
		}, nil
	default:
		return category, nil, fmt.Errorf("cannot parse: %s", category)
	}
}

func Parse(rawText string) (*domain.ParsedMessage, error) {
	mu.Lock()
	defer mu.Unlock()
	message, err := ParseHeader(rawText)
	if err != nil {
		return nil, err
	}

	bodyParser := NewBodyParser(message.Body)
	category, bodyData, err := bodyParser.Parse()
	message.Category = category
	message.ParsedAt = time.Now()

	if err != nil {
		return &message, err
	}

	message.BodyData = bodyData
	return &message, nil
}

func cleanMessage(text string) string {
	cleanedText := emptyLineRemove.ReplaceAllString(text, "")
	cleanText := strings.ReplaceAll(cleanedText, "\n\n", "\n")
	if match := bodyOnly.FindStringSubmatch(cleanText); len(match) > 1 {
		bodyContent := match[2]
		if bodyContent[len(bodyContent)-1] == '\n' {
			return bodyContent[:len(bodyContent)-1]
		}
		return bodyContent
	}
	return ""
}

func ParseHeader(fullMessage string) (domain.ParsedMessage, error) {
	log := utils.GetSugaredLogger()
	cleaned := cleanMessage(fullMessage)
	lines := strings.Split(cleaned, "\n")

	if len(lines) < 3 {
		log.Warnf("invalid message format: %s", fullMessage)
		return domain.ParsedMessage{Text: fullMessage}, fmt.Errorf("invalid message format: %s", fullMessage)
	}

	_, messageID, dateTime, err := parseStartIndicator(lines[0])
	if err != nil {
		return domain.ParsedMessage{Text: fullMessage}, err
	}

	priorityIndicator, primaryAddress := parsePriorityAndPrimary(lines[1])
	secondaryAddresses, originator, originatorDateTime, body := parseRemainingLines(lines[2:])

	return domain.ParsedMessage{
		MessageID:          messageID,
		DateTime:           dateTime,
		PriorityIndicator:  priorityIndicator,
		PrimaryAddress:     primaryAddress,
		SecondaryAddresses: secondaryAddresses,
		Originator:         originator,
		OriginatorDateTime: originatorDateTime,
		Text:               fullMessage,
		Body:               body,
		ReceivedAt:         time.Now(),
	}, nil
}

func parseStartIndicator(line string) (string, string, string, error) {
	parts := strings.Fields(line)
	if len(parts) >= 3 && strings.HasPrefix(parts[0], StartIndicatorPrefix) {
		return parts[0], parts[1], parts[2], nil
	}
	utils.GetSugaredLogger().Warnf("invalid start indicator line format: %s", line)
	return "", "", "", fmt.Errorf("invalid start indicator line format: %s", line)
}

func parsePriorityAndPrimary(line string) (string, string) {
	parts := strings.Fields(line)
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}
	utils.GetSugaredLogger().Warnf("invalid priority and primary address line format: %s", line)
	return "", ""
}

func parseRemainingLines(lines []string) ([]string, string, string, string) {
	var (
		secondaryAddresses []string
		originator         string
		originatorDateTime string
		bodyAndFooter      strings.Builder
		headerEnded        bool
	)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if headerEnded {
			bodyAndFooter.WriteString(line + "\n")
		} else {
			switch {
			case line == EndHeaderMarker:
			case strings.HasPrefix(line, "."):
				originatorInfo := strings.Fields(line[1:])
				if len(originatorInfo) >= 2 {
					originator = originatorInfo[0]
					originatorDateTime = originatorInfo[1]
				}
				headerEnded = true
			case strings.HasPrefix(line, BeginPartMarker) || strings.HasPrefix(line, "("):
				headerEnded = true
				if strings.Index(line, "NNNN") > 0 {
					break
				}
				bodyAndFooter.WriteString(line + "\n")
			default:
				if o1, o2 := getOriginator(line); o1 != "" {
					originatorDateTime = o1
					originator = o2
				} else {
					secondaryAddresses = append(secondaryAddresses, line)
				}
			}
		}
	}

	return secondaryAddresses, originator, originatorDateTime, bodyAndFooter.String()
}

func getOriginator(line string) (string, string) {
	match := originator.FindStringSubmatch(line)
	if len(match) >= 3 {
		return match[1], match[2]
	}
	utils.GetSugaredLogger().Warnf("invalid originator line format: %s", line)
	return "", ""
}

func parseOther(text string) map[string]string {
	data := make(map[string]string)
	for _, re := range otherPatterns {
		if match := re.FindStringSubmatch(text); len(match) > 0 {
			for i, name := range re.SubexpNames() {
				if i != 0 && name != "" {
					data[name] = strings.TrimSpace(match[i])
				}
			}
		}
	}
	return data
}
