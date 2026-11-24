package aviation

import "fmt"

// Body represents a parsed aviation telegram body payload capable of self-validation.
type Body interface {
	Validate() error
}

// ARR represents an Arrival (ARR) telegram body.
type ARR struct {
	Category             string `json:"category"`
	AircraftID           string `json:"aircraft_id"`
	SSRModeAndCode       string `json:"ssr_mode_and_code,omitempty"`
	DepartureAirport     string `json:"departure_airport"`
	DepartureTime        string `json:"departure_time,omitempty"`
	ArrivalAirport       string `json:"arrival_airport"`
	ArrivalTime          string `json:"arrival_time"`
	EstimatedElapsedTime string `json:"estimated_elapsed_time,omitempty"`
	AlternateAirport     string `json:"alternate_airport,omitempty"`
	OtherInfo            string `json:"other_info,omitempty"`
}

// Validate checks required ARR fields per ICAO Doc 4444.
func (a *ARR) Validate() error {
	if a.Category == "" {
		return fmt.Errorf("category is required")
	}
	if a.AircraftID == "" {
		return fmt.Errorf("aircraft id is required")
	}
	if a.DepartureAirport == "" {
		return fmt.Errorf("departure airport is required")
	}
	if a.ArrivalAirport == "" {
		return fmt.Errorf("arrival airport is required")
	}
	if a.ArrivalTime == "" {
		return fmt.Errorf("arrival time is required")
	}
	return nil
}

// DEP represents a Departure (DEP) telegram body.
type DEP struct {
	Category             string `json:"category"`
	AircraftID           string `json:"aircraft_id"`
	SSRModeAndCode       string `json:"ssr_mode_and_code,omitempty"`
	DepartureAirport     string `json:"departure_airport"`
	DepartureTime        string `json:"departure_time"`
	Destination          string `json:"destination"`
	EstimatedElapsedTime string `json:"estimated_elapsed_time,omitempty"`
	AlternateAirport     string `json:"alternate_airport,omitempty"`
	OtherInfo            string `json:"other_info,omitempty"`
}

// Validate checks required DEP fields per ICAO Doc 4444.
func (d *DEP) Validate() error {
	if d.Category == "" {
		return fmt.Errorf("telegram category is required")
	}
	if d.AircraftID == "" {
		return fmt.Errorf("aircraft id is required")
	}
	if d.DepartureAirport == "" {
		return fmt.Errorf("departure airport is required")
	}
	if d.DepartureTime == "" {
		return fmt.Errorf("departure time is required")
	}
	if d.Destination == "" {
		return fmt.Errorf("destination is required")
	}
	return nil
}

// CNL represents a Flight Plan Cancellation (CNL) telegram body.
type CNL struct {
	Category           string `json:"category"`
	AircraftID         string `json:"aircraft_id"`
	DepartureAirport   string `json:"departure_airport"`
	DestinationAirport string `json:"destination_airport"`
	OtherInfo          string `json:"other_info,omitempty"`
}

// Validate checks required CNL fields.
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

// DLA represents a Flight Delay (DLA) telegram body.
type DLA struct {
	Category         string `json:"category"`
	AircraftID       string `json:"aircraft_id"`
	SSRModeAndCode   string `json:"ssr_mode_and_code,omitempty"`
	DepartureAirport string `json:"departure_airport"`
	NewDepartureTime string `json:"new_departure_time,omitempty"`
	ArrivalAirport   string `json:"arrival_airport"`
	ArrivalTime      string `json:"arrival_time,omitempty"`
	OtherInfo        string `json:"other_info,omitempty"`
}

// Validate checks required DLA fields.
func (d *DLA) Validate() error {
	if d.Category == "" {
		return fmt.Errorf("telegram category is required")
	}
	if d.AircraftID == "" {
		return fmt.Errorf("aircraft id is required")
	}
	if d.DepartureAirport == "" {
		return fmt.Errorf("departure airport is required")
	}
	if d.ArrivalAirport == "" {
		return fmt.Errorf("arrival airport is required")
	}
	return nil
}

// FPL represents a Filed Flight Plan (FPL) telegram body.
type FPL struct {
	Category                string `json:"category"`
	FlightNumber            string `json:"flight_number"`
	ReferenceData           string `json:"reference_data,omitempty"`
	AircraftID              string `json:"aircraft_id"`
	SSRModeAndCode          string `json:"ssr_mode_and_code,omitempty"`
	FlightRulesAndType      string `json:"flight_rules_and_type"`
	CruisingSpeedAndLevel   string `json:"cruising_speed_and_level"`
	DepartureAirport        string `json:"departure_airport"`
	DepartureTime           string `json:"departure_time"`
	Route                   string `json:"route"`
	DestinationAndTotalTime string `json:"destination_and_total_time"`
	AlternateAirport        string `json:"alternate_airport,omitempty"`
	OtherInfo               string `json:"other_info,omitempty"`
	SupplementaryInfo       string `json:"supplementary_info,omitempty"`
	// TotalEET holds ICAO Item 16 total estimated elapsed time (HHMM).
	// The JSON key remains "estimated_arrival_time" for backward compatibility with downstream consumers.
	TotalEET             string `json:"estimated_arrival_time,omitempty"`
	PBN                  string `json:"pbn,omitempty"`
	NavigationEquipment  string `json:"navigation_equipment,omitempty"`
	EstimatedElapsedTime string `json:"estimated_elapsed_time,omitempty"`
	SELCALCode           string `json:"selcal_code,omitempty"`
	Register             string `json:"register,omitempty"`
	PerformanceCategory  string `json:"performance_category,omitempty"`
	RerouteInformation   string `json:"reroute_information,omitempty"`
	Remarks              string `json:"remarks,omitempty"`
}

// Validate checks required FPL fields (Item 18 fields are optional per ICAO Doc 4444).
func (f *FPL) Validate() error {
	if f.Category == "" {
		return fmt.Errorf("category is required")
	}
	if f.FlightNumber == "" {
		return fmt.Errorf("flight number is required")
	}
	if f.AircraftID == "" {
		return fmt.Errorf("aircraft id is required")
	}
	if f.FlightRulesAndType == "" {
		return fmt.Errorf("flight rules and type is required")
	}
	if f.CruisingSpeedAndLevel == "" {
		return fmt.Errorf("cruising speed and level is required")
	}
	if f.DepartureAirport == "" {
		return fmt.Errorf("departure airport is required")
	}
	if f.DepartureTime == "" {
		return fmt.Errorf("departure time is required")
	}
	if f.Route == "" {
		return fmt.Errorf("route is required")
	}
	if f.DestinationAndTotalTime == "" {
		return fmt.Errorf("destination and total time is required")
	}
	return nil
}
