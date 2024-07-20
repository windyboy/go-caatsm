package domain

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
// DEP 电报体中的起飞报文结构
type DEP struct {
	Category             string `json:"category"`                    // 电报类别
	AircraftID           string `json:"aircraft_id"`                 // 航空器识别标志
	SSRModeAndCode       string `json:"ssr_mode_and_code,omitempty"` // SSR 模式及编码（可选）
	DepartureAirport     string `json:"departure_airport"`           // 起飞机场
	DepartureTime        string `json:"departure_time"`              // 起飞时间
	Destination          string `json:"destination"`                 // 目的地机场
	EstimatedElapsedTime string `json:"estimated_elapsed_time"`      // 估计总耗时
	AlternateAirport     string `json:"alternate_airport,omitempty"` // 目的地备降机场（可选）
	OtherInfo            string `json:"other_info,omitempty"`        // 其他信息（可选）
}

// Validate validates the DEP struct fields
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
	if d.EstimatedElapsedTime == "" {
		return fmt.Errorf("estimated elapsed time is required")
	}
	return nil
}
