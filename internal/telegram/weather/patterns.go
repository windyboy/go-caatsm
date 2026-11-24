package weather

import "regexp"

var (
	windPattern                  = regexp.MustCompile(`^(?P<dir>\d{3}|VRB)(?P<speed>\d{2,3})(G(?P<gust>\d{2,3}))?(?P<unit>KT|MPS)$`)
	variableWindPattern          = regexp.MustCompile(`^(?P<from>\d{3})V(?P<to>\d{3})$`)
	visibilityPattern            = regexp.MustCompile(`^(?P<modifier>[MP\+\-]?)(?P<dist>\d+/\d+|\d+)(?P<unit>SM|M)?$`)
	mixedVisWholePattern         = regexp.MustCompile(`^[1-9]\d?$`)
	mixedVisFracPattern          = regexp.MustCompile(`^(?P<frac>\d+/\d+)(?P<unit>SM)$`)
	directionalVisibilityPattern = regexp.MustCompile(`^(?P<dist>\d{4})(?P<dir>[NSEW]{1,2})$`)
	rvrPattern                   = regexp.MustCompile(`^R\d{2}[LRC]?/`)
	cloudPattern                 = regexp.MustCompile(`^(?P<type>FEW|SCT|BKN|OVC|VV)(?P<alt>\d{3})(?P<modifier>CB|TCU)?$`)
	skyClearPattern              = regexp.MustCompile(`^(SKC|CLR|NSC)$`)
	tempPattern                  = regexp.MustCompile(`^(?P<temp_mod>M?)(?P<temp>\d{2})/(?P<dew_mod>M?)(?P<dew>\d{2})$`)
	altimeterPattern             = regexp.MustCompile(`^(?P<unit>[QA])(?P<value>\d{4})$`)
	phenomenonPattern            = regexp.MustCompile(`^(?P<intensity>[\+\-]|VC)?(?P<descriptor>MI|BC|PR|DR|BL|SH|TS|FZ)?(?P<weather>(?:DZ|RA|SN|SG|IC|PL|GR|GS|UP|BR|FG|FU|VA|DU|SA|HZ|PY|PO|SQ|FC|SS|DS)+)$`)
	vicinityDescPattern          = regexp.MustCompile(`^(?P<intensity>VC)(?P<descriptor>SH|TS)$`)
	timePattern                  = regexp.MustCompile(`^(?P<day>\d{2})(?P<hour>\d{2})(?P<min>\d{2})Z$`)
	tafValidityPattern           = regexp.MustCompile(`^(?P<from_day>\d{2})(?P<from_hour>\d{2})/(?P<to_day>\d{2})(?P<to_hour>\d{2})$`)
	tafFMPattern                 = regexp.MustCompile(`^FM(?P<day>\d{2})(?P<hour>\d{2})(?P<min>\d{2})$`)
	tafTEMPOPattern              = regexp.MustCompile(`^TEMPO(?P<from_day>\d{2})(?P<from_hour>\d{2})/(?P<to_day>\d{2})(?P<to_hour>\d{2})$`)
	tafBECMGPattern              = regexp.MustCompile(`^BECMG(?P<from_day>\d{2})(?P<from_hour>\d{2})/(?P<to_day>\d{2})(?P<to_hour>\d{2})$`)
	modifierPattern              = regexp.MustCompile(`^(AUTO|COR|NIL)$`)
)
