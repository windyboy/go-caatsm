package aviation

import (
	"caatsm/internal/adapter/dto"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	otherPatterns = []*regexp.Regexp{navPattern,
		remarkPattern,
		selPattern,
		pbnPattern,
		eetPattern,
		performancePattern,
		regPattern,
		reroutePattern}
	// ErrHeaderParse indicates an invalid header section.
	ErrHeaderParse = errors.New("invalid telegram header")
	// ErrBodyParse indicates a failure matching the telegram body.
	ErrBodyParse = errors.New("invalid telegram body")
)

type BodyParser struct {
	body         string
	bodyPatterns map[string]BodyConfig
}

func NewBodyParser(body string) *BodyParser {
	return &BodyParser{
		bodyPatterns: bodyPatterns,
		body:         body,
	}
}

// GetBodyPatterns returns the body patterns map.
// The bodyPatterns map is a reference to the package-level bodyPatterns,
// which is initialized once at startup and never modified, making it safe
// for concurrent reads without synchronization.
func (parser *BodyParser) GetBodyPatterns() map[string]BodyConfig {
	return parser.bodyPatterns
}

func (parser *BodyParser) Parse() (string, interface{}, error) {
	parser.body = strings.TrimSpace(parser.body)
	category := findCategory(parser.body)
	if category == "" {
		return "", nil, fmt.Errorf("no category found in body text")
	}

	patternConfig, exists := parser.bodyPatterns[category]
	if !exists || patternConfig.Patterns == nil {
		return "", nil, fmt.Errorf("no matching pattern found for category: %s", category)
	}

	ctx := ParseContext{
		Body:   parser.body,
		Tokens: Tokenizer{}.Tokenize(parser.body),
	}

	for _, p := range patternConfig.Patterns {
		if data := extract(parser.body, p.Expression); data != nil {
			parsed, err := parseCategory(category, ctx, data)
			if err != nil {
				return "", nil, err
			}
			return category, parsed, nil
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

func extract(data string, exp *regexp.Regexp) map[string]string {
	// Use timeout protection to prevent ReDoS attacks
	match, err := MatchWithTimeout(exp, data, DefaultRegexTimeout)
	if err != nil {
		// Timeout occurred - return nil to indicate no match
		return nil
	}
	if len(match) > 0 {
		return extractData(match, exp)
	}
	return nil
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

func headerToParsedTelegram(header Header) dto.ParsedTelegram {
	return dto.ParsedTelegram{
		MessageID:          header.MessageID,
		DateTime:           header.DateTime,
		PriorityIndicator:  header.PriorityIndicator,
		PrimaryAddress:     header.PrimaryAddress,
		SecondaryAddresses: header.SecondaryAddresses,
		Originator:         header.Originator,
		OriginatorDateTime: header.OriginatorDateTime,
		Category:           header.Category,
		Body:               header.Body,
		Content:            header.Content,
		ReceivedAt:         header.ReceivedAt,
		ParsedAt:           header.ParsedAt,
	}
}

// Parse parses a raw ICAO aviation telegram and returns a ParsedTelegram with parsing status.
//
// IMPORTANT ERROR HANDLING PATTERN:
// This function intentionally returns both a non-nil ParsedTelegram AND an error when parsing fails.
// This design decision allows the caller to persist failed parse attempts with error details to the
// database for audit and compliance purposes. This pattern is specific to the aviation parser's
// error handling strategy where parser failures are permanent (ACK'd, not retried) and must be
// stored for regulatory compliance and troubleshooting.
//
// Error Handling Strategy:
//   - Input validation failure: Returns ParsedTelegram with MessageStatusHeaderError + ErrHeaderParse
//   - Header parse failure: Returns ParsedTelegram with MessageStatusHeaderError + ErrHeaderParse
//   - Body parse failure: Returns ParsedTelegram with MessageStatusBodyError + ErrBodyParse
//   - Success: Returns ParsedTelegram with MessageStatusParsed + nil error
//
// The returned ParsedTelegram is ALWAYS non-nil and safe to use, even when error is non-nil.
// Callers should check both the error and the ParsedTelegram.Status field to determine the outcome.
//
// Security:
//   - Input size validation prevents DoS attacks (max 1800 chars per AFTN standard)
//   - Regex timeout protection prevents ReDoS attacks (100ms timeout)
//
// Example usage:
//
//	parsed, err := Parse(rawTelegram)
//	if err != nil {
//	    // Parse failed, but parsed contains error details for storage
//	    repository.InsertRaw(parsed) // Store for audit
//	    return Permanent(err)         // Don't retry
//	}
//	// Parse succeeded
//	repository.Insert(parsed)
//	publisher.Publish(parsed.BodyData)
func Parse(rawText string) (*dto.ParsedTelegram, error) {
	// Validate input size to prevent DoS attacks
	if err := ValidateInputSize(rawText); err != nil {
		msg := dto.NewParsedTelegram()
		msg.Content = rawText
		msg.Comments = err.Error()
		msg.ErrorReason = err.Error()
		msg.Status = dto.MessageStatusHeaderError
		return msg, fmt.Errorf("%w: %w", ErrHeaderParse, err)
	}

	header, err := ParseHeader(rawText)
	if err != nil {
		msg := dto.NewParsedTelegram()
		msg.Content = rawText
		msg.Comments = err.Error()
		msg.ErrorReason = err.Error()
		msg.Status = dto.MessageStatusHeaderError
		return msg, fmt.Errorf("%w: %w", ErrHeaderParse, err)
	}

	bodyParser := NewBodyParser(header.Body)
	category, bodyData, bodyErr := bodyParser.Parse()
	header.Category = category
	header.ParsedAt = time.Now()

	if bodyErr != nil {
		parsed := headerToParsedTelegram(header)
		parsed.Parsed = false
		parsed.Comments = bodyErr.Error()
		parsed.Status = dto.MessageStatusBodyError
		parsed.ErrorReason = bodyErr.Error()
		return &parsed, fmt.Errorf("%w: %w", ErrBodyParse, bodyErr)
	}

	parsed := headerToParsedTelegram(header)
	parsed.BodyData = bodyData
	parsed.Parsed = true
	parsed.Status = dto.MessageStatusParsed
	parsed.Uuid = uuid.New().String()
	return &parsed, nil
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

// ParseHeader parses only the header portion of the message and returns a lightweight struct
// with header fields and body content. It is used internally by the aviation parser.
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

// ParseHeader parses the header portion of an ICAO telegram.
// It extracts message metadata (ID, datetime, addresses, originator) and separates
// the body content for subsequent parsing.
//
// Returns Header struct with parsed fields and the raw body content.
// On error, returns Header with Content field populated for audit purposes.
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

	priorityIndicator, primaryAddress := parsePriorityAndPrimary(lines[1])
	secondaryAddresses, originator, originatorDateTime, body := parseRemainingLines(lines[2:])

	return Header{
		MessageID:          messageID,
		DateTime:           dateTime,
		PriorityIndicator:  priorityIndicator,
		PrimaryAddress:     primaryAddress,
		SecondaryAddresses: secondaryAddresses,
		Originator:         originator,
		OriginatorDateTime: originatorDateTime,
		Content:            fullMessage,
		Body:               body,
		ReceivedAt:         time.Now(),
	}, nil
}

func parseStartIndicator(line string) (string, string, string, error) {
	parts := strings.Fields(line)
	if len(parts) >= MinStartIndicatorParts && strings.HasPrefix(parts[0], StartIndicatorPrefix) {
		return parts[0], parts[1], parts[2], nil
	}
	return "", "", "", fmt.Errorf("invalid start indicator line format: %s", line)
}

func parsePriorityAndPrimary(line string) (string, string) {
	parts := strings.Fields(line)
	if len(parts) >= MinPriorityLineParts {
		return parts[0], parts[1]
	}
	return "", ""
}

// isOriginatorLine checks if a dot-prefixed line matches the originator format (.CODE DATETIME).
// Returns the originator code, datetime, and whether it's a valid match.
func isOriginatorLine(line string) (originator, dateTime string, isMatch bool) {
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

// isBodyStartLine checks if a line indicates the start of the message body.
func isBodyStartLine(line string) bool {
	return strings.HasPrefix(line, BeginPartMarker) || strings.HasPrefix(line, "(")
}

func parseRemainingLines(lines []string) (string, string, string, string) {
	var (
		secondaryAddresses strings.Builder
		originator         string
		originatorDateTime string
		body               strings.Builder
		inBody             bool
	)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and single dots
		if line == "" || line == EndHeaderMarker {
			continue
		}

		// Once in body, collect all remaining lines
		if inBody {
			body.WriteString(line + "\n")
			continue
		}

		// Check for originator line (.CODE DATETIME)
		if orig, dt, isOrig := isOriginatorLine(line); isOrig {
			originator = orig
			originatorDateTime = dt
			inBody = true
			continue
		}

		// Check for body start markers
		if isBodyStartLine(line) {
			inBody = true
			// Skip lines containing NNNN (end marker)
			if !strings.Contains(line, "NNNN") {
				body.WriteString(line + "\n")
			}
			continue
		}

		// Try to parse as originator using regex (fallback)
		if dt, orig := getOriginator(line); orig != "" {
			originatorDateTime = dt
			originator = orig
			continue
		}

		// Otherwise, treat as secondary address
		secondaryAddresses.WriteString(" " + line)
	}

	return secondaryAddresses.String(), originator, originatorDateTime, body.String()
}

func getOriginator(line string) (string, string) {
	match := originator.FindStringSubmatch(line)
	if len(match) >= MinOriginatorMatchGroups {
		return match[1], match[2]
	}
	return "", ""
}

// isAllUppercaseLetters checks if a string contains only uppercase letters
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

// isAllDigits checks if a string contains only digits
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
