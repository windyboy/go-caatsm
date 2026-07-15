package weather

import "time"

// Wind represents wind information
type Wind struct {
	Direction     int    `json:"direction"`              // Degrees
	Speed         int    `json:"speed"`                 // KT or MPS
	Gust          int    `json:"gust,omitempty"`         // Gust speed
	Variable      bool   `json:"variable,omitempty"`     // VRB
	VariableFrom  int    `json:"variable_from,omitempty"` // Variable wind from direction
	VariableTo    int    `json:"variable_to,omitempty"`   // Variable wind to direction
	Unit          string `json:"unit"`                  // "KT", "MPS"
}

// Visibility represents visibility information
type Visibility struct {
	Distance  float64 `json:"distance"`            // Meters or statute miles
	Unit      string  `json:"unit"`              // "M", "SM"
	Direction string  `json:"direction,omitempty"` // Directional visibility
	Modifier  string  `json:"modifier,omitempty"`  // +, -, M, P
}

// Cloud represents cloud information
type Cloud struct {
	Type     string `json:"type"`              // FEW, SCT, BKN, OVC, VV
	Altitude int    `json:"altitude"`         // Feet
	Modifier string `json:"modifier,omitempty"` // CB, TCU
}

// Temperature represents temperature or dewpoint
type Temperature struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"` // "C"
}

// Altimeter represents altimeter setting
type Altimeter struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"` // "QNH" (hPa), "A" (inHg)
}

// Phenomenon represents weather phenomenon
type Phenomenon struct {
	Intensity  string `json:"intensity,omitempty"`  // -, +
	Descriptor string `json:"descriptor,omitempty"` // MI, BC, PR, TS, etc.
	Weather    string `json:"weather"`             // RA, SN, FG, etc.
}

// TafPeriod represents a TAF period (FM, TEMPO, BECMG, or main forecast)
type TafPeriod struct {
	Type       string       `json:"type"` // "FM", "TEMPO", "BECMG", "MAIN"
	ValidFrom  time.Time    `json:"valid_from,omitempty"`
	ValidTo    time.Time    `json:"valid_to,omitempty"`
	Wind       *Wind        `json:"wind,omitempty"`
	Visibility *Visibility  `json:"visibility,omitempty"`
	Clouds     []Cloud       `json:"clouds,omitempty"`
	Phenomena  []Phenomenon `json:"phenomena,omitempty"`
	Probability int         `json:"probability,omitempty"` // PROB30, PROB40
}
