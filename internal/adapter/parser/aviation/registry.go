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
	// Validate and extract required fields
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

	return &domain.ARR{
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
	return []PatternConfig{
		{
			Pattern:    DepPatternString,
			Comments:   "Pattern for DEP message",
			Expression: DepPatternExpression,
		},
	}
}

func (depParser) Parse(_ ParseContext, data map[string]string) (interface{}, error) {
	// Validate and extract required fields
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

	return &domain.DEP{
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
	return []PatternConfig{
		{
			Pattern:    CnlPatternString,
			Comments:   "Pattern for CNL message",
			Expression: CnlPatternExpression,
		},
	}
}

func (cnlParser) Parse(_ ParseContext, data map[string]string) (interface{}, error) {
	// Validate and extract required fields
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

	return &domain.CNL{
		Category:           category,
		AircraftID:         aircraftID,
		DepartureAirport:   depAirport,
		DestinationAirport: destAirport,
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
	// Validate and extract required fields
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

	return &domain.DLA{
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
	return []PatternConfig{
		{
			Pattern:    FplPatternString,
			Comments:   "Pattern for FPL message",
			Expression: FplPatternExpression,
		},
	}
}

func (fplParser) Parse(_ ParseContext, data map[string]string) (interface{}, error) {
	// Validate and extract required fields
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

	// Parse optional "other" fields
	otherInfo := GetOptionalField(data, OtherInfo)
	otherData := parseOther(otherInfo)

	return &domain.FPL{
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
		EstimatedArrivalTime:    estTime,
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
