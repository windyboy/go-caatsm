package aviation

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Parsing status constants returned inside Result.
const (
	StatusParsed      = "parsed"
	StatusHeaderError = "header_error"
	StatusBodyError   = "body_error"
)

var (
	otherPatterns = []*regexp.Regexp{
		navPattern,
		remarkPattern,
		selPattern,
		pbnPattern,
		eetPattern,
		performancePattern,
		regPattern,
		reroutePattern,
	}

	// ErrHeaderParse indicates an invalid header section.
	ErrHeaderParse = errors.New("invalid telegram header")
	// ErrBodyParse indicates a failure matching or validating the telegram body.
	ErrBodyParse = errors.New("invalid telegram body")
)

// PatternConfig binds a regex expression to a description for a category.
type PatternConfig struct {
	Pattern    string
	Comments   string
	Expression *regexp.Regexp
}

// CategoryParser parses a single aviation telegram category.
type CategoryParser interface {
	Category() string
	Patterns() []PatternConfig
	Parse(data map[string]string) (Body, error)
}

// Header contains metadata extracted from the AFTN envelope of an aviation telegram.
type Header struct {
	MessageID          string
	DateTime           string
	PriorityIndicator  string
	PrimaryAddress     string
	SecondaryAddresses string
	Originator         string
	OriginatorDateTime string
	Category           string
	Content            string
	Body               string
	ReceivedAt         time.Time
	ParsedAt           time.Time
}

// Result contains the parsed aviation header, body payload, and audit status.
// Parse always returns a non-nil *Result even on failure for audit persistence.
type Result struct {
	Header      Header
	BodyData    Body
	Parsed      bool
	Status      string
	Comments    string
	ErrorReason string
}

// BodyParser matches and parses an aviation telegram body string.
type BodyParser struct {
	body         string
	bodyPatterns map[string][]PatternConfig
}

// NewBodyParser creates a BodyParser for the given body text.
func NewBodyParser(body string) *BodyParser {
	return &BodyParser{
		body:         body,
		bodyPatterns: bodyPatterns,
	}
}

// Parse identifies the category, extracts capture groups, builds the typed Body,
// and calls Body.Validate().
func (p *BodyParser) Parse() (string, Body, error) {
	p.body = strings.TrimSpace(p.body)
	category := findCategory(p.body)
	if category == "" {
		return "", nil, fmt.Errorf("no category found in body text")
	}

	patterns, exists := p.bodyPatterns[category]
	if !exists || len(patterns) == 0 {
		return "", nil, fmt.Errorf("no matching pattern found for category: %s", category)
	}

	for _, pat := range patterns {
		if data := extract(p.body, pat.Expression); data != nil {
			parsed, err := parseCategory(category, data)
			if err != nil {
				return "", nil, err
			}
			if err := parsed.Validate(); err != nil {
				return "", nil, fmt.Errorf("validation failed for %s: %w", category, err)
			}
			return category, parsed, nil
		}
	}

	return "", nil, fmt.Errorf("no matching pattern found for body: %s", p.body)
}

// Parse parses a raw ICAO aviation telegram and ALWAYS returns a non-nil *Result,
// even when an error occurs, so callers can persist failed attempts for audit.
func Parse(rawText string) (*Result, error) {
	now := time.Now()
	if err := ValidateInputSize(rawText); err != nil {
		return &Result{
			Header: Header{
				Content:    rawText,
				ReceivedAt: now,
				ParsedAt:   now,
			},
			Parsed:      false,
			Status:      StatusHeaderError,
			Comments:    err.Error(),
			ErrorReason: err.Error(),
		}, fmt.Errorf("%w: %w", ErrHeaderParse, err)
	}

	header, err := ParseHeader(rawText)
	if err != nil {
		return &Result{
			Header: Header{
				Content:    rawText,
				ReceivedAt: now,
				ParsedAt:   now,
			},
			Parsed:      false,
			Status:      StatusHeaderError,
			Comments:    err.Error(),
			ErrorReason: err.Error(),
		}, fmt.Errorf("%w: %w", ErrHeaderParse, err)
	}

	category, bodyData, bodyErr := NewBodyParser(header.Body).Parse()
	header.Category = category
	header.ParsedAt = time.Now()

	if bodyErr != nil {
		return &Result{
			Header:      header,
			Parsed:      false,
			Status:      StatusBodyError,
			Comments:    bodyErr.Error(),
			ErrorReason: bodyErr.Error(),
		}, fmt.Errorf("%w: %w", ErrBodyParse, bodyErr)
	}

	return &Result{
		Header:   header,
		BodyData: bodyData,
		Parsed:   true,
		Status:   StatusParsed,
	}, nil
}

