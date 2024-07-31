package domain

import "fmt"

type ScheduleLine struct {
	Index string `json:"index,omitempty"`

	// Date of the flight schedule.
	// Example: "30OCT"
	Date string `json:"date"`

	// Category of the flight schedule. It could represent different categories based on the airline or flight type.
	// Example: "H/G"
	Task string `json:"task,omitempty"`

	// Flight number of the flight schedule. This is a unique identifier for the flight.
	// Example: "CA1014"
	FlightNumber []string `json:"flight_number"`

	// Aircraft registration number. This is a unique identifier for the aircraft used for the flight.
	// Example: "B2458"
	AircraftReg string `json:"aircraft_reg"`

	// Passenger configuration or seating arrangement for the flight. This field may include details about class configurations.
	// Example: "1/1"
	PassengerConfig string `json:"passenger_config,omitempty"`

	// Instrument Landing System (ILS) category or configuration used for the flight. This field may include ILS category details.
	// Example: "ILS(0)"
	ILS string `json:"ils,omitempty"`

	Waypoints []WayPoint `json:"waypoints"`

	// Additional comments or remarks about the flight schedule. This field may include any relevant notes or observations.
	// Example: "Special cargo handling required"
	Comments string `json:"comments,omitempty"`

	// Reference information or external identifiers related to the flight schedule. This field may include references to external systems or documents.
	Reference string `json:"reference,omitempty"`
}

type WayPoint struct {
	ArrivalTime   string
	Airport       string
	DepartureTime string
}

func (f *ScheduleLine) Validate() error {
	if f.Date == "" {
		return fmt.Errorf("date is required")
	}
	if len(f.FlightNumber) == 0 {
		return fmt.Errorf("flight number is required")
	}
	if f.AircraftReg == "" {
		return fmt.Errorf("aircraft registration is required")
	}
	return nil
}
