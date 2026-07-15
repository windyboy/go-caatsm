package schedule

import "regexp"

// LineParser represents a line parser configuration.
type LineParser struct {
	Airlines      []string
	MinLen        int
	WaypointStart int
	Fields        map[int]string
}

var (
	parserMap = map[string]*regexp.Regexp{}
	parserDef = &[]LineParser{}
)

func init() {
	// Initialize parser map.
	parserMap = map[string]*regexp.Regexp{
		Index:        IndexExpression,
		Task:         TaskExpression,
		Date:         DateExpression,
		FlightNumber: FlightNumberExpression,
		Register:     RegisterExpression,
	}

	// Initialize parser definitions.
	parserDef = &[]LineParser{
		{
			Airlines:      []string{"FM"},
			MinLen:        6,
			WaypointStart: 5,
			Fields: map[int]string{
				0: Task,
				1: FlightNumber,
				2: Register,
			},
		},
		{
			Airlines:      []string{"MF"},
			MinLen:        5,
			WaypointStart: 4,
			Fields: map[int]string{
				0: Index,
				1: FlightNumber,
				2: Register,
			},
		},
		{
			Airlines:      []string{"8X"},
			MinLen:        9,
			WaypointStart: 7,
			Fields: map[int]string{
				0: Index,
				1: Date,
				2: FlightNumber,
				3: Register,
			},
		},
		{
			Airlines:      []string{"HU"},
			MinLen:        6,
			WaypointStart: 5,
			Fields: map[int]string{
				0: Index,
				1: Task,
				2: FlightNumber,
				3: Register,
			},
		},
		{
			Airlines:      []string{"JD"},
			MinLen:        7,
			WaypointStart: 5,
			Fields: map[int]string{
				0: Index,
				1: FlightNumber,
				2: Register,
			},
		},
		{
			Airlines:      []string{"GS"},
			MinLen:        4,
			WaypointStart: 3,
			Fields: map[int]string{
				0: Index,
				1: FlightNumber,
				2: Register,
			},
		},
		{
			Airlines:      []string{"Y8"},
			MinLen:        6,
			WaypointStart: 3,
			Fields: map[int]string{
				0: Index,
				1: FlightNumber,
				2: Register,
			},
		},
		{
			Airlines:      []string{"3U"},
			MinLen:        8,
			WaypointStart: 6,
			Fields: map[int]string{
				0: Index,
				1: Date,
				2: FlightNumber,
				3: Register,
			},
		},
		{
			Airlines:      []string{"CK"},
			MinLen:        4,
			WaypointStart: 3,
			Fields: map[int]string{
				0: Task,
				1: FlightNumber,
				2: Register,
			},
		},
		{
			Airlines:      []string{"G5"},
			MinLen:        8,
			WaypointStart: 5,
			Fields: map[int]string{
				0: Index,
				1: Task,
				2: FlightNumber,
				3: Register,
			},
		},
		{
			Airlines:      []string{"9C"},
			MinLen:        8,
			WaypointStart: 6,
			Fields: map[int]string{
				0: Date,
				1: Task,
				2: FlightNumber,
				3: Register,
			},
		},
		{
			Airlines:      []string{"ZH"},
			MinLen:        9,
			WaypointStart: 7,
			Fields: map[int]string{
				0: Index,
				1: Task,
				2: Date,
				3: FlightNumber,
				4: Register,
			},
		},
		{
			Airlines:      []string{"8L"},
			MinLen:        6,
			WaypointStart: 4,
			Fields: map[int]string{
				0: Index,
				1: Task,
				2: FlightNumber,
				3: Register,
			},
		},
		{
			Airlines:      []string{"SC"},
			MinLen:        9,
			WaypointStart: 7,
			Fields: map[int]string{
				0: Index,
				1: FlightNumber,
				2: Register,
			},
		},
		{
			Airlines:      []string{"PN"},
			MinLen:        7,
			WaypointStart: 5,
			Fields: map[int]string{
				0: Index,
				1: FlightNumber,
				2: Register,
			},
		},
		{
			Airlines:      []string{"CZ"},
			MinLen:        6,
			WaypointStart: 4,
			Fields: map[int]string{
				0: Index,
				1: FlightNumber,
				2: Register,
			},
		},
		{
			Airlines:      []string{"HO"},
			MinLen:        7,
			WaypointStart: 6,
			Fields: map[int]string{
				0: Index,
				1: Date,
				2: Task,
				3: FlightNumber,
				4: Register,
			},
		},
		{
			Airlines:      []string{"NS"},
			MinLen:        7,
			WaypointStart: 6,
			Fields: map[int]string{
				0: Index,
				1: Date,
				2: Task,
				3: FlightNumber,
				4: Register,
			},
		},
		{
			Airlines:      []string{"EU"},
			MinLen:        7,
			WaypointStart: 6,
			Fields: map[int]string{
				0: Task,
				1: Date,
				2: FlightNumber,
				3: Register,
			},
		},
	}
}
