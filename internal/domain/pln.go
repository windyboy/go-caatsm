package domain // Already has package comment

import "fmt"

// ScheduleLine represents a single line or entry in a flight schedule, often from a PLN (Planned Flight) message.
// It details a specific flight's operational data including dates, flight numbers, aircraft registration, and waypoints.
type ScheduleLine struct {
	Index string `json:"index,omitempty"` // Index is an optional identifier for the schedule line.

	// Date of the flight schedule.
	// Example: "30OCT"
	Date string `json:"date"` // Date represents the operational date of the flight.

	// Task or Category of the flight schedule.
	// It could represent different categories based on the airline or flight type.
	// Example: "H/G"
	Task string `json:"task,omitempty"` // Task is an optional category or task identifier for the flight.

	// FlightNumber is a list of flight numbers associated with this schedule entry.
	// Typically one, but can be multiple for code-sharing or multi-segment flights represented as one line.
	// Example: "CA1014"
	FlightNumber []string `json:"flight_number"` // FlightNumber is the list of flight designators.

	// AircraftReg is the registration mark of the aircraft assigned to the flight.
	// Example: "B2458"
	AircraftReg string `json:"aircraft_reg"` // AircraftReg is the aircraft registration.

	// PassengerConfig describes the passenger configuration or seating arrangement.
	// Example: "1/1"
	PassengerConfig string `json:"passenger_config,omitempty"` // PassengerConfig is an optional field for passenger layout.

	// ILS indicates the Instrument Landing System category or configuration.
	// Example: "ILS(0)"
	ILS string `json:"ils,omitempty"` // ILS is an optional field for ILS capabilities.

	Waypoints []WayPoint `json:"waypoints"` // Waypoints is a list of waypoints detailing the flight's route and times.

	// Comments provides any additional remarks or observations about the flight schedule.
	// Example: "Special cargo handling required"
	Comments string `json:"comments,omitempty"` // Comments is an optional field for remarks.

	// Reference provides external identifiers or references related to the flight schedule.
	Reference string `json:"reference,omitempty"` // Reference is an optional field for external references.
}

// WayPoint defines a point in the flight path, typically an airport,
// with associated arrival and departure times.
type WayPoint struct {
	ArrivalTime   string `json:"arrival_time,omitempty"`   // ArrivalTime at the waypoint (typically HHMM format). Optional for the first waypoint.
	Airport       string `json:"airport"`                  // Airport is the ICAO or IATA code of the waypoint/airport.
	DepartureTime string `json:"departure_time,omitempty"` // DepartureTime from the waypoint (typically HHMM format). Optional for the last waypoint.
}

// Validate checks if the mandatory fields of the ScheduleLine are present.
// Currently, it validates Date, FlightNumber (non-empty slice), and AircraftReg.
// It does not validate the contents of Waypoints or individual FlightNumber strings.
func (f *ScheduleLine) Validate() error {
	if f.Date == "" {
		return fmt.Errorf("date is required") // Consider "ScheduleLine.Date: date is required" for consistency if this file is refactored later
	}
	if len(f.FlightNumber) == 0 {
		return fmt.Errorf("flight number is required")
	}
	if f.AircraftReg == "" {
		return fmt.Errorf("aircraft registration is required")
	}
	return nil
}
