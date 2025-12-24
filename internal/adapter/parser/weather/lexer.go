package weather

import "strings"

// Tokenize splits the raw text into tokens by whitespace
func Tokenize(raw string) []string {
	raw = strings.TrimSpace(raw)
	// Remove trailing '=' if present
	if strings.HasSuffix(raw, "=") {
		raw = raw[:len(raw)-1]
		raw = strings.TrimSpace(raw)
	}

	tokens := strings.Fields(raw)
	return tokens
}

// Section represents a special section in the report (RMK, TEMPO, BECMG, FM)
type Section struct {
	Type      string   // "RMK", "TEMPO", "BECMG", "FM"
	StartIdx  int      // Starting token index
	EndIdx    int      // Ending token index (exclusive)
	Tokens    []string // Tokens in this section
}

// FindSpecialSections identifies special sections in the tokenized report
func FindSpecialSections(tokens []string) []Section {
	var sections []Section
	var currentSection *Section

	for i, token := range tokens {
		upperToken := strings.ToUpper(token)

		switch {
		case strings.HasPrefix(upperToken, "RMK"):
			if currentSection != nil {
				currentSection.EndIdx = i
				sections = append(sections, *currentSection)
			}
			currentSection = &Section{
				Type:     "RMK",
				StartIdx: i,
				Tokens:   []string{token},
			}

		case strings.HasPrefix(upperToken, "TEMPO"):
			if currentSection != nil && currentSection.Type != "RMK" {
				currentSection.EndIdx = i
				sections = append(sections, *currentSection)
			}
			currentSection = &Section{
				Type:     "TEMPO",
				StartIdx: i,
				Tokens:   []string{token},
			}

		case strings.HasPrefix(upperToken, "BECMG"):
			if currentSection != nil && currentSection.Type != "RMK" {
				currentSection.EndIdx = i
				sections = append(sections, *currentSection)
			}
			currentSection = &Section{
				Type:     "BECMG",
				StartIdx: i,
				Tokens:   []string{token},
			}

		case strings.HasPrefix(upperToken, "FM"):
			if currentSection != nil && currentSection.Type != "RMK" {
				currentSection.EndIdx = i
				sections = append(sections, *currentSection)
			}
			currentSection = &Section{
				Type:     "FM",
				StartIdx: i,
				Tokens:   []string{token},
			}

		default:
			if currentSection != nil {
				currentSection.Tokens = append(currentSection.Tokens, token)
			}
		}
	}

	// Close the last section
	if currentSection != nil {
		currentSection.EndIdx = len(tokens)
		sections = append(sections, *currentSection)
	}

	return sections
}

// ExtractSectionTokens extracts tokens for a specific section type
func ExtractSectionTokens(tokens []string, sectionType string) []string {
	sections := FindSpecialSections(tokens)
	for _, section := range sections {
		if section.Type == sectionType {
			return section.Tokens
		}
	}
	return nil
}

