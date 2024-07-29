package domain

import "fmt"

type FlightSchedule struct {
	Index string `json:"index,omitempty"`

	// Date of the flight schedule.
	// Example: "30OCT"
	Date string `json:"date"`

	// Category of the flight schedule. It could represent different categories based on the airline or flight type.
	// Example: "H/G"
	Task string `json:"task,omitempty"`

	// Flight number of the flight schedule. This is a unique identifier for the flight.
	// Example: "CA1014"
	FlightNumber string `json:"flight_number"`

	// Aircraft registration number. This is a unique identifier for the aircraft used for the flight.
	// Example: "B2458"
	AircraftReg string `json:"aircraft_reg"`

	// Passenger configuration or seating arrangement for the flight. This field may include details about class configurations.
	// Example: "1/1"
	PassengerConfig string `json:"passenger_config,omitempty"`

	// Instrument Landing System (ILS) category or configuration used for the flight. This field may include ILS category details.
	// Example: "ILS(0)"
	ILS string `json:"ils,omitempty"`

	// ICAO code for the departure airport.
	// Example: "ZBTJ"
	DepartureAirport string `json:"departure_airport"`

	// Scheduled departure time of the flight. This field may be formatted as HHMM or include date information.
	// Example: "1845"
	DepartureTime string `json:"departure_time,omitempty"`

	// Additional schedule information. This field may include special instructions or additional scheduling details.
	// Example: "SI:TSN1845"
	ScheduleInfo string `json:"schedule_info,omitempty"`

	// Estimated or scheduled arrival time of the flight. This field may be formatted as HHMM or include date information.
	// Example: "2100"
	ArrivalTime string `json:"arrival_time,omitempty"`

	// ICAO code for the arrival airport.
	// Example: "ZSPD"
	ArrivalAirport string `json:"arrival_airport"`

	// Additional comments or remarks about the flight schedule. This field may include any relevant notes or observations.
	// Example: "Special cargo handling required"
	Comments string `json:"comments,omitempty"`

	// Reference information or external identifiers related to the flight schedule. This field may include references to external systems or documents.
	Reference string `json:"reference,omitempty"`
}

func (f *FlightSchedule) Validate() error {
	if f.Date == "" {
		return fmt.Errorf("date is required")
	}
	if f.FlightNumber == "" {
		return fmt.Errorf("flight number is required")
	}
	if f.AircraftReg == "" {
		return fmt.Errorf("aircraft registration is required")
	}
	if f.DepartureAirport == "" {
		return fmt.Errorf("departure airport is required")
	}
	if f.ArrivalAirport == "" {
		return fmt.Errorf("arrival airport is required")
	}

	if f.DepartureTime == "" && f.ArrivalTime == "" {
		return fmt.Errorf("either departure time or arrival time is required")
	}
	return nil
}
