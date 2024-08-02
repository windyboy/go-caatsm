package parsers

import "regexp"

const (
	CANCELLED   = "CNL"
	AirportCode = "airport"
	Date        = "date"
	Task        = "task"
	// Index               = "idx"
	AllDigitsPattern = `^(?P<dep_time>\d+)$`
	IndexPattern     = `^(?P<idx>\(?L?[0-9]+\)?:?\.?)$`
	DatePattern      = `^(?P<date>\d{2}\w{3})$`
	// CK task: 01)H/Z
	TaskPattern         = `(?P<task>[A-Z]\/[A-Z])$`
	WaypointPattern     = `^(SI:)?(?P<arr_time>\d{4}(\(\d{2}[A-Z]{3}\))?)?\/?(?P<airport>[A-Z]{3})\/?(?P<dep_time>\d{4}(\(\d{2}[A-Z]{3}\))?)?$`
	FlightNumberPattern = `^(?P<number>[0-9A-Z][0-9A-Z]\d{3,5})$`
	RegisterPattern     = `^(?P<reg>B\d{4})$`
)

var (
	AllDigitsExpression = regexp.MustCompile(AllDigitsPattern)

	IndexExpression        = regexp.MustCompile(IndexPattern)
	TaskExpression         = regexp.MustCompile(TaskPattern)
	DateExpression         = regexp.MustCompile(DatePattern)
	WaypointExpression     = regexp.MustCompile(WaypointPattern)
	FlightNumberExpression = regexp.MustCompile(FlightNumberPattern)
	RegisterExpression     = regexp.MustCompile(RegisterPattern)

	parserMap = map[string]*regexp.Regexp{
		Index:        IndexExpression,
		Task:         TaskExpression,
		Date:         DateExpression,
		FlightNumber: FlightNumberExpression,
		Register:     RegisterExpression,
	}

	parserDef = &[]LineParser{
		{
			/**
			* 上航（FM）解析
			* 解析参考
			* W/Z FM9134 B2688 1/1ILS (00) TSN0100 SHA
			* W/Z FM9133 B2688 1/1ILS (00) SHA0340 TSN
			 */
			Airlines:      []string{"FM"},
			MinLen:        6,
			WaypointStart: 5,
			Fields: map[int]string{
				0: Task,
				1: FlightNumber,
				2: Register,
			},
		},
		{
			/**
			* 解析厦航(MF)计划
			* 01) MF8193 B5595 ILS(8) HGH1100 1305TSN
			* 02) MF8194 B5595 ILS(8) TSN1355 1550HGH
			 */
			Airlines:      []string{"MF"},
			MinLen:        5,
			WaypointStart: 4,
			Fields: map[int]string{
				0: Index,
				1: FlightNumber,
				2: Register,
			},
		},
		{
			/**
			* 解析奥凯(8X)计划
			* 计划参考：
			*L1:  29OCT  BK2735 B2863  ILS  IS (3/6)  TSN2350(28OCT)   HAK
			*L2:  29OCT  BK2735 B2863  ILS  IS (3/6)  HAK0435   NKG
			 */
			Airlines:      []string{"8X"},
			MinLen:        9,
			WaypointStart: 7,
			Fields: map[int]string{
				0: Index,
				1: Date,
				2: FlightNumber,
				3: Register,
			},
		},
		{
			/**
			* 解析海航(HU)计划
			* 计划参考
			*L04 W/Z HU7204 B5637 (9) SZX/0500 TSN
			*L05 W/Z HU7205 B5406 (9) TSN/2355(30OCT) PVG
			*1)  JD5195 B6727 ILS I(9) SYX/0800 1135/TSN
			*
			*L07 W/Z GS6571 B3155 (7) XIY/0025 TSN/0245 CGQ
			 */
			Airlines:      []string{"HU"},
			MinLen:        6,
			WaypointStart: 5,
			Fields: map[int]string{
				0: Index,
				1: Task,
				2: FlightNumber,
				3: Register,
			},
		},
		{
			/**
			* 1)  JD5195 B6727 ILS I(9) SYX/0800 1135/TSN
			 */
			Airlines:      []string{"JD"},
			MinLen:        7,
			WaypointStart: 5,
			Fields: map[int]string{
				0: Index,
				1: FlightNumber,
				2: Register,
			},
		},
		{
			/**
			* 01 GS7635 B3193 XIY0020(16APR) CGD
			 */
			Airlines:      []string{"GS"},
			MinLen:        4,
			WaypointStart: 3,
			Fields: map[int]string{
				0: Index,
				1: FlightNumber,
				2: Register,
			},
		},
		{
			/**
			* 杨子快运(Y8)
			*01 Y87969 B2119 XMN 1540 HGH
			*13 Y87444 B2578 ICN 0235 TSN
			 */
			Airlines:      []string{"Y8"},
			MinLen:        6,
			WaypointStart: 3,
			Fields: map[int]string{
				0: Index,
				1: FlightNumber,
				2: Register,
			},
		},
		{
			/**
			* 四川航空 (3U)
			* 01)  31OCT 3U8863 B6598 CAT1 (10) CKG0010 0235TSN
			 */
			Airlines:      []string{"3U"},
			MinLen:        8,
			WaypointStart: 6,
			Fields: map[int]string{
				0: Index,
				1: Date,
				2: FlightNumber,
				3: Register,
			},
		},
		{
			/**
			* 中国货运 (CK)
			*01)H/Z CK261 B2076 PVG1535(30OCT) 1705TPE
			 */
			Airlines:      []string{"CK"},
			MinLen:        4,
			WaypointStart: 3,
			Fields: map[int]string{
				0: Task,
				1: FlightNumber,
				2: Register,
			},
		},
		{
			/**
			* 华夏航空 (G5)
			*L01 W/Z G52665 B7762 (6) CKG/0725 CIH/0940 TSN
			 */
			Airlines:      []string{"G5"},
			MinLen:        8,
			WaypointStart: 5,
			Fields: map[int]string{
				0: Index,
				1: Task,
				2: FlightNumber,
				3: Register,
			},
		},
		{
			/**
			* 春秋航空 (9C)
			*31OCT W/Z 9C8884 B6573 ILS1/1 (06) TSN0650 SYX
			 */
			Airlines:      []string{"9C"},
			MinLen:        8,
			WaypointStart: 6,
			Fields: map[int]string{
				0: Date,
				1: Task,
				2: FlightNumber,
				3: Register,
			},
		},
		{
			/**
			* 深圳航空 (ZH)
			*204) W/Z 31OCT ZH9783 B5670 CAT1 (10) SZX0045 0355TSN
			 */
			Airlines:      []string{"ZH"},
			MinLen:        9,
			WaypointStart: 7,
			Fields: map[int]string{
				0: Index,
				1: Task,
				2: Date,
				3: FlightNumber,
				4: Register,
			},
		},
	}
)

type LineParser struct {
	Airlines      []string
	MinLen        int
	WaypointStart int
	Fields        map[int]string
}
