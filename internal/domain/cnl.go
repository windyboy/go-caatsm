package domain // Already has package comment

import "fmt"

// CNL represents a Cancellation message (取消航班报文).
// It is used to cancel a previously filed flight plan.
type CNL struct {
	Category           string `json:"category"`             // Category is the message type, typically "CNL". (电报类别)
	AircraftID         string `json:"aircraft_id"`          // AircraftID is the unique identifier of the aircraft whose flight plan is being cancelled. (航空器识别标志)
	DepartureAirport   string `json:"departure_airport"`    // DepartureAirport is the ICAO code of the departure airport of the cancelled flight. (起飞机场)
	DestinationAirport string `json:"destination_airport"`  // DestinationAirport is the ICAO code of the destination airport of the cancelled flight. (到达机场)
	OtherInfo          string `json:"other_info,omitempty"` // OtherInfo provides a field for any additional relevant information (optional), such as the reason for cancellation. (其他信息)
}

// Validate checks if the mandatory fields of the CNL message are present.
// Returns an error if any mandatory field is empty, otherwise nil.
func (c *CNL) Validate() error {
	if c.Category == "" {
		return fmt.Errorf("CNL.Category: telegram category is required")
	}
	if c.AircraftID == "" {
		return fmt.Errorf("CNL.AircraftID: aircraft id is required")
	}
	if c.DepartureAirport == "" {
		return fmt.Errorf("CNL.DepartureAirport: departure airport is required")
	}
	if c.DestinationAirport == "" {
		return fmt.Errorf("CNL.DestinationAirport: destination airport is required")
	}
	return nil
}