// ParseHeader parses the AFTN header portion of an ICAO telegram.
func ParseHeader(fullMessage string) (Header, error) {
	cleaned := cleanMessage(fullMessage)
	lines := strings.Split(cleaned, "\n")

	if len(lines) < MinHeaderLines {
		return Header{Content: fullMessage}, fmt.Errorf("invalid message format: expected at least %d lines, got %d", MinHeaderLines, len(lines))
	}

	_, messageID, dateTime, err := parseStartIndicator(lines[0])
	if err != nil {
		return Header{Content: fullMessage}, err
	}

	priorityIndicator, primaryAddress, extraAddrs := parsePriorityAndPrimary(lines[1])
	secondaryAddresses, originatorCode, originatorDateTime, body := parseRemainingLines(lines[2:], extraAddrs)

	return Header{
		MessageID:          messageID,
		DateTime:           dateTime,
		PriorityIndicator:  priorityIndicator,
		PrimaryAddress:     primaryAddress,
		SecondaryAddresses: secondaryAddresses,
		Originator:         originatorCode,
		OriginatorDateTime: originatorDateTime,
		Content:            fullMessage,
		Body:               body,
		ReceivedAt:         time.Now(),
	}, nil
}

func findCategory(body string) string {
	if match := categoryRegex.FindStringSubmatch(body); match != nil {
		for i, name := range categoryRegex.SubexpNames() {
			if i != 0 && name == Category {
				return match[i]
			}
		}
	}
	return ""
}

func extract(data string, exp *regexp.Regexp) map[string]string {
	match := exp.FindStringSubmatch(data)
	if len(match) == 0 {
		return nil
	}
	res := make(map[string]string)
	for i, name := range exp.SubexpNames() {
		if i != 0 && name != "" {
			res[name] = strings.TrimSpace(match[i])
		}
	}
	return res
}

func cleanMessage(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	lines := strings.Split(text, "\n")
	cleaned := make([]string, 0, len(lines))
	foundStart := false
	foundEnd := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !foundStart {
			idx := strings.Index(trimmed, StartIndicatorPrefix)
			if idx < 0 {
				continue
			}
			foundStart = true
			trimmed = trimmed[idx:]
		}
		if trimmed == "NNNN" {
			foundEnd = true
			break
		}
		if strings.HasSuffix(trimmed, "NNNN") {
			before := strings.TrimSpace(trimmed[:len(trimmed)-len("NNNN")])
			if strings.HasSuffix(before, ")") {
				cleaned = append(cleaned, before)
				foundEnd = true
				break
			}
		}
		cleaned = append(cleaned, trimmed)
	}

	if !foundStart || !foundEnd || len(cleaned) == 0 {
		return ""
	}
	return strings.Join(cleaned, "\n")
}

func parseStartIndicator(line string) (string, string, string, error) {
	parts := strings.Fields(line)
	if len(parts) >= MinStartIndicatorParts && strings.HasPrefix(parts[0], StartIndicatorPrefix) {
		return parts[0], parts[1], parts[2], nil
	}
	return "", "", "", fmt.Errorf("invalid start indicator line format: %s", line)
}

func parsePriorityAndPrimary(line string) (string, string, []string) {
	parts := strings.Fields(line)
	if len(parts) >= MinPriorityLineParts {
		var extras []string
		if len(parts) > 2 {
			extras = parts[2:]
		}
		return parts[0], parts[1], extras
	}
	return "", "", nil
}

func isOriginatorLine(line string) (originatorCode, dateTime string, isMatch bool) {
	if !strings.HasPrefix(line, ".") {
		return "", "", false
	}
	parts := strings.Fields(line[1:])
	if len(parts) < MinOriginatorParts {
		return "", "", false
	}
	if isAllUppercaseLetters(parts[0]) && isAllDigits(parts[1]) {
		return parts[0], parts[1], true
	}
	return "", "", false
}

func isBodyStartLine(line string) bool {
	return strings.HasPrefix(line, BeginPartMarker) || strings.HasPrefix(line, "(")
}

