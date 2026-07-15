package aviation

import "regexp"

// String constants
const (
	StartIndicatorPrefix = "ZCZC"
	EndHeaderMarker      = "."
	BeginPartMarker      = "BEGIN PART"

	Category             = "category"
	CategoryArrival      = "ARR"
	CategoryDeparture    = "DEP"
	CategoryCancellation = "CNL"
	CategoryDelay        = "DLA"
	CategoryFlightPlan   = "FPL"

	FlightNumber = "number"
	Register     = "reg"

	SSR             = "ssr"
	DepartureCode   = "dep"
	DepartureTime   = "dep_time"
	ArrivalCode     = "arr"
	ArrivalTime     = "arr_time"
	DestinationCode = "dest"
	OtherInfo       = "other"

	ReferenceData        = "reference_data"
	CategorySurveillance = "surve"
	Indicator            = "indicator"
	Other                = "other"
	AircraftID           = "aircraft"
	Surveillance         = "surve"
	Speed                = "speed"
	Level                = "level"
	Route                = "route"
	EstimatedTime        = "estt"
	AlternateAirport     = "alter"
	PBN                  = "pbn"
	NavigationEquipment  = "nav"
	EstimatedElapsedTime = "eet"
	SELCALCode           = "sel"
	PerformanceCategory  = "per"
	RerouteInformation   = "rif"
	Remarks              = "remark"
)

// Parser configuration constants
const (
	// MinHeaderLines is the minimum number of lines required for a valid telegram header
	MinHeaderLines = 3

	// MinStartIndicatorParts is the minimum number of parts in the start indicator line (ZCZC MessageID DateTime)
	MinStartIndicatorParts = 3

	// MinPriorityLineParts is the minimum number of parts in the priority line (Priority PrimaryAddress)
	MinPriorityLineParts = 2

	// MinOriginatorParts is the minimum number of parts in an originator line (.CODE DATETIME)
	MinOriginatorParts = 2

	// MinOriginatorMatchGroups is the minimum number of regex match groups for originator pattern
	MinOriginatorMatchGroups = 3
)

