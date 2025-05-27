package domain // Already has package comment

import "fmt"

/*
延误报文（DLA）通常包括以下内容：

    编组 3：电报类别、编号和参考数据
    编组 7：航空器识别标志和 SSR 模式及编码
    编组 13：起飞机场和新的起飞时间
    编组 16：目的地机场和估计总耗时
    编组 18：其他信息（如需要）
*/
/*
MessageType (电报类别):

Field: MessageType
Description: Indicates the type of telegram (e.g., "DLA").
AircraftID (航空器识别标志):

Field: AircraftID
Description: Unique identifier of the aircraft.
SSRModeAndCode (SSR 模式及编码):

Field: SSRModeAndCode
Description: SSR (Secondary Surveillance Radar) mode and code.
DepartureAirport (起飞机场):

Field: DepartureAirport
Description: ICAO code of the departure airport.
NewDepartureTime (新的起飞时间):

Field: NewDepartureTime
Description: New departure time in UTC.
ArrivalAirport (到达机场):

Field: ArrivalAirport
Description: ICAO code of the arrival airport.
EstimatedElapsedTime (估计总耗时):

Field: EstimatedElapsedTime
Description: Estimated total elapsed time.
OtherInfo (其他信息):

Field: OtherInfo
Description: Optional field for any additional relevant information.
*/
/*
(DLA-CCA7890-IS
-B6513
-A1234
-ZBTJ1500
-ZGGG0135
-Weather delay)
*/

// DLA represents a Delay message (延误报文).
// It is used to report a delay in a flight's departure.
type DLA struct {
	Category             string `json:"category"`                           // Category is the message type, typically "DLA". (电报类别)
	AircraftID           string `json:"aircraft_id"`                        // AircraftID is the unique identifier of the aircraft. (航空器识别标志)
	SSRModeAndCode       string `json:"ssr_mode_and_code,omitempty"`        // SSRModeAndCode is the SSR mode and code (optional). (SSR 模式及编码)
	DepartureAirport     string `json:"departure_airport"`                  // DepartureAirport is the ICAO code of the departure airport. (起飞机场)
	NewDepartureTime     string `json:"new_departure_time"`                 // NewDepartureTime is the new estimated departure time (typically HHMMSS or similar format). (新的起飞时间)
	ArrivalAirport       string `json:"arrival_airport"`                    // ArrivalAirport is the ICAO code of the arrival airport. (到达机场)
	EstimatedElapsedTime string `json:"estimated_elapsed_time"`           // EstimatedElapsedTime is the estimated total flight time based on the new departure time. (估计总耗时)
	OtherInfo            string `json:"other_info,omitempty"`               // OtherInfo provides a field for any additional relevant information (optional), such as the reason for the delay. (其他信息)
}

// Validate checks if the mandatory fields of the DLA message are present.
// Returns an error if any mandatory field is empty, otherwise nil.
func (d *DLA) Validate() error {
	if d.Category == "" {
		return fmt.Errorf("DLA.Category: category is required")
	}
	if d.AircraftID == "" {
		return fmt.Errorf("DLA.AircraftID: aircraft id is required")
	}
	if d.DepartureAirport == "" {
		return fmt.Errorf("DLA.DepartureAirport: departure airport is required")
	}
	if d.NewDepartureTime == "" {
		return fmt.Errorf("DLA.NewDepartureTime: new departure time is required")
	}
	if d.ArrivalAirport == "" {
		return fmt.Errorf("DLA.ArrivalAirport: arrival airport is required")
	}
	if d.EstimatedElapsedTime == "" {
		return fmt.Errorf("DLA.EstimatedElapsedTime: estimated elapsed time is required")
	}
	return nil
}
