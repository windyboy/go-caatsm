package aviation

import (
	"caatsm/internal/domain"
	"fmt"
)

// ParseContext carries the raw body and tokens for field parsers.
type ParseContext struct {
	Body   string
	Tokens []Token
}

// CategoryParser maps a category to patterns and output parsing.
type CategoryParser interface {
	Category() string
	Patterns() []PatternConfig
	Parse(ctx ParseContext, data map[string]string) (interface{}, error)
}

type arrParser struct{}

func (arrParser) Category() string { return CategoryArrival }

func (arrParser) Patterns() []PatternConfig {
	return []PatternConfig{
		{
			Pattern:    ArrPatternString,
			Comments:   "Pattern for ARR message",
			Expression: ArrPatternExpression,
		},
	}
}

func (arrParser) Parse(_ ParseContext, data map[string]string) (interface{}, error) {
	return &domain.ARR{
		Category:         data[Category],
		AircraftID:       data[FlightNumber],
		SSRModeAndCode:   data[SSR],
		DepartureAirport: data[DepartureCode],
		ArrivalAirport:   data[ArrivalCode],
		ArrivalTime:      data[ArrivalTime],
	}, nil
}

type depParser struct{}

func (depParser) Category() string { return CategoryDeparture }

func (depParser) Patterns() []PatternConfig {
	return []PatternConfig{
		{
			Pattern:    DepPatternString,
			Comments:   "Pattern for DEP message",
			Expression: DepPatternExpression,
		},
	}
}

func (depParser) Parse(_ ParseContext, data map[string]string) (interface{}, error) {
	return &domain.DEP{
		Category:         data[Category],
		AircraftID:       data[FlightNumber],
		SSRModeAndCode:   data[SSR],
		DepartureAirport: data[DepartureCode],
		DepartureTime:    data[DepartureTime],
		Destination:      data[ArrivalCode],
	}, nil
}

type cnlParser struct{}

func (cnlParser) Category() string { return CategoryCancellation }

func (cnlParser) Patterns() []PatternConfig {
	return []PatternConfig{
		{
			Pattern:    CnlPatternString,
			Comments:   "Pattern for CNL message",
			Expression: CnlPatternExpression,
		},
	}
}

func (cnlParser) Parse(_ ParseContext, data map[string]string) (interface{}, error) {
	return &domain.CNL{
		Category:           data[Category],
		AircraftID:         data[FlightNumber],
		DepartureAirport:   data[DepartureCode],
		DestinationAirport: data[ArrivalCode],
	}, nil
}

type dlaParser struct{}

func (dlaParser) Category() string { return CategoryDelay }

func (dlaParser) Patterns() []PatternConfig {
	return []PatternConfig{
		{
			Pattern:    DlaPatternString,
			Comments:   "Pattern for DLA message",
			Expression: DlaPatternExpression,
		},
	}
}

func (dlaParser) Parse(_ ParseContext, data map[string]string) (interface{}, error) {
	return &domain.DLA{
		Category:         data[Category],
		AircraftID:       data[FlightNumber],
		DepartureAirport: data[DepartureCode],
		NewDepartureTime: data[DepartureTime],
		ArrivalAirport:   data[ArrivalCode],
		ArrivalTime:      data[ArrivalTime],
	}, nil
}

type fplParser struct{}

func (fplParser) Category() string { return CategoryFlightPlan }

func (fplParser) Patterns() []PatternConfig {
	return []PatternConfig{
		{
			Pattern:    FplPatternString,
			Comments:   "Pattern for FPL message",
			Expression: FplPatternExpression,
		},
	}
}

func (fplParser) Parse(_ ParseContext, data map[string]string) (interface{}, error) {
	otherData := parseOther(data[OtherInfo])
	return &domain.FPL{
		Category:                data[Category],
		FlightNumber:            data[FlightNumber],
		ReferenceData:           data[ReferenceData],
		AircraftID:              data[AircraftID],
		SSRModeAndCode:          data[Surveillance],
		FlightRulesAndType:      data[Indicator],
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

var categoryRegistry = map[string]CategoryParser{
	CategoryArrival:      arrParser{},
	CategoryDeparture:    depParser{},
	CategoryCancellation: cnlParser{},
	CategoryDelay:        dlaParser{},
	CategoryFlightPlan:   fplParser{},
}

func lookupCategoryParser(category string) (CategoryParser, bool) {
	parser, ok := categoryRegistry[category]
	return parser, ok
}

func buildBodyPatterns() map[string]BodyConfig {
	patterns := make(map[string]BodyConfig, len(categoryRegistry))
	for category, parser := range categoryRegistry {
		patterns[category] = BodyConfig{Patterns: parser.Patterns()}
	}
	return patterns
}

func parseCategory(category string, ctx ParseContext, data map[string]string) (interface{}, error) {
	parser, ok := lookupCategoryParser(category)
	if !ok {
		return nil, fmt.Errorf("invalid message type: %s", category)
	}
	return parser.Parse(ctx, data)
}