func parseRemainingLines(lines []string, initialSecondary []string) (string, string, string, string) {
	var (
		secondaryList      []string
		originatorCode     string
		originatorDateTime string
		body               strings.Builder
		inBody             bool
	)

	if len(initialSecondary) > 0 {
		secondaryList = append(secondaryList, strings.Join(initialSecondary, " "))
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == EndHeaderMarker {
			continue
		}
		if inBody {
			body.WriteString(line + "\n")
			continue
		}
		if orig, dt, isOrig := isOriginatorLine(line); isOrig {
			originatorCode = orig
			originatorDateTime = dt
			inBody = true
			continue
		}
		if isBodyStartLine(line) {
			inBody = true
			if !strings.Contains(line, "NNNN") {
				body.WriteString(line + "\n")
			}
			continue
		}
		if dt, orig := getOriginator(line); orig != "" {
			originatorDateTime = dt
			originatorCode = orig
			continue
		}
		secondaryList = append(secondaryList, line)
	}

	return strings.Join(secondaryList, " "), originatorCode, originatorDateTime, body.String()
}

func getOriginator(line string) (string, string) {
	match := originator.FindStringSubmatch(line)
	if len(match) >= MinOriginatorMatchGroups {
		return match[1], match[2]
	}
	return "", ""
}

func isAllUppercaseLetters(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}

