package aviation

import (
	"caatsm/internal/domain"
	"testing"
)

func TestTokenizerDefaultWhitespace(t *testing.T) {
	t.Parallel()

	input := "A B\nC\tD\rE"
	tokens := Tokenizer{}.Tokenize(input)

	expected := []Token{
		{Text: "A", Start: 0, End: 1},
		{Text: "B", Start: 2, End: 3},
		{Text: "C", Start: 4, End: 5},
		{Text: "D", Start: 6, End: 7},
		{Text: "E", Start: 8, End: 9},
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, token := range tokens {
		if token != expected[i] {
			t.Fatalf("token %d mismatch: got %#v, expected %#v", i, token, expected[i])
		}
	}
}

func TestTokenizerSlashWhitespace(t *testing.T) {
	t.Parallel()

	input := "A/B C"
	tokens := Tokenizer{Whitespace: " \n\t\r/"}.Tokenize(input)

	expected := []Token{
		{Text: "A", Start: 0, End: 1},
		{Text: "/", Start: 1, End: 2},
		{Text: "B", Start: 2, End: 3},
		{Text: "C", Start: 4, End: 5},
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, token := range tokens {
		if token != expected[i] {
			t.Fatalf("token %d mismatch: got %#v, expected %#v", i, token, expected[i])
		}
	}
}

func TestBuildBodyPatternsIncludesCategories(t *testing.T) {
	t.Parallel()

	patterns := buildBodyPatterns()
	for _, category := range []string{
		CategoryArrival,
		CategoryDeparture,
		CategoryCancellation,
		CategoryDelay,
		CategoryFlightPlan,
	} {
		config, ok := patterns[category]
		if !ok {
			t.Fatalf("expected category %s in body patterns", category)
		}
		if len(config.Patterns) == 0 || config.Patterns[0].Expression == nil {
			t.Fatalf("expected pattern expression for category %s", category)
		}
	}
}

func TestParseCategoryInvalid(t *testing.T) {
	t.Parallel()

	_, err := parseCategory("XYZ", ParseContext{}, map[string]string{})
	if err == nil {
		t.Fatal("expected error for invalid category")
	}
}

func TestParseCategoryFPL(t *testing.T) {
	t.Parallel()

	body := `(FPL-CCA1532-IS
-A332/H
-SDE3FGHIJ4J5M1RWY/LB101
-ZSSS2035
-K0859S1040 PIAKS G330 PIMOL A539 BTO W82 DOGAR
-ZBAA0153 ZBYN
-PBN/A1B2B3B4B5D1L1 NAV/ABAS REG/B6513 EET/ZBPE0112 SEL/KMAL PER/C RIF/FRT N640 ZBYN RMK/TCAS EQUIPPED)`

	data := extract(body, FplPatternExpression)
	if data == nil {
		t.Fatal("expected FPL pattern to match")
	}

	parsed, err := parseCategory(CategoryFlightPlan, ParseContext{
		Body:   body,
		Tokens: Tokenizer{}.Tokenize(body),
	}, data)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	fpl, ok := parsed.(*domain.FPL)
	if !ok {
		t.Fatalf("expected *domain.FPL, got %T", parsed)
	}

	if fpl.FlightNumber != "CCA1532" {
		t.Fatalf("expected flight number CCA1532, got %s", fpl.FlightNumber)
	}
	if fpl.PBN != "A1B2B3B4B5D1L1" {
		t.Fatalf("expected PBN A1B2B3B4B5D1L1, got %s", fpl.PBN)
	}
	if fpl.RerouteInformation != "FRT N640 ZBYN" {
		t.Fatalf("expected reroute information, got %s", fpl.RerouteInformation)
	}
	if fpl.Remarks != "TCAS EQUIPPED" {
		t.Fatalf("expected remarks TCAS EQUIPPED, got %s", fpl.Remarks)
	}
}
