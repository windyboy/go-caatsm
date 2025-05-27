package domain // Already has package comment

import "fmt"

/*
起飞报（DEP）报文的规范和组成如下：

编组 3：电报类别、编号和参考数据
编组 7：航空器识别标志和 SSR 模式及编码
编组 13：起飞机场和时间
编组 16：目的地机场和估计总耗时，目的地备降机场
编组 18：其他信息（如需要）
*/

/*
TelegramCategory (电报类别):

Field: TelegramCategory
Description: This field indicates the type of telegram, which in this case would be "DEP" for departure.
AircraftID (航空器识别标志):

Field: AircraftID
Description: This field contains the unique identifier of the aircraft.
SSRModeAndCode (SSR 模式及编码, optional):

Field: SSRModeAndCode
Description: This field contains the SSR (Secondary Surveillance Radar) mode and code. It is optional and indicated as a pointer.
DepartureAirport (起飞机场):

Field: DepartureAirport
Description: This field contains the ICAO code of the airport from which the aircraft is departing.
DepartureTime (起飞时间):

Field: DepartureTime
Description: This field contains the departure time in UTC.
Destination (目的地机场):

Field: Destination
Description: This field contains the ICAO code of the destination airport.
EstimatedElapsedTime (估计总耗时):

Field: EstimatedElapsedTime
Description: This field contains the estimated total elapsed time of the flight.
AlternateAirport (目的地备降机场, optional):

Field: AlternateAirport
Description: This field contains the ICAO code of the alternate destination airport. It is optional and indicated as a pointer.
OtherInfo (其他信息, optional):

Field: OtherInfo
Description: This field contains any additional relevant information. It is optional and indicated as a pointer.
*/
// DEP represents a Departure message (起飞报文).
// It signifies that an aircraft has departed from an airport.
type DEP struct {
	Category             string `json:"category"`                    // Category is the message type, typically "DEP". (电报类别)
	AircraftID           string `json:"aircraft_id"`                 // AircraftID is the unique identifier of the aircraft. (航空器识别标志)
	SSRModeAndCode       string `json:"ssr_mode_and_code,omitempty"` // SSRModeAndCode is the SSR mode and code (optional). (SSR 模式及编码)
	DepartureAirport     string `json:"departure_airport"`           // DepartureAirport is the ICAO code of the departure airport. (起飞机场)
	DepartureTime        string `json:"departure_time"`              // DepartureTime is the actual departure time (typically HHMMSS or similar format). (起飞时间)
	Destination          string `json:"destination"`                 // Destination is the ICAO code of the destination airport. (目的地机场)
	EstimatedElapsedTime string `json:"estimated_elapsed_time"`      // EstimatedElapsedTime is the estimated total flight time. (估计总耗时)
	AlternateAirport     string `json:"alternate_airport,omitempty"` // AlternateAirport is the ICAO code of the alternate destination airport (optional). (目的地备降机场)
	OtherInfo            string `json:"other_info,omitempty"`        // OtherInfo provides a field for any additional relevant information (optional). (其他信息)
}

// Validate checks if the mandatory fields of the DEP message are present.
// Returns an error if any mandatory field is empty, otherwise nil.
func (d *DEP) Validate() error {
	if d.Category == "" {
		return fmt.Errorf("DEP.Category: telegram category is required")
	}
	if d.AircraftID == "" {
		return fmt.Errorf("DEP.AircraftID: aircraft id is required")
	}
	if d.DepartureAirport == "" {
		return fmt.Errorf("DEP.DepartureAirport: departure airport is required")
	}
	if d.DepartureTime == "" {
		return fmt.Errorf("DEP.DepartureTime: departure time is required")
	}
	if d.Destination == "" {
		return fmt.Errorf("DEP.Destination: destination is required")
	}
	if d.EstimatedElapsedTime == "" {
		return fmt.Errorf("DEP.EstimatedElapsedTime: estimated elapsed time is required")
	}
	return nil
}