func isAllDigits(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
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

// --- Category Parsers & Registry ---

type arrParser struct{}

func (arrParser) Category() string { return CategoryArrival }

func (arrParser) Patterns() []PatternConfig {
	return []PatternConfig{{
		Pattern:    ArrPatternString,
		Comments:   "Pattern for ARR message",
		Expression: ArrPatternExpression,
	}}
}

func (arrParser) Parse(data map[string]string) (Body, error) {
	category, err := GetRequiredField(data, Category)
	if err != nil {
		return nil, err
	}
	aircraftID, err := GetRequiredField(data, FlightNumber)
	if err != nil {
		return nil, err
	}
	depAirport, err := GetRequiredField(data, DepartureCode)
	if err != nil {
		return nil, err
	}
	arrAirport, err := GetRequiredField(data, ArrivalCode)
	if err != nil {
		return nil, err
	}
	arrTime, err := GetRequiredField(data, ArrivalTime)
	if err != nil {
		return nil, err
	}

	return &ARR{
		Category:         category,
		AircraftID:       aircraftID,
		SSRModeAndCode:   GetOptionalField(data, SSR),
		DepartureAirport: depAirport,
		ArrivalAirport:   arrAirport,
		ArrivalTime:      arrTime,
	}, nil
}

type depParser struct{}

func (depParser) Category() string { return CategoryDeparture }

func (depParser) Patterns() []PatternConfig {
	return []PatternConfig{{
		Pattern:    DepPatternString,
		Comments:   "Pattern for DEP message",
		Expression: DepPatternExpression,
	}}
}

func (depParser) Parse(data map[string]string) (Body, error) {
	category, err := GetRequiredField(data, Category)
	if err != nil {
		return nil, err
	}
	aircraftID, err := GetRequiredField(data, FlightNumber)
	if err != nil {
		return nil, err
	}
	depAirport, err := GetRequiredField(data, DepartureCode)
	if err != nil {
		return nil, err
	}
	depTime, err := GetRequiredField(data, DepartureTime)
	if err != nil {
		return nil, err
	}
	destination, err := GetRequiredField(data, ArrivalCode)
	if err != nil {
		return nil, err
	}

	return &DEP{
		Category:         category,
		AircraftID:       aircraftID,
		SSRModeAndCode:   GetOptionalField(data, SSR),
		DepartureAirport: depAirport,
		DepartureTime:    depTime,
		Destination:      destination,
	}, nil
}

type cnlParser struct{}

func (cnlParser) Category() string { return CategoryCancellation }

func (cnlParser) Patterns() []PatternConfig {
	return []PatternConfig{{
		Pattern:    CnlPatternString,
		Comments:   "Pattern for CNL message",
		Expression: CnlPatternExpression,
	}}
}

func (cnlParser) Parse(data map[string]string) (Body, error) {
	category, err := GetRequiredField(data, Category)
	if err != nil {
		return nil, err
	}
	aircraftID, err := GetRequiredField(data, FlightNumber)
	if err != nil {
		return nil, err
	}
	depAirport, err := GetRequiredField(data, DepartureCode)
	if err != nil {
		return nil, err
	}
	destAirport, err := GetRequiredField(data, ArrivalCode)
	if err != nil {
		return nil, err
	}

	return &CNL{
		Category:           category,
		AircraftID:         aircraftID,
		DepartureAirport:   depAirport,
		DestinationAirport: destAirport,
	}, nil
}

type dlaParser struct{}

func (dlaParser) Category() string { return CategoryDelay }

func (dlaParser) Patterns() []PatternConfig {
	return []PatternConfig{{
		Pattern:    DlaPatternString,
		Comments:   "Pattern for DLA message",
		Expression: DlaPatternExpression,
	}}
}

func (dlaParser) Parse(data map[string]string) (Body, error) {
	category, err := GetRequiredField(data, Category)
	if err != nil {
		return nil, err
	}
	aircraftID, err := GetRequiredField(data, FlightNumber)
	if err != nil {
		return nil, err
	}
	depAirport, err := GetRequiredField(data, DepartureCode)
	if err != nil {
		return nil, err
	}
	arrAirport, err := GetRequiredField(data, ArrivalCode)
	if err != nil {
		return nil, err
	}

	return &DLA{
		Category:         category,
		AircraftID:       aircraftID,
		DepartureAirport: depAirport,
		NewDepartureTime: GetOptionalField(data, DepartureTime),
		ArrivalAirport:   arrAirport,
		ArrivalTime:      GetOptionalField(data, ArrivalTime),
	}, nil
}

type fplParser struct{}

func (fplParser) Category() string { return CategoryFlightPlan }

func (fplParser) Patterns() []PatternConfig {
	return []PatternConfig{{
		Pattern:    FplPatternString,
		Comments:   "Pattern for FPL message",
		Expression: FplPatternExpression,
	}}
}

func (fplParser) Parse(data map[string]string) (Body, error) {
	category, err := GetRequiredField(data, Category)
	if err != nil {
		return nil, err
	}
	flightNumber, err := GetRequiredField(data, FlightNumber)
	if err != nil {
		return nil, err
	}
	aircraftID, err := GetRequiredField(data, AircraftID)
	if err != nil {
		return nil, err
	}
	indicator, err := GetRequiredField(data, Indicator)
	if err != nil {
		return nil, err
	}
	speed, err := GetRequiredField(data, Speed)
	if err != nil {
		return nil, err
	}
	level, err := GetRequiredField(data, Level)
	if err != nil {
		return nil, err
	}
	depAirport, err := GetRequiredField(data, DepartureCode)
	if err != nil {
		return nil, err
	}
	depTime, err := GetRequiredField(data, DepartureTime)
	if err != nil {
		return nil, err
	}
	route, err := GetRequiredField(data, Route)
	if err != nil {
		return nil, err
	}
	destCode, err := GetRequiredField(data, DestinationCode)
	if err != nil {
		return nil, err
	}
	estTime, err := GetRequiredField(data, EstimatedTime)
	if err != nil {
		return nil, err
	}

	otherInfo := GetOptionalField(data, OtherInfo)
	otherData := parseOther(otherInfo)

	return &FPL{
		Category:                category,
		FlightNumber:            flightNumber,
		ReferenceData:           GetOptionalField(data, ReferenceData),
		AircraftID:              aircraftID,
		SSRModeAndCode:          GetOptionalField(data, Surveillance),
		FlightRulesAndType:      indicator,
		CruisingSpeedAndLevel:   speed + level,
		DepartureAirport:        depAirport,
		DepartureTime:           depTime,
		Route:                   route,
		DestinationAndTotalTime: destCode + estTime,
		AlternateAirport:        GetOptionalField(data, AlternateAirport),
		OtherInfo:               otherInfo,
		Register:                otherData[Register],
		TotalEET:                estTime,
		PBN:                     otherData[PBN],
		NavigationEquipment:     otherData[NavigationEquipment],
		EstimatedElapsedTime:    otherData[EstimatedElapsedTime],
		SELCALCode:              otherData[SELCALCode],
		PerformanceCategory:     otherData[PerformanceCategory],
		RerouteInformation:      otherData[RerouteInformation],
		Remarks:                 otherData[Remarks],
	}, nil
}

var categoryRegistry = map[string]CategoryParser{
	CategoryArrival:      arrParser{},
	CategoryDeparture:    depParser{},
	CategoryCancellation: cnlParser{},
	CategoryDelay:        dlaParser{},
	CategoryFlightPlan:   fplParser{},
}

var bodyPatterns = buildBodyPatterns()

func buildBodyPatterns() map[string][]PatternConfig {
	patterns := make(map[string][]PatternConfig, len(categoryRegistry))
	for category, p := range categoryRegistry {
		patterns[category] = p.Patterns()
	}
	return patterns
}

func parseCategory(category string, data map[string]string) (Body, error) {
	p, ok := categoryRegistry[category]
	if !ok {
		return nil, fmt.Errorf("invalid message type: %s", category)
	}
	return p.Parse(data)
}
