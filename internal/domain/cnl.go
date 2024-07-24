package domain

import "fmt"

// CNL represents the structure of a cancellation (CNL) message
type CNL struct {
	Category           string `json:"category"`             // 电报类别 (Message Category)
	AircraftID         string `json:"aircraft_id"`          // 航空器识别标志 (Aircraft Identification)
	DepartureAirport   string `json:"departure_airport"`    // 起飞机场 (Departure Airport)
	DestinationAirport string `json:"destination_airport"`  // 到达机场 (Destination Airport)
	OtherInfo          string `json:"other_info,omitempty"` // 其他信息 (optional) (Other Information)
}

// Validate validates the CNL struct fields
func (c *CNL) Validate() error {
	if c.Category == "" {
		return fmt.Errorf("telegram category is required")
	}
	if c.AircraftID == "" {
		return fmt.Errorf("aircraft id is required")
	}
	if c.DepartureAirport == "" {
		return fmt.Errorf("departure airport is required")
	}
	if c.DestinationAirport == "" {
		return fmt.Errorf("destination airport is required")
	}
	return nil
}
