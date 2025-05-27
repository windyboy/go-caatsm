// Package parsers provides functionality for parsing various types of aviation messages,
// including standard ICAO/SITA messages and potentially other formats like flight schedules.
// It defines structures for parser configuration, regular expressions for pattern matching,
// and functions to extract meaningful data from raw message strings into domain objects.
package parsers

import (
	"caatsm/internal/domain"
	"caatsm/pkg/utils" // Assuming utils.FirstNChars exists as used in later comments
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Constants used as keys for extracting data from parsed messages,
// particularly from the body and 'OtherInfo' (Field 18) of Flight Plan (FPL) messages.
// These often correspond to standard ICAO field identifiers or common abbreviations.
const (
	// SSR is the key for Secondary Surveillance Radar mode and code.
	SSR = "ssr"
	// DepartureCode is the key for the departure airport code (ICAO).
	DepartureCode = "dep"
	// DepartureTime is the key for the departure time.
	DepartureTime = "dep_time"
	// ArrivalCode is the key for the arrival airport code (ICAO).
	ArrivalCode = "arr"
	// ArrivalTime is the key for the arrival time.
	ArrivalTime = "arr_time"
	// DestinationCode is the key for the destination airport code (ICAO).
	DestinationCode = "dest"
	// OtherInfo typically refers to ICAO Field 18 content in an FPL.
	OtherInfo = "other" // Field 18 in FPL

	// ReferenceData is the key for reference data (e.g., FPL Field 5).
	ReferenceData = "reference_data"
	// Aircraft is the key for aircraft type and wake turbulence category (e.g., FPL Field 9).
	Aircraft = "aircraft" // Type of aircraft and wake turbulence category
	// CategorySurveillance is the key for surveillance equipment category (e.g., FPL Field 10b - SSR equipment).
	CategorySurveillance = "surve" // Surveillance equipment
	// Indicator is the key for flight rules and type of flight (e.g., FPL Field 8).
	Indicator = "indicator" // Flight rules and type of flight
	// AircraftIDFPL is the key specifically for aircraft identification in an FPL context (Field 7),
	// distinguished from generic aircraft IDs in other message types.
	AircraftIDFPL = "aircraft_id_fpl" // Distinguish from generic AircraftID used in other message types
	// Surveillance is an alias for CategorySurveillance, often used for SSR equipment details from FPL Field 10b.
	Surveillance = "surve" // Equipment (SSR mode and code from field 10b)
	// Speed is the key for cruising speed (e.g., FPL Field 15).
	Speed = "speed" // Cruising speed
	// Level is the key for cruising level (e.g., FPL Field 15).
	Level = "level" // Cruising level
	// Route is the key for the flight route details (e.g., FPL Field 15).
	Route = "route" // Route of flight
	// EstimatedTime is the key for estimated time (departure/arrival/enroute, e.g., FPL Field 13 or 16).
	EstimatedTime = "estt" // Estimated time
	// AlternateAirport is the key for alternate airport(s) (e.g., FPL Field 16).
	AlternateAirport = "alter" // Alternate airport
	// PBN is the key for Performance-Based Navigation capabilities (from FPL Field 18 PBN/).
	PBN = "pbn"
	// NavigationEquipment is the key for navigation equipment (from FPL Field 18 NAV/).
	NavigationEquipment = "nav"
	// EstimatedElapsedTime is the key for estimated elapsed time to waypoints or FIR boundaries (from FPL Field 18 EET/).
	EstimatedElapsedTime = "eet" // Estimated Elapsed Time
	// SELCALCode is the key for SELCAL code (from FPL Field 18 SEL/).
	SELCALCode = "sel"
	// PerformanceCategory is the key for aircraft performance category (from FPL Field 18 PER/).
	PerformanceCategory = "per"
	// RerouteInformation is the key for reroute information (from FPL Field 18 RIF/).
	RerouteInformation = "rif"
	// Remarks is the key for general remarks (from FPL Field 18 RMK/).
	Remarks = "remark"
	// Register is the key for aircraft registration mark (from FPL Field 18 REG/).
	Register = "reg"
)

// otherPatterns is a list of compiled regular expressions used by the parseOther function
// to extract specific data from the 'OtherInfo' (Field 18) string of an FPL message.
var (
	otherPatterns = []*regexp.Regexp{
		navPattern,         // Parses NAV/ codes for navigation equipment
		remarkPattern,      // Parses RMK/ free-form remarks
		selPattern,         // Parses SEL/ SELCAL codes
		pbnPattern,         // Parses PBN/ Performance-Based Navigation capabilities
		eetPattern,         // Parses EET/ Estimated Elapsed Times to waypoints or FIR boundaries
		performancePattern, // Parses PER/ Aircraft performance data
		regPattern,         // Parses REG/ Aircraft registration marks
		reroutePattern,     // Parses RIF/ Reroute information
		// Note: Other patterns like STS/, DOF/, OPR/, ORGN/, DLE/ etc. could be added here if needed.
	}
)

// BodyParser is responsible for parsing the body of an aviation message.
// It uses a map of category-specific patterns (BodyConfig) to extract structured data.
type BodyParser struct {
	body         string                 // body is the raw string content of the message body to be parsed.
	bodyPatterns map[string]BodyConfig  // bodyPatterns maps message categories (e.g., "ARR") to their parsing configurations.
	mu           sync.Mutex             // mu provides thread-safe access to bodyPatterns if it can be modified concurrently.
}

// NewBodyParser creates and returns a new BodyParser initialized with the provided message body string.
// It sets the default bodyPatterns for parsing known message types.
// body: The raw string content of the message body to be parsed.
func NewBodyParser(body string) *BodyParser {
	return &BodyParser{
		bodyPatterns: bodyPatterns, // bodyPatterns is assumed to be a global or package-level variable
		body:         body,
	}
}

// GetBodyPatterns returns the current map of body patterns used by the parser.
// It is a thread-safe method.
func (parser *BodyParser) GetBodyPatterns() map[string]BodyConfig {
	parser.mu.Lock()
	defer parser.mu.Unlock()
	return parser.bodyPatterns
}

// SetBodyPatterns allows replacing the parser's body patterns map.
// This could be used for dynamic updates or testing with different pattern sets.
// It is a thread-safe method.
// patterns: A map where keys are message categories (e.g., "ARR", "FPL") and values are BodyConfig containing regex patterns.
func (parser *BodyParser) SetBodyPatterns(patterns map[string]BodyConfig) {
	parser.mu.Lock()
	defer parser.mu.Unlock()
	parser.bodyPatterns = patterns
}

// Parse attempts to parse the message body stored in the BodyParser.
// It first identifies the message category (e.g., ARR, FPL) using findCategory.
// Then, it iterates through the configured regular expressions for that category,
// attempting to extract data. If a match is found, it uses createBodyData
// to construct the appropriate domain-specific message struct (e.g., domain.ARR, domain.FPL).
// Returns:
// - The identified message category (string).
// - The parsed data as an interface{} (e.g., *domain.ARR).
// - An error if no category is found, no matching pattern is found, or if createBodyData fails.
// This method is thread-safe due to the lock at the beginning.
func (parser *BodyParser) Parse() (string, interface{}, error) {
	parser.mu.Lock()
	defer parser.mu.Unlock()

	parser.body = strings.TrimSpace(parser.body) // Clean the body string first
	category := findCategory(parser.body)
	if category == "" {
		return "", nil, fmt.Errorf("no category found in body text")
	}

	if patternConfig, exists := parser.bodyPatterns[category]; exists && patternConfig.Patterns != nil {
		for _, p := range patternConfig.Patterns {
			if data := extract(parser.body, p.Expression); data != nil {
				return parser.createBodyData(data)
			}
		}
	}
	return "", nil, fmt.Errorf("no matching pattern found for body: %s", parser.body)
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

// extract attempts to find a match for the given expression in the text and extracts named capture groups.
// text: The input string to search within.
// expression: The regular expression to use for matching.
// Returns a map of extracted data if a match is found, otherwise nil.
func extract(text string, expression *regexp.Regexp) map[string]string {
	match := expression.FindStringSubmatch(text)
	if len(match) > 0 {
		return extractData(match, expression)
	}
	return nil
}

// extractData extracts named capture groups from a regex match into a map.
// match: The result of a regex FindStringSubmatch operation.
// regex: The regular expression that was used (needed for SubexpNames).
// Returns a map where keys are capture group names and values are the captured strings.
func extractData(match []string, regex *regexp.Regexp) map[string]string {
	extractedData := make(map[string]string)
	for i, name := range regex.SubexpNames() {
		if i != 0 && name != "" { // Group 0 is the full match,Unnamed groups are ignored
			extractedData[name] = strings.TrimSpace(match[i])
		}
	}
	return extractedData
}

// createArrivalData creates a domain.ARR struct from parsed data.
// data: A map of strings containing parsed ARR message fields.
// Returns a pointer to a domain.ARR struct and an error if creation fails (though current impl returns nil error).
// This is an unexported helper for BodyParser.Parse.
func (parser *BodyParser) createArrivalData(data map[string]string) (*domain.ARR, error) {
	return &domain.ARR{
		Category:         data[Category],
		AircraftID:       data[FlightNumber], // FlightNumber is a parser constant, maps to "flight_number" named group
		SSRModeAndCode:   data[SSR],
		DepartureAirport: data[DepartureCode],
		DepartureTime:    data[DepartureTime], // Added based on ARR regex having dep_time
		ArrivalAirport:   data[ArrivalCode],
		ArrivalTime:      data[ArrivalTime],
		EstimatedElapsedTime: data[EstimatedTime], // Added based on ARR regex having estt
		AlternateAirport: data[AlternateAirport],  // Added based on ARR regex having alter
		OtherInfo:        data[OtherInfo],         // Added based on ARR regex having other
	}, nil
}

// createDepartureData creates a domain.DEP struct from parsed data.
// data: A map of strings containing parsed DEP message fields.
// Returns a pointer to a domain.DEP struct and an error if creation fails (current impl returns nil error).
// This is an unexported helper for BodyParser.Parse.
func (parser *BodyParser) createDepartureData(data map[string]string) (*domain.DEP, error) {
	return &domain.DEP{
		Category:         data[Category],
		AircraftID:       data[FlightNumber],
		SSRModeAndCode:   data[SSR],
		DepartureAirport: data[DepartureCode],
		DepartureTime:    data[DepartureTime],
		Destination:      data[ArrivalCode],       // For DEP, ArrivalCode from regex is used as Destination
		EstimatedElapsedTime: data[EstimatedTime], // For DEP, EstimatedTime from regex is used as EstimatedElapsedTime
		AlternateAirport: data[AlternateAirport],
		OtherInfo:        data[OtherInfo],
	}, nil
}

// createCancellationData creates a domain.CNL struct from parsed data.
// data: A map of strings containing parsed CNL message fields.
// Returns a pointer to a domain.CNL struct and an error if creation fails (current impl returns nil error).
// This is an unexported helper for BodyParser.Parse.
func (parser *BodyParser) createCancellationData(data map[string]string) (*domain.CNL, error) {
	return &domain.CNL{
		Category:           data["category"], // "category" key is lowercase in this specific regex's named group
		AircraftID:         data[FlightNumber],
		DepartureAirport:   data[DepartureCode],
		DestinationAirport: data[ArrivalCode], // For CNL, ArrivalCode from regex is used as DestinationAirport
		OtherInfo:          data[OtherInfo],
	}, nil
}

// createDelayData creates a domain.DLA struct from parsed data.
// data: A map of strings containing parsed DLA message fields.
// Returns a pointer to a domain.DLA struct and an error if creation fails (current impl returns nil error).
// This is an unexported helper for BodyParser.Parse.
func (parser *BodyParser) createDelayData(data map[string]string) (*domain.DLA, error) {
	return &domain.DLA{
		Category:             data[Category],
		AircraftID:           data[FlightNumber],
		SSRModeAndCode:       data[SSR],
		DepartureAirport:     data[DepartureCode],
		NewDepartureTime:     data[DepartureTime], // For DLA, DepartureTime from regex is NewDepartureTime
		ArrivalAirport:       data[ArrivalCode],
		EstimatedElapsedTime: data[EstimatedTime], // For DLA, EstimatedTime from regex is EstimatedElapsedTime
		OtherInfo:            data[OtherInfo],
	}, nil
}

// createFlightPlanData creates a domain.FPL struct from parsed data.
// data: A map of strings containing parsed FPL message fields.
// Returns a pointer to a domain.FPL struct and an error if creation fails (current impl returns nil error).
// This is an unexported helper for BodyParser.Parse.
func (parser *BodyParser) createFlightPlanData(data map[string]string) (*domain.FPL, error) {
	otherInfoMap := parseOther(data[OtherInfo]) // Parse sub-fields from Field 18
	return &domain.FPL{
		Category:                data[Category],
		FlightNumber:            data[FlightNumber],
		ReferenceData:           data[ReferenceData],       // Not typically in FPL body regex, might be from header or other source. Assuming it's a named group if present.
		AircraftID:              data[AircraftIDFPL],       // Specific key for FPL aircraft ID (type+wake)
		SSRModeAndCode:          data[Surveillance],        // SSR/Equipment from FPL specific field
		FlightRulesAndType:      data[Indicator],
		CruisingSpeedAndLevel:   data[Speed] + data[Level], // Combine speed and level
		DepartureAirport:        data[DepartureCode],
		DepartureTime:           data[DepartureTime],
		Route:                   data[Route],
		DestinationAndTotalTime: data[DestinationCode] + data[EstimatedTime], // Combine destination and EET part of Field 16
		AlternateAirport:        data[AlternateAirport],
		OtherInfo:               data[OtherInfo], // Raw Field 18 string

		// Fields parsed from OtherInfo
		Register:             otherInfoMap[Register],
		PBN:                  otherInfoMap[PBN],
		NavigationEquipment:  otherInfoMap[NavigationEquipment],
		EstimatedElapsedTime: otherInfoMap[EstimatedElapsedTime], // This is Field 18 EET/
		SELCALCode:           otherInfoMap[SELCALCode],
		PerformanceCategory:  otherInfoMap[PerformanceCategory],
		RerouteInformation:   otherInfoMap[RerouteInformation],
		Remarks:              otherInfoMap[Remarks],
		// Note: FPL domain struct has EstimatedArrivalTime, but FPL regex usually provides Total Estimated Elapsed Time in Field 16.
		// The parser's FPL regex has 'estt' for the time in Field 16 (DestinationAndTotalTime).
		// The domain.FPL.EstimatedArrivalTime might need to be derived or is from a different source.
		// For now, assuming it might be redundant if DestinationAndTotalTime's time part is the primary EET.
		// If it's specifically ETA, the parser needs to provide it.
		// The createFlightPlanData in parser.go does: `EstimatedArrivalTime:    data[EstimatedTime],` which is the Field 16 time.
		EstimatedArrivalTime: data[EstimatedTime], // This is the time part of Field 16 (e.g., 0153 from ZBAA0153)
	}, nil
}

// Parse is the main entry point for parsing a complete raw aviation message string.
// It first attempts to parse the header of the message using ParseHeader.
// If header parsing is successful, it then uses a BodyParser to parse the extracted message body.
// The resulting data (header, parsed body, category, etc.) is populated into a domain.ParsedMessage struct.
// rawText: The complete, raw aviation message as a string.
// Returns a pointer to a domain.ParsedMessage. If parsing fails at any critical stage,
// the ParsedMessage.Parsed field will be false, and ParsedMessage.Comments may contain error details.
// The function always returns a ParsedMessage pointer, even on errors, to capture partial data or the original content.
func Parse(rawText string) *domain.ParsedMessage {
	message, err := ParseHeader(rawText) // message here is a domain.ParsedMessage
	if err != nil {
		// ParseHeader populates pm.Content and potentially pm.Comments.
		// Ensure Uuid is set for tracking, even on header parse failure.
		if message.Uuid == "" {
			message.Uuid = uuid.New().String()
		}
		message.Parsed = false // Explicitly set Parsed to false
		return &message
	}

	// If header parsing was okay, proceed to parse the body.
	// message already contains header data and the extracted body string.
	bodyParser := NewBodyParser(message.Body)
	category, bodyData, bodyParserErr := bodyParser.Parse()
	message.Category = category // Set category even if body parsing fails, as category is found first
	message.ParsedAt = time.Now()

	if bodyParserErr != nil {
		message.Comments = bodyParserErr.Error()
		if message.Uuid == "" { // Ensure UUID is set
			message.Uuid = uuid.New().String()
		}
		message.Parsed = false
		return &message
	}

	// If body parsing was successful
	message.Parsed = true
	message.BodyData = bodyData
	if message.Uuid == "" { // Ensure UUID is set
		message.Uuid = uuid.New().String()
	}
	return &message
}

// cleanMessage removes leading/trailing whitespace, empty lines, and standard message footers (like NNNN) from the raw message text.
// It attempts to isolate the core message content for easier header and body parsing.
// text: The raw message string.
// Returns a cleaned version of the message string.
// Note: The effectiveness of this function depends on the `emptyLineRemove` and `bodyOnly` regexes (not shown here).
func cleanMessage(text string) string {
	cleanedText := emptyLineRemove.ReplaceAllString(text, "") // emptyLineRemove is a package-level precompiled regex.
	cleanText := strings.ReplaceAll(cleanedText, "\n\n", "\n")

	// The bodyOnly regex aims to strip any higher-level SITA envelope or transmission-related lines
	// that are not part of the core aviation message header and body.
	if match := bodyOnly.FindStringSubmatch(cleanText); len(match) > 1 { // bodyOnly is a package-level precompiled regex.
		bodyContent := match[2] // Assuming group 2 of bodyOnly captures the core message.
		if len(bodyContent) > 0 && bodyContent[len(bodyContent)-1] == '\n' {
			bodyContent = bodyContent[:len(bodyContent)-1] // Remove trailing newline if present
		}
		// The NNNN (end of message) marker is typically handled by parseHeaderFieldsAndBody
		// if it's part of the content passed to it. This function's primary role is broader cleaning.
		return bodyContent
	}
	// Fallback if bodyOnly regex doesn't match (e.g., unexpected message format).
	// Log this eventuality or handle as an error if strict format adherence is required.
	utils.GetSugaredLogger().Debugf("cleanMessage: bodyOnly regex did not match for text starting with: %s", utils.FirstNChars(cleanText, 30))
	return cleanText // Return the line-cleaned text
}

// ParseHeader parses the header part of a complete aviation message string.
// It extracts fields like MessageID, DateTime, PriorityIndicator, Addresses, and Originator information.
// fullMessage: The complete raw aviation message string.
// Returns a domain.ParsedMessage struct populated with header fields and the extracted Body string,
// or an error if critical header components are missing or malformed.
// The returned ParsedMessage will always have the original Content set.
func ParseHeader(fullMessage string) (domain.ParsedMessage, error) {
	log := utils.GetSugaredLogger()
	// Initialize a ParsedMessage to hold results and original content.
	// ReceivedAt is set here as it's the first point of processing this message.
	pm := domain.ParsedMessage{Content: fullMessage, ReceivedAt: time.Now()}

	cleanedMessageText := cleanMessage(fullMessage) // Clean the full message first
	lines := strings.Split(cleanedMessageText, "\n")

	// Minimum lines for a valid message: ZCZC line, Priority/Address line, and at least one more line
	// which could be an originator line or a body marker like "(".
	if len(lines) < 2 { // Adjusted this check; originator might not be present, body marker is key.
		                    // If only ZCZC and Addr line, subsequent checks will fail.
		errMsg := fmt.Sprintf("invalid message format: not enough lines (found %d) for message starting with: %s", len(lines), utils.FirstNChars(fullMessage, 30))
		log.Warnf(errMsg)
		pm.Comments = errMsg
		return pm, fmt.Errorf(errMsg)
	}

	// The first line should contain the start indicator, message ID, and date/time.
	startIndicator, messageID, dateTime, err := parseStartIndicator(lines[0])
	if err != nil {
		log.Warnf("Failed to parse start indicator from line '%s': %v", lines[0], err)
		pm.Comments = fmt.Sprintf("Error parsing start indicator: %v. Line: '%s'", err, lines[0])
		return pm, fmt.Errorf("parsing start indicator failed: %w", err)
	}
	// TODO: Add StartIndicator back to domain.ParsedMessage if it's intended to be stored.
	// pm.StartIndicator = startIndicator 
	pm.MessageID = messageID
	pm.DateTime = dateTime

	// The second line should contain the priority indicator and primary address.
	if len(lines) < 2 { // Should have been caught by earlier len check, but good for safety.
		errMsg := fmt.Sprintf("invalid message format: missing priority/address line for message: %s", fullMessage)
		log.Warnf(errMsg)
		pm.Comments = errMsg
		return pm, fmt.Errorf(errMsg)
	}
	priorityIndicator, primaryAddress, err := parsePriorityAndPrimary(lines[1])
	if err != nil {
		log.Warnf("Failed to parse priority and primary address from line '%s': %v", lines[1], err)
		pm.Comments = fmt.Sprintf("Error parsing priority/address: %v. Line: '%s'", err, lines[1])
		return pm, fmt.Errorf("parsing priority and primary address failed: %w", err)
	}
	pm.PriorityIndicator = priorityIndicator
	pm.PrimaryAddress = primaryAddress

	// The remaining lines (if any) contain secondary addresses, originator info, and the message body.
	// If only 2 lines were present, lines[2:] will be an empty slice.
	remainingLines := []string{}
	if len(lines) > 2 {
		remainingLines = lines[2:]
	}
	headerData, body := parseHeaderFieldsAndBody(remainingLines)
	pm.SecondaryAddresses = headerData.SecondaryAddresses
	pm.Originator = headerData.Originator
	pm.OriginatorDateTime = headerData.OriginatorDateTime
	pm.Body = body // Store the extracted body string for further parsing by BodyParser

	return pm, nil
}

// parseStartIndicator extracts the start indicator, message ID, and date/time from the first line of a message.
// line: The first line of the message.
// Returns the start indicator string, message ID string, date/time string, and an error if parsing fails.
// StartIndicatorPrefix is a package-level constant (e.g., "ZCZC").
func parseStartIndicator(line string) (string, string, string, error) {
	parts := strings.Fields(line)
	if len(parts) >= 3 && strings.HasPrefix(parts[0], StartIndicatorPrefix) {
		return parts[0], parts[1], parts[2], nil
	}
	// Log is omitted here as the caller (ParseHeader) logs this specific error.
	return "", "", "", fmt.Errorf("invalid start indicator line format: '%s'", line)
}

// parsePriorityAndPrimary extracts the priority indicator and primary address from the second line of a message.
// line: The second line of the message.
// Returns the priority indicator string, primary address string, and an error if parsing fails.
func parsePriorityAndPrimary(line string) (string, string, error) {
	parts := strings.Fields(line)
	if len(parts) >= 2 {
		return parts[0], parts[1], nil
	}
	// Log is omitted here as the caller (ParseHeader) logs this specific error.
	return "", "", fmt.Errorf("invalid priority and primary address line format: '%s'", line)
}

// ParsedHeaderFields holds the extracted data from header lines processed by parseHeaderFieldsAndBody.
type ParsedHeaderFields struct {
	SecondaryAddresses string // Concatenated secondary addresses.
	Originator         string // Originator identifier.
	OriginatorDateTime string // Originator timestamp.
}

// parseHeaderFieldsAndBody processes lines after the primary address line (lines[2:] from cleaned message),
// extracting secondary addresses, originator information, and the message body content.
// lines: A slice of strings, where each string is a line from the message header/body.
// Returns ParsedHeaderFields containing extracted header data, and a string for the message body.
// EndHeaderMarker, BeginPartMarker, and the originator regex are package-level constants/variables.
func parseHeaderFieldsAndBody(lines []string) (ParsedHeaderFields, string) {
	var (
		headerData    ParsedHeaderFields
		bodyAndFooter strings.Builder
		headerParsing bool = true 
	)

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if !headerParsing {
			bodyAndFooter.WriteString(trimmedLine + "\n")
			continue
		}

		switch {
		case trimmedLine == EndHeaderMarker:
			// Explicit end of address fields. No specific action, parsing continues to check for originator or body.
		case strings.HasPrefix(trimmedLine, "."): // Originator line explicitly prefixed with "."
			originatorInfo := strings.Fields(trimmedLine[1:]) 
			if len(originatorInfo) >= 2 {
				headerData.Originator = originatorInfo[0]
				headerData.OriginatorDateTime = originatorInfo[1]
			} else {
				utils.GetSugaredLogger().Warnf("Potentially malformed prefixed originator line: '%s'", trimmedLine)
			}
		case strings.HasPrefix(trimmedLine, BeginPartMarker) || strings.HasPrefix(trimmedLine, "("): // Start of message body
			headerParsing = false 
			if strings.Contains(trimmedLine, "NNNN") { // End of message marker "NNNN"
				// If NNNN is on this line, it might be content then NNNN, or just NNNN.
				// The original logic was to break, effectively skipping the line if NNNN was present.
				// This means content on the same line as NNNN (but before it) might be lost if it's a body-starting line.
				// For now, if NNNN is on a line that starts a body part, the line is skipped.
				break 
			}
			bodyAndFooter.WriteString(trimmedLine + "\n")
		default: // Line is either a secondary address or an unprefixed originator.
			dateTime, orig := getOriginator(trimmedLine) // Attempt to parse as originator
			if orig != "" { 
				if headerData.Originator == "" { // Set only if not already found by "." prefix
					headerData.OriginatorDateTime = dateTime
					headerData.Originator = orig
				} else { 
					utils.GetSugaredLogger().Debugf("Originator already parsed by explicit prefix, treating line as secondary address: '%s'", trimmedLine)
					headerData.SecondaryAddresses = strings.TrimSpace(headerData.SecondaryAddresses + " " + trimmedLine)
				}
			} else {
				headerData.SecondaryAddresses = strings.TrimSpace(headerData.SecondaryAddresses + " " + trimmedLine)
			}
		}
	}
	
	bodyStr := bodyAndFooter.String()
	if len(bodyStr) > 0 && bodyStr[len(bodyStr)-1] == '\n' {
		bodyStr = bodyStr[:len(bodyStr)-1] // Remove single trailing newline
	}

	return headerData, bodyStr
}

// getOriginator attempts to parse a line as an originator line (DateTime and Originator ID)
// using a predefined regular expression (package-level 'originator' variable).
// line: The input string line to parse.
// Returns the extracted DateTime string and Originator ID string.
// If parsing fails (no match), both returned strings are empty.
func getOriginator(line string) (string, string) {
	match := originator.FindStringSubmatch(line) 
	if len(match) >= 3 { 
		return match[1], match[2] 
	}
	return "", ""
}

// parseOther extracts various data fields from the "other information" string (typically FPL Field 18)
// using a list of predefined regular expression patterns (package-level 'otherPatterns' slice).
// otherInfoText: The string containing the "other information" fields.
// Returns a map where keys are field identifiers (e.g., PBN, NAV, REG) and values are the extracted data.
func parseOther(otherInfoText string) map[string]string {
	extractedData := make(map[string]string)
	for _, pattern := range otherPatterns { 
		if match := pattern.FindStringSubmatch(otherInfoText); len(match) > 0 {
			for i, name := range pattern.SubexpNames() {
				if i != 0 && name != "" { // Group 0 is full match, only care about named groups.
					extractedData[name] = strings.TrimSpace(match[i])
				}
			}
		}
	}
	return extractedData
}

func (parser *BodyParser) createArrivalData(data map[string]string) (*domain.ARR, error) {
	return &domain.ARR{
		Category:         data[Category],
		AircraftID:       data[FlightNumber],
		SSRModeAndCode:   data[SSR],
		DepartureAirport: data[DepartureCode],
		ArrivalAirport:   data[ArrivalCode],
		ArrivalTime:      data[ArrivalTime],
	}, nil
}

func (parser *BodyParser) createDepartureData(data map[string]string) (*domain.DEP, error) {
	return &domain.DEP{
		Category:         data[Category],
		AircraftID:       data[FlightNumber],
		SSRModeAndCode:   data[SSR],
		DepartureAirport: data[DepartureCode],
		DepartureTime:    data[DepartureTime],
		Destination:      data[ArrivalCode],
	}, nil
}

func (parser *BodyParser) createCancellationData(data map[string]string) (*domain.CNL, error) {
	return &domain.CNL{
		Category:           data["category"], // "category" key is lowercase in this case
		AircraftID:         data[FlightNumber],
		DepartureAirport:   data[DepartureCode],
		DestinationAirport: data[ArrivalCode],
	}, nil
}

func (parser *BodyParser) createDelayData(data map[string]string) (*domain.DLA, error) {
	return &domain.DLA{
		Category:         data[Category],
		AircraftID:       data[FlightNumber],
		DepartureAirport: data[DepartureCode],
		NewDepartureTime: data[DepartureTime],
		ArrivalAirport:   data[ArrivalCode],
		ArrivalTime:      data[ArrivalTime],
	}, nil
}

func (parser *BodyParser) createFlightPlanData(data map[string]string) (*domain.FPL, error) {
	otherData := parseOther(data[OtherInfo])
	return &domain.FPL{
		Category:                data[Category],
		FlightNumber:            data[FlightNumber], // This is Aircraft ID from Field 7 for FPL
		ReferenceData:           data[ReferenceData],
		AircraftID:              data[AircraftIDFPL], // Specific for FPL structure, e.g. "aircraft_id_fpl"
		SSRModeAndCode:          data[Surveillance],  // Field 10b
		FlightRulesAndType:      data[Indicator],     // Field 8
		CruisingSpeedAndLevel:   data[Speed] + data[Level],
		DepartureAirport:        data[DepartureCode],
		DepartureTime:           data[DepartureTime],
		Route:                   data[Route],
		DestinationAndTotalTime: data[DestinationCode] + data[EstimatedTime],
		AlternateAirport:        data[AlternateAirport],
		OtherInfo:               data[OtherInfo],
		Register:                otherData[Register],
		EstimatedArrivalTime:    data[EstimatedTime],
		PBN:                     otherData[PBN],
		NavigationEquipment:     otherData[NavigationEquipment],
		EstimatedElapsedTime:    otherData[EstimatedElapsedTime],
		SELCALCode:              otherData[SELCALCode],
		PerformanceCategory:     otherData[PerformanceCategory],
		RerouteInformation:      otherData[RerouteInformation],
		Remarks:                 otherData[Remarks],
	}, nil
}

func (parser *BodyParser) createBodyData(data map[string]string) (string, interface{}, error) {
	category := data["category"]
	switch category {
	case CategoryArrival:
		arrData, err := parser.createArrivalData(data)
		return category, arrData, err
	case CategoryDeparture:
		depData, err := parser.createDepartureData(data)
		return category, depData, err
	case CategoryCancellation:
		cnlData, err := parser.createCancellationData(data)
		return category, cnlData, err
	case CategoryDelay:
		dlaData, err := parser.createDelayData(data)
		return category, dlaData, err
	case CategoryFlightPlan:
		fplData, err := parser.createFlightPlanData(data)
		return category, fplData, err
	default:
		return category, nil, fmt.Errorf("invalid message type: %s", category)
	}
}

func Parse(rawText string) *domain.ParsedMessage {
	message, err := ParseHeader(rawText)
	if err != nil {
		msg := domain.NewParsedMessage()
		msg.Content = rawText
		return msg
	}

	bodyParser := NewBodyParser(message.Body)
	category, bodyData, err := bodyParser.Parse()
	message.Category = category
	message.ParsedAt = time.Now()

	if err != nil {
		message.Comments = err.Error()
		return &message
	}
	message.Parsed = true
	message.BodyData = bodyData
	message.Uuid = uuid.New().String()
	return &message
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

// ParseHeader parses the header of a raw message string and returns a ParsedMessage struct.
// It handles initial cleaning, splitting into lines, and delegates to specific parsing functions for different header parts.
func ParseHeader(fullMessage string) (domain.ParsedMessage, error) {
	log := utils.GetSugaredLogger()
	cleanedMessageBody := cleanMessage(fullMessage) // Renamed `cleaned` to `cleanedMessageBody`
	lines := strings.Split(cleanedMessageBody, "\n")

	if len(lines) < 3 { // Minimum lines for a valid message (start, priority/primary, originator)
		log.Warnf("Invalid message format: not enough lines. Message: %s", fullMessage)
		return domain.ParsedMessage{Content: fullMessage}, fmt.Errorf("invalid message format: not enough lines for message: %s", fullMessage)
	}

	// The first line should contain the start indicator, message ID, and date/time.
	_, messageID, dateTime, err := parseStartIndicator(lines[0])
	if err != nil {
		log.Warnf("Failed to parse start indicator from line '%s': %v", lines[0], err)
		return domain.ParsedMessage{Content: fullMessage}, fmt.Errorf("parsing start indicator failed: %w", err)
	}

	// The second line should contain the priority indicator and primary address.
	priorityIndicator, primaryAddress, err := parsePriorityAndPrimary(lines[1])
	if err != nil {
		log.Warnf("Failed to parse priority and primary address from line '%s': %v", lines[1], err)
		// Decide if this is a fatal error or if we can proceed with partial data
		// For now, let's assume it's fatal for header parsing to ensure data integrity
		return domain.ParsedMessage{Content: fullMessage}, fmt.Errorf("parsing priority and primary address failed: %w", err)
	}

	// The remaining lines contain secondary addresses, originator info, and the message body.
	headerData, body := parseHeaderFieldsAndBody(lines[2:])

	return domain.ParsedMessage{
		MessageID:          messageID,
		DateTime:           dateTime,
		PriorityIndicator:  priorityIndicator,
		PrimaryAddress:     primaryAddress,
		SecondaryAddresses: headerData.SecondaryAddresses,
		Originator:         headerData.Originator,
		OriginatorDateTime: headerData.OriginatorDateTime,
		Content:            fullMessage,
		Body:               body,
		ReceivedAt:         time.Now(),
	}, nil
}

// parseStartIndicator extracts the start indicator, message ID, and date/time from the first line of a message.
func parseStartIndicator(line string) (string, string, string, error) {
	parts := strings.Fields(line)
	// Expecting at least 3 parts: StartIndicator, MessageID, DateTime
	if len(parts) >= 3 && strings.HasPrefix(parts[0], StartIndicatorPrefix) {
		return parts[0], parts[1], parts[2], nil
	}
	utils.GetSugaredLogger().Warnf("Invalid start indicator line format: '%s'", line)
	return "", "", "", fmt.Errorf("invalid start indicator line format: '%s'", line)
}

// parsePriorityAndPrimary extracts the priority indicator and primary address from the second line of a message.
func parsePriorityAndPrimary(line string) (string, string, error) {
	parts := strings.Fields(line)
	// Expecting at least 2 parts: PriorityIndicator, PrimaryAddress
	if len(parts) >= 2 {
		return parts[0], parts[1], nil
	}
	utils.GetSugaredLogger().Warnf("Invalid priority and primary address line format: '%s'", line)
	return "", "", fmt.Errorf("invalid priority and primary address line format: '%s'", line)
}

// ParsedHeaderFields holds the extracted data from header lines.
type ParsedHeaderFields struct {
	SecondaryAddresses string
	Originator         string
	OriginatorDateTime string
}

// parseHeaderFieldsAndBody processes lines after the primary address line,
// extracting secondary addresses, originator information, and the message body.
func parseHeaderFieldsAndBody(lines []string) (ParsedHeaderFields, string) {
	var (
		headerData    ParsedHeaderFields
		bodyAndFooter strings.Builder
		headerParsing bool = true // Indicates if we are currently parsing header fields
	)

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if !headerParsing {
			bodyAndFooter.WriteString(trimmedLine + "\n")
			continue
		}

		switch {
		// EndHeaderMarker signifies the end of header addresses explicitly.
		case trimmedLine == EndHeaderMarker:
			// No specific action needed other than to continue to next lines which might be originator or start of body.
			// This marker is more for visual separation in the raw message.
		// Originator line typically starts with a dot.
		case strings.HasPrefix(trimmedLine, "."):
			originatorInfo := strings.Fields(trimmedLine[1:]) // Skip the dot
			if len(originatorInfo) >= 2 {
				headerData.Originator = originatorInfo[0]
				headerData.OriginatorDateTime = originatorInfo[1]
			} else {
				utils.GetSugaredLogger().Warnf("Potentially malformed originator line: '%s'", trimmedLine)
				// Decide: treat as secondary address or log and skip? For now, log and assume end of specific header fields.
			}
			// After originator line, any subsequent lines are part of the body or special markers.
			// However, the original logic implied header could end here or with BeginPartMarker.
			// To simplify, let's assume after explicitly finding an originator, we are done with header address fields.
			// The next check for BeginPartMarker or "(" will then transition to body.
			// If this line IS the originator, we are done with address parsing.
			// The problem is if secondary addresses appear AFTER the originator line in some message formats.
			// The original logic `headerEnded = true` after this.
			// Let's stick to: originator line means other addresses are done.
			// The next iteration will either be body or a marker like "(".

		// BeginPartMarker or an opening parenthesis usually indicates the start of the message body.
		case strings.HasPrefix(trimmedLine, BeginPartMarker) || strings.HasPrefix(trimmedLine, "("):
			headerParsing = false // Stop parsing header fields, switch to body.
			// Check for "NNNN" which indicates end of message, not part of body content.
			if strings.Contains(trimmedLine, "NNNN") { // Changed from Index > 0 to Contains for robustness
				// If NNNN is present, this line might be just the marker, or content then marker.
				// The original logic was `if strings.Index(line, "NNNN") > 0 { break }`.
				// This implies if NNNN is found, this line is skipped.
				// However, content *before* NNNN could be part of the body.
				// For now, if NNNN is on a line that starts with ( or BeginPartMarker, we treat this line as non-body.
				// This might need refinement based on exact message specs.
				break // Skip this line from being added to body
			}
			bodyAndFooter.WriteString(trimmedLine + "\n")

		// Default case: line is considered a secondary address or part of originator if not matched above.
		default:
			// Try to parse as an originator line if not explicitly marked with "."
			// This handles cases where originator info might not be prefixed.
			// The original `getOriginator` was called here.
			dateTime, orig := getOriginator(trimmedLine)
			if orig != "" { // Successfully parsed as an originator line
				if headerData.Originator == "" { // Only set if not already found by "." prefix
					headerData.OriginatorDateTime = dateTime
					headerData.Originator = orig
				} else {
					// Originator already found, this might be a secondary address that looks like an originator
					// or a misformatted message. For now, append to secondary addresses.
					utils.GetSugaredLogger().Debugf("Originator already parsed, treating line as secondary address: '%s'", trimmedLine)
					headerData.SecondaryAddresses = strings.TrimSpace(headerData.SecondaryAddresses + " " + trimmedLine)
				}
			} else {
				// Not an originator line, so it's a secondary address.
				headerData.SecondaryAddresses = strings.TrimSpace(headerData.SecondaryAddresses + " " + trimmedLine)
			}
		}
	}
	// Clean up body string: remove trailing newline if body is non-empty
	bodyStr := bodyAndFooter.String()
	if len(bodyStr) > 0 && bodyStr[len(bodyStr)-1] == '\n' {
		bodyStr = bodyStr[:len(bodyStr)-1]
	}

	return headerData, bodyStr
}


// getOriginator attempts to parse a line as an originator line (DateTime and Originator ID).
// Returns DateTime and Originator ID. If parsing fails, both are empty strings.
func getOriginator(line string) (string, string) {
	match := originator.FindStringSubmatch(line) // `originator` is a package-level regex variable
	if len(match) >= 3 { // Expecting full match + 2 capture groups
		return match[1], match[2] // Group 1: DateTime, Group 2: Originator ID
	}
	// It's common for lines not to be originator lines, so warning here might be too verbose.
	// Consider logging at Debug level or removing if this function is purely a "try-parse".
	// utils.GetSugaredLogger().Warnf("Invalid originator line format: %s", line) // Original line
	return "", ""
}

// parseOther extracts various data fields from the "other information" string (Field 18 of FPL)
// using a list of predefined regular expression patterns.
// otherInfoText: The string containing the "other information" fields.
// Returns a map where keys are field identifiers (e.g., PBN, NAV, REG) and values are the extracted data.
func parseOther(otherInfoText string) map[string]string {
	extractedData := make(map[string]string)
	// Iterate through each predefined regex pattern for "other information" fields.
	for _, pattern := range otherPatterns {
		// Attempt to find a match for the current pattern in the otherInfoText.
		if match := pattern.FindStringSubmatch(otherInfoText); len(match) > 0 {
			// If a match is found, extract data from named capture groups.
			for i, name := range pattern.SubexpNames() {
				// Group 0 is the full match, and we only care about named groups.
				if i != 0 && name != "" {
					extractedData[name] = strings.TrimSpace(match[i])
				}
			}
		}
	}
	return extractedData
}
