package weather

import "regexp"

var (
	// Wind patterns: 35012KT, VRB05KT, 27015G25KT, 00000KT
	windPattern = regexp.MustCompile(`^(?P<dir>\d{3}|VRB)(?P<speed>\d{2,3})(G(?P<gust>\d{2,3}))?(?P<unit>KT|MPS)$`)

	// Variable wind: 180V240 (variable from 180 to 240 degrees)
	variableWindPattern = regexp.MustCompile(`^(?P<from>\d{3})V(?P<to>\d{3})$`)

	// Visibility patterns: 9999, 10SM, M1/4SM, 1 1/2SM, 1500
	visibilityPattern = regexp.MustCompile(`^(?P<modifier>[MP\+\-]?)(?P<dist>\d+(?:\s*\d+/\d+)?)(?P<unit>SM|M)?$`)

	// Directional visibility: 2000NE (visibility in a specific direction)
	directionalVisibilityPattern = regexp.MustCompile(`^(?P<dist>\d{4})(?P<dir>[NSEW]{1,2})$`)

	// Cloud patterns: FEW020, SCT030CB, BKN100, OVC200, VV010, SKC, CLR, NSC
	cloudPattern = regexp.MustCompile(`^(?P<type>FEW|SCT|BKN|OVC|VV)(?P<alt>\d{3})(?P<modifier>CB|TCU)?$`)

	// Special cloud codes: SKC (sky clear), CLR (clear), NSC (no significant clouds)
	skyClearPattern = regexp.MustCompile(`^(SKC|CLR|NSC)$`)

	// Temperature/Dewpoint: 25/18, M05/M10, XX/XX
	tempPattern = regexp.MustCompile(`^(?P<temp_mod>M?)(?P<temp>\d{2})/(?P<dew_mod>M?)(?P<dew>\d{2})$`)

	// Altimeter patterns: Q1013 (hPa), A2992 (inHg)
	altimeterPattern = regexp.MustCompile(`^(?P<unit>[QA])(?P<value>\d{4})$`)

	// Weather phenomenon patterns: -RA, +SN, TSRA, FZFG, BR, FG, etc.
	// Intensity: -, +, or empty
	// Descriptors: MI, BC, PR, DR, BL, SH, TS, FZ, DZ, RA, SN, SG, IC, PL, GR, GS, UP, BR, FG, FU, VA, DU, SA, HZ, PY, PO, SQ, FC, SS, DS
	// Weather: DZ, RA, SN, SG, IC, PL, GR, GS, UP, BR, FG, FU, VA, DU, SA, HZ, PY, PO, SQ, FC, SS, DS
	phenomenonPattern = regexp.MustCompile(`^(?P<intensity>[\+\-])?(?P<descriptor>MI|BC|PR|DR|BL|SH|TS|FZ|DZ|RA|SN|SG|IC|PL|GR|GS|UP|BR|FG|FU|VA|DU|SA|HZ|PY|PO|SQ|FC|SS|DS)?(?P<weather>DZ|RA|SN|SG|IC|PL|GR|GS|UP|BR|FG|FU|VA|DU|SA|HZ|PY|PO|SQ|FC|SS|DS)+$`)

	// Time pattern: 251200Z (DDHHmmZ format)
	timePattern = regexp.MustCompile(`^(?P<day>\d{2})(?P<hour>\d{2})(?P<min>\d{2})Z$`)

	// TAF validity period: 2512/2612 (DDHH/DDHH format)
	tafValidityPattern = regexp.MustCompile(`^(?P<from_day>\d{2})(?P<from_hour>\d{2})/(?P<to_day>\d{2})(?P<to_hour>\d{2})$`)

	// TAF period markers: FM251200, TEMPO2512/2515, BECMG2512/2515
	tafFMPattern    = regexp.MustCompile(`^FM(?P<day>\d{2})(?P<hour>\d{2})(?P<min>\d{2})$`)
	tafTEMPOPattern = regexp.MustCompile(`^TEMPO(?P<from_day>\d{2})(?P<from_hour>\d{2})/(?P<to_day>\d{2})(?P<to_hour>\d{2})$`)
	tafBECMGPattern = regexp.MustCompile(`^BECMG(?P<from_day>\d{2})(?P<from_hour>\d{2})/(?P<to_day>\d{2})(?P<to_hour>\d{2})$`)

	// Modifiers: AUTO, COR, NIL, etc.
	modifierPattern = regexp.MustCompile(`^(AUTO|COR|NIL)$`)
)

