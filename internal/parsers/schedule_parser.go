package parsers

import "regexp"

const (
	// DepartureCode   = "dep"
	// DepartureTime   = "dep_time"
	// ArrivalCode     = "arr"
	AirportCode     = "airport"
	WaypointPattern = `(?P<arr_time>\d{4}(\(\d{2}\w{3}\))?)(?P<airport>\w{3})\/?(?P<dep_time>\d{4}(\(\d{2}\w{3}\))?)`
)

var (
	WaypointExpression = regexp.MustCompile(WaypointPattern)
)

func FindWaypoints(message string) map[string]string {
	matches := WaypointExpression.FindStringSubmatch(message)
	if matches == nil {
		return nil
	}

	result := make(map[string]string)
	for i, name := range WaypointExpression.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = matches[i]
		}
	}

	return result
}