// Regular expression patterns for ICAO telegram body parsing.
// These patterns match specific message types defined in ICAO standards.
const (
	// ArrPatternString matches ARR (Arrival) messages.
	// Format: (ARR-FLIGHTNUM[/SSR]-DEPICAO-ARRICAOTIME)
	// Example: (ARR-CES5470/A1234-ZBTJ-ZSHC1614)
	// Capture groups:
	//   - category: Message type (ARR)
	//   - number: Flight number (alphanumeric, e.g., CES5470)
	//   - ssr: SSR mode and code (optional, after /, e.g., A1234)
	//   - dep: Departure airport (4-letter ICAO code, e.g., ZBTJ)
	//   - arr: Arrival airport (4-letter ICAO code, e.g., ZSHC)
	//   - arr_time: Arrival time (4 digits HHMM, e.g., 1614)
	ArrPatternString = `^\((?P<category>[A-Z]{3})-(?P<number>[A-Z0-9]+)(\/?(?P<ssr>[A-Z0-9]+))?-(?P<dep>[A-Z]{4})-(?P<arr>[A-Z]{4})(?P<arr_time>\d{4})\)$`

	// DepPatternString matches DEP (Departure) messages.
	// Format: (DEP-FLIGHTNUM[/SSR]-DEPICAOTIME-ARRICAO)
	// Example: (DEP-CYZ9017/A5633-ZBTJ1638-ZSPD)
	// Capture groups:
	//   - category: Message type (DEP)
	//   - number: Flight number (alphanumeric, e.g., CYZ9017)
	//   - ssr: SSR mode and code (optional, after /, e.g., A5633)
	//   - dep: Departure airport (4-letter ICAO code, e.g., ZBTJ)
	//   - dep_time: Departure time (4 digits HHMM, e.g., 1638)
	//   - arr: Destination airport (4-letter ICAO code, e.g., ZSPD)
	DepPatternString = `^\((?P<category>[A-Z]{3})-(?P<number>[A-Z0-9]+)(\/(?P<ssr>[A-Z0-9]+))?-(?P<dep>[A-Z]{4})(?P<dep_time>\d{4})-(?P<arr>[A-Z]{4})\)$`

	// FplPatternString matches FPL (Flight Plan) messages.
	// This is the most complex pattern, matching ICAO Doc 4444 Field Type 15 format.
	// Format spans multiple lines with specific field ordering per ICAO standards.
	// Example: (FPL-CCA1532-IS\n-A332/H\n-SDE3FGHIJ4J5M1RWY/LB101\n-ZSSS2035\n-K0859S1040 PIAKS G330...\n-ZBAA0153 ZBYN\n-PBN/A1B2... RMK/TCAS EQUIPPED)
	// Capture groups:
	//   - category: Message type (FPL)
	//   - number: Flight number (e.g., CCA1532)
	//   - indicator: Flight rules and type (2 letters, e.g., IS)
	//   - aircraft: Aircraft type and wake turbulence (e.g., A332/H)
	//   - surve: Surveillance equipment codes
	//   - dep: Departure airport (4-letter ICAO)
	//   - dep_time: Departure time (4 digits HHMM)
	//   - speed: Cruising speed (e.g., K0859)
	//   - level: Flight level (e.g., S1040)
	//   - route: Flight route (can span multiple lines)
	//   - dest: Destination airport (4-letter ICAO)
	//   - estt: Estimated elapsed time (4 digits)
	//   - alter: Alternate airports (space-separated ICAO codes)
	//   - other: Other information fields (PBN, NAV, REG, EET, SEL, PER, RIF, RMK)
	FplPatternString = `\((?P<category>[A-Z]{3})-(?P<number>[A-Z]+\d+)-(?P<indicator>[A-Z]{2})\n-(?P<aircraft>[A-Z]+\d+\/?[A-Z]?)\n?-(?P<surve>.*)\n?-(?P<dep>[A-Z]{4})(?P<dep_time>\d{4})\n?-(?P<speed>[A-Z]+\d+)(?P<level>[A-Z0-9]+)\s+(?P<route>(.|\n)+)\n-(?P<dest>[A-Z]{4})(?P<estt>\d{4})\s?(?P<alter>(\s[A-Z]{4})+)\n?-([A-Z]{3}\/(?:[A-Z]{4}\d{4}\s?)+)?(?P<other>(?m)[A-Z]{3}\/(.|\n)*)\)$`

	// CnlPatternString matches CNL (Cancellation) messages.
	// Format: (CNL-FLIGHTNUM-[DEPICAO]-ARRICAO)
	// Example: (CNL-YZR7979-ZSPD-ZBTJ)
	// Capture groups:
	//   - category: Message type (CNL)
	//   - number: Flight number (alphanumeric, e.g., YZR7979)
	//   - dep: Departure airport (4-letter ICAO, optional)
	//   - arr: Destination airport (4-letter ICAO)
	CnlPatternString = `^\((?P<category>[A-Z]{3})-(?P<number>\w+\d+)-?(?P<dep>[A-Z]{4})?-?(?<arr>[A-Z]{4})\)$`

	// DlaPatternString matches DLA (Delay) messages.
	// Format: (DLA-FLIGHTNUM-DEPICAO[TIME]-ARRICAO[TIME])
	// Example: (DLA-CSN3133-ZGGG0110-ZBTJ)
	// Capture groups:
	//   - category: Message type (DLA)
	//   - number: Flight number (alphanumeric, e.g., CSN3133)
	//   - dep: Departure airport (4-letter ICAO)
	//   - dep_time: New departure time (4 digits HHMM, optional)
	//   - arr: Arrival airport (4-letter ICAO)
	//   - arr_time: Arrival time (4 digits HHMM, optional)
	DlaPatternString = `^\((?P<category>[A-Z]{3})-(?P<number>\w+\d+)-?(?P<dep>[A-Z]{4})(?P<dep_time>\d{4})?-?(?<arr>[A-Z]{4})(?<arr_time>\d{4})?\)$`
)

// Compiled regular expressions
var (
	ArrPatternExpression = regexp.MustCompile(ArrPatternString)
	DepPatternExpression = regexp.MustCompile(DepPatternString)
	FplPatternExpression = regexp.MustCompile(FplPatternString)
	CnlPatternExpression = regexp.MustCompile(CnlPatternString)
	DlaPatternExpression = regexp.MustCompile(DlaPatternString)
	BodyTypePattern      = regexp.MustCompile(`^\(([A-Z]{3})(.*\n?)+\)$`)

	categoryRegex      = regexp.MustCompile(`\((?P<category>[A-Z]+)-`)
	emptyLineRemove    = regexp.MustCompile(`(?m)^\s*$`)
	bodyOnly           = regexp.MustCompile(`(.|\n)?(ZCZC(.|\n)*)NNNN(.|\n)?$`)
	originator         = regexp.MustCompile(`(?P<originatorDateTime>[0-9]+)\s(?P<originator>[A-Z]+)`)
	navPattern         = regexp.MustCompile(`(?m)NAV\/(?P<nav>\w+)`)
	remarkPattern      = regexp.MustCompile(`(?s)RMK\/(?P<remark>.*)`)
	selPattern         = regexp.MustCompile(`(?m)SEL\/(?P<sel>\w+)`)
	regPattern         = regexp.MustCompile(`(?m)REG\/(?P<reg>[A-Z0-9]+)`)
	pbnPattern         = regexp.MustCompile(`(?m)PBN\/(?P<pbn>[A-Z0-9]+)`)
	eetPattern         = regexp.MustCompile(`(?s)(-?EET\/(?P<eet>(?:[A-Z]{4}\d{4}\s*)+))`)
	performancePattern = regexp.MustCompile(`(?s)-?PER\/(?P<per>\w)`)
	reroutePattern     = regexp.MustCompile(`(?m)RIF\/(?P<rif>.*)[A-Z]{3}\/`)
)
