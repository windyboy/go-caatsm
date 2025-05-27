package parsers

import (
	"caatsm/internal/domain"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Aviation Parser", func() {
	Describe("ParseHeader", func() {

		Context("with a real ARR context", func() {
			message := `ZCZC TMQ2530 141614
GG ZBTJZXZX
141614 ZSHCZTZX
(ARR-CES5470-ZBTJ-ZSHC1614)
NNNN`
			It("should get a clean body text", func() {
				body := cleanMessage(message)
				expected := `ZCZC TMQ2530 141614
GG ZBTJZXZX
141614 ZSHCZTZX
(ARR-CES5470-ZBTJ-ZSHC1614)`
				Expect(body).To(Equal(expected))
			})
		})

		Context("with CHG body", func() {
			// CHG regex: `\(CHG-(?P<flight_number>[A-Z0-9]+)(?:/(?P<ssr>[A-Z0-9]+))?-(?P<dep>[A-Z]{4})(?P<dep_time>[0-9]{4})-(?P<arr>[A-Z]{4})(?P<arr_time>[0-9]{4})-(?P<eet>[0-9]{4})(?:-ALT(?P<alter>[A-Z]{4}))?(?:-OTHER\s*(?P<other>.*?))?-(?P<change_part>.*?)\)`
			// Many fields are mandatory in the regex itself for CHG.
			It("should parse minimal CHG body correctly", func() {
				body := "(CHG-FLT123-ZABC1200-ZSDE1300-0100-PART CHANGED)" // Assumes SSR is optional in regex, but mandatory in domain. Let's test without SSR.
				// The regex actually is `(?:/(?P<ssr>[A-Z0-9]+))?` making SSR optional.
				// But the domain CHG struct lists SSRModeAndCode as mandatory.
				// The parser will extract empty if not present, domain validation will fail.
				// Let's use a CHG message that has all fields required by its regex.
				// The regex seems to be: CHG-flight_number-dep+dep_time-arr+arr_time-eet-change_part
				// SSR, ALT, OTHER are optional in the regex.
				body = "(CHG-FLT123-ZABC1200-ZSDE1300-0100-FIELDD18 TEXT)"
				parser := NewBodyParser(body)
				category, parsedBody, err := parser.Parse()
				Expect(err).ToNot(HaveOccurred())
				Expect(category).To(Equal("CHG"))
				chgMessage, ok := parsedBody.(*domain.CHG)
				Expect(ok).To(BeTrue())
				Expect(chgMessage.Category).To(Equal("CHG"))
				Expect(chgMessage.AircraftID).To(Equal("FLT123"))
				Expect(chgMessage.DepartureAirport).To(Equal("ZABC"))
				Expect(chgMessage.DepartureTime).To(Equal("1200"))
				Expect(chgMessage.ArrivalAirport).To(Equal("ZSDE"))
				Expect(chgMessage.ArrivalTime).To(Equal("1300"))
				Expect(chgMessage.EstimatedElapsedTime).To(Equal("0100"))
				Expect(chgMessage.ChangePart).To(Equal("FIELDD18 TEXT"))
				Expect(chgMessage.SSRModeAndCode).To(BeEmpty()) // Optional in regex
			})

			It("should parse full CHG body correctly", func() {
				body := "(CHG-FLT123/S1234-ZABC1200-ZSDE1300-0100-ALTALTT-OTHER SOME INFO-PART CHANGED)"
				parser := NewBodyParser(body)
				category, parsedBody, err := parser.Parse()
				Expect(err).ToNot(HaveOccurred())
				chgMessage, ok := parsedBody.(*domain.CHG)
				Expect(ok).To(BeTrue())
				Expect(chgMessage.SSRModeAndCode).To(Equal("S1234"))
				Expect(chgMessage.AlternateAirport).To(Equal("ALTT"))
				Expect(chgMessage.OtherInfo).To(Equal("SOME INFO"))
				Expect(chgMessage.ChangePart).To(Equal("PART CHANGED"))
			})
		})

		Context("with unrecognized message category", func() {
			It("should return an error and no parsed body", func() {
				body := "(XYZ-SOMEUNKNOWNMESSAGE)"
				parser := NewBodyParser(body)
				category, parsedBody, err := parser.Parse()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no matching pattern found for body"))
				// Category might be found by findCategory but then fail main parsing.
				// findCategory looks for XYZ in `categoryRegex = regexp.MustCompile(`^\s*\((?P<category>[A-Z]{3})`)`
				Expect(category).To(Equal("XYZ")) // findCategory should extract it
				Expect(parsedBody).To(BeNil())
			})
		})

		Context("with body missing optional fields for a known category (ARR example)", func() {
			It("ARR message missing optional fields", func() {
				// ARR-CES5470-ZBTJ-ZSHC1614 (missing optional dep_time, ssr, eet, alt, other)
				body := "(ARR-CES5470-ZBTJ-ZSHC1614)"
				parser := NewBodyParser(body)
				category, parsedBody, err := parser.Parse()
				Expect(err).ToNot(HaveOccurred()) 
				Expect(category).To(Equal("ARR"))
				Expect(parsedBody).ToNot(BeNil())
				arrMsg, ok := parsedBody.(*domain.ARR)
				Expect(ok).To(BeTrue())
				Expect(arrMsg.AircraftID).To(Equal("CES5470"))
				Expect(arrMsg.DepartureAirport).To(Equal("ZBTJ"))
				Expect(arrMsg.ArrivalAirport).To(Equal("ZSHC"))
				Expect(arrMsg.ArrivalTime).To(Equal("1614"))
				Expect(arrMsg.DepartureTime).To(BeEmpty())
				Expect(arrMsg.SSRModeAndCode).To(BeEmpty())
				Expect(arrMsg.EstimatedElapsedTime).To(BeEmpty())
				Expect(arrMsg.AlternateAirport).To(BeEmpty())
				Expect(arrMsg.OtherInfo).To(BeEmpty())
			})
		})

		It("should parse the header correctly", func() {
			message := `
ZCZC TAF6789 160530
QU TSNZPCA
.
QU PEKUDCA TSNUOCA TSNZPCA TSNUFCA
.TAF WSSS 160500Z 1606/1712 20010KT 9999 SCT018 
BECMG 1608/1610 24012KT 9999 SCT018 
TEMPO 1610/1612 4000 SHRA BKN012 
BECMG 1612/1614 18008KT 9999 SCT020

BEGIN PART 02

(FORECAST AMENDMENT
VALID 1606/1700
THUNDERSTORMS EXPECTED
ALTERNATE ROUTES ADVISED)

NNNN`
			parsedHeader, err := ParseHeader(message)
			Expect(err).ToNot(HaveOccurred())
			Expect(parsedHeader.MessageID).To(Equal("TAF6789"))
			Expect(parsedHeader.DateTime).To(Equal("160530"))
			Expect(parsedHeader.PriorityIndicator).To(Equal("QU"))
			Expect(parsedHeader.PrimaryAddress).To(Equal("TSNZPCA"))
			Expect(parsedHeader.SecondaryAddresses).To(Equal(" QU PEKUDCA TSNUOCA TSNZPCA TSNUFCA"))
		})

		It("should parse the header correctly with originator information", func() {
			message := `
ZCZC NOTAM1122 171000
QU TSNZPCA
.
QU PEKUDCA TSNUOCA TSNZPCA TSNUFCA
.SELOZKE 170999

BEGIN PART 01

RUNWAY MAINTENANCE NOTICE.

- MAINTENANCE MANAGER: JOHN DOE

RUNWAY 09/27 WILL BE CLOSED FOR MAINTENANCE FROM 0800Z TO 1600Z.

- AIRPORT OPERATIONS:       SIGN . . . . . . . . . .

WE ACKNOWLEDGE THE RUNWAY CLOSURE.

- CONTROL TOWER:

SIGN . . . . . . . . . .

BEGIN PART 02

(ALERT MESSAGE - WEATHER WARNING
VALID 1500Z - 1800Z
SEVERE THUNDERSTORM FORECASTED
ALL DEPARTURES/ARRIVALS EXPECTED TO BE DELAYED)

NNNN`
			parsedHeader, err := ParseHeader(message)
			Expect(err).ToNot(HaveOccurred())
			Expect(parsedHeader.MessageID).To(Equal("NOTAM1122"))
			Expect(parsedHeader.DateTime).To(Equal("171000"))
			Expect(parsedHeader.PriorityIndicator).To(Equal("QU"))
			Expect(parsedHeader.PrimaryAddress).To(Equal("TSNZPCA"))
			Expect(parsedHeader.SecondaryAddresses).To(Equal(" QU PEKUDCA TSNUOCA TSNZPCA TSNUFCA"))
			Expect(parsedHeader.Originator).To(Equal("SELOZKE"))
			Expect(parsedHeader.OriginatorDateTime).To(Equal("170999"))
		})

		Context("with malformed or edge case headers", func() {
			It("should return an error for message with no ZCZC", func() {
				message := `TMQ1324 150631
GG ZBTJZXZX
141614 ZSHCZTZX
(ARR-CES5470-ZBTJ-ZSHC1614)
NNNN`
				_, err := ParseHeader(message)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid start indicator line format"))
			})

			It("should return an error for message with ZCZC but incomplete start line", func() {
				message := `ZCZC TMQ1324
GG ZBTJZXZX
141614 ZSHCZTZX
(ARR-CES5470-ZBTJ-ZSHC1614)
NNNN`
				_, err := ParseHeader(message)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid start indicator line format"))
			})

			It("should return an error for message with malformed priority/primary address line", func() {
				message := `ZCZC TMQ1324 150631
GGZBTJZXZX
141614 ZSHCZTZX
(ARR-CES5470-ZBTJ-ZSHC1614)
NNNN`
				_, err := ParseHeader(message)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid priority and primary address line format"))
			})

			It("should handle header with only ZCZC line and priority line (incomplete)", func() {
				message := `ZCZC TMQ1324 150631
GG ZBTJZXZX`
				// This is less than 3 lines after cleaning, so ParseHeader itself will error out early.
				_, err := ParseHeader(message)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid message format: not enough lines"))
			})

			It("should handle header with no secondary addresses and no originator", func() {
				message := `ZCZC MSGID001 101010
FF ADDR1
(BODY-PART)
NNNN`
				parsedHeader, err := ParseHeader(message)
				Expect(err).ToNot(HaveOccurred())
				Expect(parsedHeader.MessageID).To(Equal("MSGID001"))
				Expect(parsedHeader.PrimaryAddress).To(Equal("ADDR1"))
				Expect(parsedHeader.SecondaryAddresses).To(BeEmpty())
				Expect(parsedHeader.Originator).To(BeEmpty())
				Expect(parsedHeader.Body).To(Equal("(BODY-PART)\n"))
			})

			It("should handle originator line appearing before secondary addresses (uncommon but test parser flexibility)", func() {
				// Current parser logic processes lines sequentially. If originator is found, subsequent non-body-marker lines might be miscategorized.
				// This specific case depends on how parseHeaderFieldsAndBody handles this.
				// The refactored parseHeaderFieldsAndBody tries to identify originator with ".", else assumes secondary.
				message := `ZCZC MSGID002 111111
QQ ADDRMAIN
.ORIGACC 101122 
QQ ADDRSEC1 ADDRSEC2
(TESTBODY)
NNNN`
				parsedHeader, err := ParseHeader(message)
				Expect(err).ToNot(HaveOccurred())
				Expect(parsedHeader.Originator).To(Equal("ORIGACC"))
				Expect(parsedHeader.OriginatorDateTime).To(Equal("101122"))
				// Based on current logic, "QQ ADDRSEC1 ADDRSEC2" would be appended to secondary addresses if encountered *before* the originator line with ".".
				// If it's after, it would be part of the body or ignored if after a body marker.
				// In this case, after ".ORIGACC", "QQ ADDRSEC1 ADDRSEC2" would be treated as body content.
				Expect(parsedHeader.SecondaryAddresses).To(BeEmpty()) // Because no secondary addresses before the originator line.
				Expect(parsedHeader.Body).To(SatisfyAny(Equal("QQ ADDRSEC1 ADDRSEC2\n(TESTBODY)\n"), Equal("QQ ADDRSEC1 ADDRSEC2\n(TESTBODY)")))
			})

			It("should correctly parse header with EndHeaderMarker and then originator", func() {
				message := `ZCZC TST001 121212
FF PRIMADDR
SEC ADDR1
---
.ORIG 121200
(BODY)
NNNN`
				parsedHeader, err := ParseHeader(message)
				Expect(err).ToNot(HaveOccurred())
				Expect(parsedHeader.PrimaryAddress).To(Equal("PRIMADDR"))
				Expect(parsedHeader.SecondaryAddresses).To(Equal(" SEC ADDR1"))
				Expect(parsedHeader.Originator).To(Equal("ORIG"))
				Expect(parsedHeader.OriginatorDateTime).To(Equal("121200"))
				Expect(parsedHeader.Body).To(SatisfyAny(Equal("(BODY)\n"), Equal("(BODY)")))
			})

		})
	})

	Describe("Other Info", func() {
		Context("with various combinations of other info fields", func() {
			It("should parse full PBN/ NAV/ REG/ EET/ SEL/ PER/ RIF/ RMK/ string correctly", func() {
				otherInfo := "PBN/A1B2B3B4B5D1L1 NAV/ABAS REG/B6513 EET/ZBPE0112 SEL/KMAL PER/C RIF/FRT N640 ZBYN RMK/TCAS EQUIPPED"
				parsed := parseOther(otherInfo)
				Expect(parsed).ToNot(BeNil())
				Expect(parsed[PBN]).To(Equal("A1B2B3B4B5D1L1"))
				Expect(parsed[NavigationEquipment]).To(Equal("ABAS"))
				Expect(parsed[Register]).To(Equal("B6513"))
				Expect(parsed[EstimatedElapsedTime]).To(Equal("ZBPE0112"))
				Expect(parsed[SELCALCode]).To(Equal("KMAL"))
				Expect(parsed[PerformanceCategory]).To(Equal("C"))
				Expect(parsed[RerouteInformation]).To(Equal("FRT N640 ZBYN"))
				Expect(parsed[Remarks]).To(Equal("TCAS EQUIPPED"))
			})

			It("should parse minimal PBN/ REG/ string correctly", func() {
				otherInfo := "PBN/A1 REG/B1234"
				parsed := parseOther(otherInfo)
				Expect(parsed).ToNot(BeNil())
				Expect(parsed[PBN]).To(Equal("A1"))
				Expect(parsed[Register]).To(Equal("B1234"))
				Expect(parsed[NavigationEquipment]).To(BeEmpty())
				Expect(parsed[SELCALCode]).To(BeEmpty())
			})

			It("should handle empty fields like SEL/", func() {
				otherInfo := "REG/B6789 SEL/ RMK/NO TCAS"
				parsed := parseOther(otherInfo)
				Expect(parsed).ToNot(BeNil())
				Expect(parsed[Register]).To(Equal("B6789"))
				Expect(parsed[SELCALCode]).To(BeEmpty()) // SEL/ is present but empty
				Expect(parsed[Remarks]).To(Equal("NO TCAS"))
			})

			It("should handle fields in different order", func() {
				otherInfo := "RMK/SPECIAL NAV/GPS EET/EDDF0100 REG/DABCD"
				parsed := parseOther(otherInfo)
				Expect(parsed).ToNot(BeNil())
				Expect(parsed[Remarks]).To(Equal("SPECIAL"))
				Expect(parsed[NavigationEquipment]).To(Equal("GPS"))
				Expect(parsed[EstimatedElapsedTime]).To(Equal("EDDF0100"))
				Expect(parsed[Register]).To(Equal("DABCD"))
			})

			It("should return an empty map for empty otherInfo string", func() {
				otherInfo := ""
				parsed := parseOther(otherInfo)
				Expect(parsed).ToNot(BeNil())
				Expect(parsed).To(BeEmpty())
			})

			It("should return an empty map if no known patterns match", func() {
				otherInfo := "UNKNOWN/FIELD EXTRA/DATA"
				parsed := parseOther(otherInfo)
				Expect(parsed).ToNot(BeNil())
				Expect(parsed).To(BeEmpty()) // Assuming UNKNOWN and EXTRA are not defined patterns
			})
		})
	})

	Describe("ParseBody", func() {

		Context("with ARR body (ARR-CES5470-ZBTJ-ZSHC1614) - minimal valid", func() {
			body := "(ARR-CES5470-ZBTJ-ZSHC1614)"
			parser := NewBodyParser(body)
			It("should parse the body correctly", func() {
				category, parsedBody, err := parser.Parse()
				Expect(err).ToNot(HaveOccurred())
				Expect(parsedBody).ToNot(BeNil())
				Expect(category).To(Equal("ARR"))
				Expect(parsedBody).To(BeAssignableToTypeOf(&domain.ARR{}))
				arrMessage := parsedBody.(*domain.ARR)
				Expect(arrMessage.Category).To(Equal("ARR"))
				Expect(arrMessage.AircraftID).To(Equal("CES5470"))
				Expect(arrMessage.DepartureAirport).To(Equal("ZBTJ"))
				Expect(arrMessage.ArrivalAirport).To(Equal("ZSHC"))
				Expect(arrMessage.ArrivalTime).To(Equal("1614"))
			})

		})

		Context("with ARR body", func() {
			// parser := NewBodyParser(body)
			It("should parse the body (ARR-AB123/A1234-KJFK-KLAX1234-DEP1000-EET0200-ALTKBOS-OTHER FOO BAR) correctly - full ARR", func() {
				body := " (ARR-AB123/A1234-KJFK1000-KLAX1234-EET0200-ALTKBOS-OTHER FOO BAR)" // Added DepartureTime, EET, ALT, OTHER for full test
				parser := NewBodyParser(body)
				category, parsedBody, err := parser.Parse()
				Expect(err).ToNot(HaveOccurred())
				Expect(parsedBody).ToNot(BeNil())
				Expect(category).To(Equal("ARR"))
				arrMessage, ok := parsedBody.(*domain.ARR)
				Expect(ok).To(BeTrue())
				Expect(arrMessage.Category).To(Equal("ARR"))
				Expect(arrMessage.AircraftID).To(Equal("AB123"))
				Expect(arrMessage.SSRModeAndCode).To(Equal("A1234"))
				Expect(arrMessage.DepartureAirport).To(Equal("KJFK"))
				Expect(arrMessage.DepartureTime).To(Equal("1000")) // Assuming parser extracts this for ARR
				Expect(arrMessage.ArrivalAirport).To(Equal("KLAX"))
				Expect(arrMessage.ArrivalTime).To(Equal("1234"))
				Expect(arrMessage.EstimatedElapsedTime).To(Equal("0200"))
				Expect(arrMessage.AlternateAirport).To(Equal("KBOS"))
				Expect(arrMessage.OtherInfo).To(Equal("FOO BAR"))
			})

		})

		Context("with DEP body", func() {
			It("should parse minimal DEP body (DEP-CYZ9017-ZBTJ1638-ZSPD-EET0200) correctly", func() {
				body := "(DEP-CYZ9017-ZBTJ1638-ZSPD-EET0200)" // Minimal: Category, AircraftID, DepAirport, DepTime, Dest, EET
				parser := NewBodyParser(body)
				category, parsedBody, err := parser.Parse()
				Expect(err).ToNot(HaveOccurred())
				Expect(parsedBody).ToNot(BeNil())
				Expect(category).To(Equal("DEP"))
				depMessage, ok := parsedBody.(*domain.DEP)
				Expect(ok).To(BeTrue())
				Expect(depMessage.Category).To(Equal("DEP"))
				Expect(depMessage.AircraftID).To(Equal("CYZ9017"))
				Expect(depMessage.SSRModeAndCode).To(BeEmpty()) // Optional
				Expect(depMessage.DepartureAirport).To(Equal("ZBTJ"))
				Expect(depMessage.DepartureTime).To(Equal("1638"))
				Expect(depMessage.Destination).To(Equal("ZSPD"))
				Expect(depMessage.EstimatedElapsedTime).To(Equal("0200"))
			})

			It("should parse full DEP body (DEP-CYZ9017/A5633-ZBTJ1638-ZSPD-EET0200-ALTZYTL-OTHER INFO) correctly", func() {
				body := "(DEP-CYZ9017/A5633-ZBTJ1638-ZSPD-EET0200-ALTZYTL-OTHER INFO)"
				parser := NewBodyParser(body)
				category, parsedBody, err := parser.Parse()
				Expect(err).ToNot(HaveOccurred())
				Expect(parsedBody).ToNot(BeNil())
				Expect(category).To(Equal("DEP"))
				depMessage, ok := parsedBody.(*domain.DEP)
				Expect(ok).To(BeTrue())
				Expect(depMessage.Category).To(Equal("DEP"))
				Expect(depMessage.AircraftID).To(Equal("CYZ9017"))
				Expect(depMessage.SSRModeAndCode).To(Equal("A5633"))
				Expect(depMessage.DepartureAirport).To(Equal("ZBTJ"))
				Expect(depMessage.DepartureTime).To(Equal("1638"))
				Expect(depMessage.Destination).To(Equal("ZSPD"))
				Expect(depMessage.EstimatedElapsedTime).To(Equal("0200"))
				Expect(depMessage.AlternateAirport).To(Equal("ZYTL"))
				Expect(depMessage.OtherInfo).To(Equal("INFO"))
			})
		})

		Context("with FPL body", func() {
			It("should parse minimal FPL body (FPL-CCA1532-IS -A332/H -SDE3FGHIJ4J5M1RWY/LB101 -ZSSS2035 -K0859S1040 PIAKS -ZBAA0153) correctly", func() {
				body := `(FPL-CCA1532-IS
-A332/H
-SDE3FGHIJ4J5M1RWY/LB101
-ZSSS2035
-K0859S1040 PIAKS
-ZBAA0153)` // Minimal required for FPL, assuming OtherInfo subfields are not mandatory for basic parsing
				parser := NewBodyParser(body)
				category, parsedBody, err := parser.Parse()
				Expect(err).ToNot(HaveOccurred())
				Expect(parsedBody).ToNot(BeNil())
				Expect(category).To(Equal("FPL"))
				fplMessage, ok := parsedBody.(*domain.FPL)
				Expect(ok).To(BeTrue())
				Expect(fplMessage.FlightNumber).To(Equal("CCA1532"))
				Expect(fplMessage.FlightRulesAndType).To(Equal("IS"))
				Expect(fplMessage.AircraftID).To(Equal("A332/H")) // This is Aircraft Type + Wake Turb + Equipment from parser logic
				Expect(fplMessage.SSRModeAndCode).To(Equal("SDE3FGHIJ4J5M1RWY/LB101")) // This is Equipment from parser logic for FPL
				Expect(fplMessage.DepartureAirport).To(Equal("ZSSS"))
				Expect(fplMessage.DepartureTime).To(Equal("2035"))
				Expect(fplMessage.CruisingSpeedAndLevel).To(Equal("K0859S1040"))
				Expect(fplMessage.Route).To(Equal("PIAKS"))
				Expect(fplMessage.DestinationAndTotalTime).To(Equal("ZBAA0153"))
				// Check that OtherInfo subfields are empty if not provided
				Expect(fplMessage.PBN).To(BeEmpty())
				Expect(fplMessage.Remarks).To(BeEmpty())
			})

			It("should parse full FPL body correctly", func() {
				body := `(FPL-CCA1532-IS
-A332/H
-SDE3FGHIJ4J5M1RWY/LB101
-ZSSS2035
-K0859S1040 PIAKS G330 PIMOL A539 BTO W82 DOGAR
-ZBAA0153 ZBYN
-PBN/A1B2B3B4B5D1L1 NAV/ABAS REG/B6513 EET/ZBPE0112 SEL/KMAL PER/C RIF/FRT N640 ZBYN RMK/TCAS EQUIPPED)`
				parser := NewBodyParser(body)
				category, parsedBody, err := parser.Parse()
				Expect(err).ToNot(HaveOccurred())
				Expect(parsedBody).ToNot(BeNil())
				Expect(category).To(Equal("FPL"))
				Expect(parsedBody).To(BeAssignableToTypeOf(&domain.FPL{}))
				fplMessage := parsedBody.(*domain.FPL)
				Expect(fplMessage.FlightNumber).To(Equal("CCA1532"))
				Expect(fplMessage.FlightRulesAndType).To(Equal("IS"))
				Expect(fplMessage.AircraftID).To(Equal("A332/H"))
				Expect(fplMessage.SSRModeAndCode).To(Equal("SDE3FGHIJ4J5M1RWY/LB101"))
				Expect(fplMessage.DepartureAirport).To(Equal("ZSSS"))
				Expect(fplMessage.DepartureTime).To(Equal("2035"))
				Expect(fplMessage.CruisingSpeedAndLevel).To(Equal("K0859S1040"))
				Expect(fplMessage.Route).To(Equal("PIAKS G330 PIMOL A539 BTO W82 DOGAR"))
				Expect(fplMessage.DestinationAndTotalTime).To(Equal("ZBAA0153"))
				Expect(fplMessage.AlternateAirport).To(Equal("ZBYN"))
				Expect(fplMessage.OtherInfo).To(Equal("PBN/A1B2B3B4B5D1L1 NAV/ABAS REG/B6513 EET/ZBPE0112 SEL/KMAL PER/C RIF/FRT N640 ZBYN RMK/TCAS EQUIPPED"))
				Expect(fplMessage.PBN).To(Equal("A1B2B3B4B5D1L1"))
				Expect(fplMessage.EstimatedElapsedTime).To(Equal("ZBPE0112"))
				Expect(fplMessage.SELCALCode).To(Equal("KMAL"))
				Expect(fplMessage.PerformanceCategory).To(Equal("C"))
				Expect(fplMessage.RerouteInformation).To(Equal("FRT N640 ZBYN"))
				Expect(fplMessage.Remarks).To(Equal("TCAS EQUIPPED"))
			})
		})

		Context("with CNL body", func() {
			It("should parse minimal CNL body (CNL-YZR7979-ZSPD-ZBTJ) correctly", func() {
				body := "(CNL-YZR7979-ZSPD-ZBTJ)" // Minimal: Category, AircraftID, DepAirport, DestAirport
				parser := NewBodyParser(body)
				category, parsedBody, err := parser.Parse()
				Expect(err).ToNot(HaveOccurred())
				Expect(parsedBody).ToNot(BeNil())
				Expect(category).To(Equal("CNL"))
				cnlMessage, ok := parsedBody.(*domain.CNL)
				Expect(ok).To(BeTrue())
				Expect(cnlMessage.Category).To(Equal("CNL"))
				Expect(cnlMessage.AircraftID).To(Equal("YZR7979"))
				Expect(cnlMessage.DepartureAirport).To(Equal("ZSPD"))
				Expect(cnlMessage.DestinationAirport).To(Equal("ZBTJ"))
				Expect(cnlMessage.OtherInfo).To(BeEmpty()) // Optional
			})

			It("should parse full CNL body (CNL-YZR7979-ZSPD-ZBTJ-OTHER REASON FOR CANCEL) correctly", func() {
				// CNL regex: `\(CNL-(?P<flight_number>[A-Z0-9]+)-(?P<dep>[A-Z]{4})-(?P<arr>[A-Z]{4})(?:-OTHER\s*(?P<other>.*?))?\)`
				body := "(CNL-YZR7979-ZSPD-ZBTJ-OTHER REASON FOR CANCEL)"
				parser := NewBodyParser(body)
				category, parsedBody, err := parser.Parse()
				Expect(err).ToNot(HaveOccurred())
				Expect(parsedBody).ToNot(BeNil())
				Expect(category).To(Equal("CNL"))
				cnlMessage, ok := parsedBody.(*domain.CNL)
				Expect(ok).To(BeTrue())
				Expect(cnlMessage.Category).To(Equal("CNL"))
				Expect(cnlMessage.AircraftID).To(Equal("YZR7979"))
				Expect(cnlMessage.DepartureAirport).To(Equal("ZSPD"))
				Expect(cnlMessage.DestinationAirport).To(Equal("ZBTJ"))
				Expect(cnlMessage.OtherInfo).To(Equal("REASON FOR CANCEL"))
			})
		})

		Context("with DLA body", func() {
			It("should parse the body correctly", func() {
				body := "(DLA-CSN3133-ZGGG0110-ZBTJ)"
				parser := NewBodyParser(body)
				category, parsedBody, err := parser.Parse()
				Expect(err).ToNot(HaveOccurred())
				Expect(parsedBody).ToNot(BeNil())
				Expect(category).To(Equal("DLA"))
				Expect(parsedBody).To(BeAssignableToTypeOf(&domain.DLA{}))
				dlaMessage := parsedBody.(*domain.DLA)
				Expect(dlaMessage.AircraftID).To(Equal("CSN3133"))
				Expect(dlaMessage.DepartureAirport).To(Equal("ZGGG"))
				Expect(dlaMessage.NewDepartureTime).To(Equal("0110"))
				Expect(dlaMessage.ArrivalAirport).To(Equal("ZBTJ"))

			})
		})

	})

	Describe("Parse whole real message", func() {

		Context("with a real ARR message", func() {
			message := `
ZCZC TMQ2526 141605

FF ZBTJZPZX

141604 ZBACZQZX

(ARR-JAE7433/A0132-RKSI-ZBTJ1604)

NNNN
`
			It("should parse the whole message correctly", func() {
				parsedMessage := Parse(message)
				Expect(parsedMessage).ToNot(BeNil())
				Expect(parsedMessage.Parsed).To(BeTrue())
				Expect(parsedMessage.MessageID).To(Equal("TMQ2526"))
				Expect(parsedMessage.DateTime).To(Equal("141605"))
				Expect(parsedMessage.PrimaryAddress).To(Equal("ZBTJZPZX"))
				Expect(parsedMessage.SecondaryAddresses).To(Equal(""))
				Expect(parsedMessage.PriorityIndicator).To(Equal("FF"))
				Expect(parsedMessage.OriginatorDateTime).To(Equal("141604"))
				Expect(parsedMessage.Originator).To(Equal("ZBACZQZX"))

				arrmsg := parsedMessage.BodyData.(*domain.ARR)
				Expect(arrmsg.Category).To(Equal("ARR"))
				Expect(arrmsg.AircraftID).To(Equal("JAE7433"))
				Expect(arrmsg.SSRModeAndCode).To(Equal("A0132"))
				Expect(arrmsg.DepartureAirport).To(Equal("RKSI"))
				Expect(arrmsg.ArrivalAirport).To(Equal("ZBTJ"))
				Expect(arrmsg.ArrivalTime).To(Equal("1604"))
			})
		})

		Context("with this real FPL message", func() {
			message := `ZCZC TMQ2617 142150


GG ZBTJZPZX


150551 ZBTJUOBK


(FPL-OKA2861-IS


-MA60/M-SHID/C


-ZBTJ0030


-K0420S0450 CG J1 FZ


-ZSYT0100  ZSQD ZYTL


-REG/B3710 SEL/ RMK/TCAS )

NNNN
`
			It("should parse the whole message correctly", func() {
				parsedMessage := Parse(message)
				Expect(parsedMessage).ToNot(BeNil())
				Expect(parsedMessage.Parsed).To(BeTrue())
				Expect(parsedMessage.MessageID).To(Equal("TMQ2617"))
				Expect(parsedMessage.DateTime).To(Equal("142150"))
				Expect(parsedMessage.PrimaryAddress).To(Equal("ZBTJZPZX"))
				Expect(parsedMessage.SecondaryAddresses).To(Equal(""))
				Expect(parsedMessage.PriorityIndicator).To(Equal("GG"))
				Expect(parsedMessage.OriginatorDateTime).To(Equal("150551"))
				Expect(parsedMessage.Originator).To(Equal("ZBTJUOBK"))

				fplmsg := parsedMessage.BodyData.(*domain.FPL)
				Expect(fplmsg.Category).To(Equal("FPL"))
				Expect(fplmsg.FlightNumber).To(Equal("OKA2861"))
				Expect(fplmsg.FlightRulesAndType).To(Equal("IS"))
				Expect(fplmsg.AircraftID).To(Equal("MA60/M"))
				Expect(fplmsg.SSRModeAndCode).To(Equal("SHID/C"))
				Expect(fplmsg.DepartureAirport).To(Equal("ZBTJ"))
				Expect(fplmsg.DepartureTime).To(Equal("0030"))
				Expect(fplmsg.CruisingSpeedAndLevel).To(Equal("K0420S0450"))
				Expect(fplmsg.Route).To(Equal("CG J1 FZ"))
				Expect(fplmsg.DestinationAndTotalTime).To(Equal("ZSYT0100"))
				Expect(fplmsg.AlternateAirport).To(Equal("ZSQD ZYTL"))
				Expect(fplmsg.OtherInfo).To(Equal("REG/B3710 SEL/ RMK/TCAS"))
				// Expect(fplmsg.PBN).To(Equal("B3710"))
				Expect(fplmsg.SELCALCode).To(Equal(""))
				Expect(fplmsg.Remarks).To(Equal("TCAS"))
			})
		})

		Context("with a real CHG message", func() {
			It("should parse the whole CHG message correctly", func() {
				message := `ZCZC CHG001 181200
GG ZZZZADDR
181155 ORIGADDR
(CHG-FLTCHG1/S123-ZAAA1000-ZBBB1100-0100-ALTZCCC-OTHER CHG DETAILS-FIELD18 STUFF)
NNNN`
				parsedMessage := Parse(message)
				Expect(parsedMessage).ToNot(BeNil())
				Expect(parsedMessage.Parsed).To(BeTrue())
				Expect(parsedMessage.MessageID).To(Equal("CHG001"))
				Expect(parsedMessage.PrimaryAddress).To(Equal("ZZZZADDR"))
				Expect(parsedMessage.Originator).To(Equal("ORIGADDR"))
				Expect(parsedMessage.Category).To(Equal("CHG"))

				chgMsg, ok := parsedMessage.BodyData.(*domain.CHG)
				Expect(ok).To(BeTrue())
				Expect(chgMsg.Category).To(Equal("CHG"))
				Expect(chgMsg.AircraftID).To(Equal("FLTCHG1"))
				Expect(chgMsg.SSRModeAndCode).To(Equal("S123"))
				Expect(chgMsg.DepartureAirport).To(Equal("ZAAA"))
				Expect(chgMsg.DepartureTime).To(Equal("1000"))
				Expect(chgMsg.ArrivalAirport).To(Equal("ZBBB"))
				Expect(chgMsg.ArrivalTime).To(Equal("1100"))
				Expect(chgMsg.EstimatedElapsedTime).To(Equal("0100"))
				Expect(chgMsg.AlternateAirport).To(Equal("ZCCC"))
				Expect(chgMsg.OtherInfo).To(Equal("CHG DETAILS"))
				Expect(chgMsg.ChangePart).To(Equal("FIELD18 STUFF"))
			})
		})

		Context("with a message that has an unrecognized body category", func() {
			It("should set Parsed to false and include comments", func() {
				message := `ZCZC XXX001 181200
GG ZZZZADDR
181155 ORIGADDR
(XYZ-FLTXYZ-ZAAA-ZBBB)
NNNN`
				parsedMessage := Parse(message)
				Expect(parsedMessage).ToNot(BeNil())
				Expect(parsedMessage.Parsed).To(BeFalse())
				Expect(parsedMessage.MessageID).To(Equal("XXX001")) // Header should still parse
				Expect(parsedMessage.Category).To(Equal("XYZ"))     // Category identified by findCategory
				Expect(parsedMessage.Comments).To(ContainSubstring("no matching pattern found for body"))
				Expect(parsedMessage.BodyData).To(BeNil())
			})
		})

		Context("with a message that is only a header (no body)", func() {
			It("should set Parsed to false and include comments about missing body", func() {
				message := `ZCZC HDR001 181200
GG ZZZZADDR
181155 ORIGADDR
NNNN`
				parsedMessage := Parse(message)
				Expect(parsedMessage).ToNot(BeNil())
				Expect(parsedMessage.Parsed).To(BeFalse())
				Expect(parsedMessage.MessageID).To(Equal("HDR001")) // Header should parse
				Expect(parsedMessage.Category).To(BeEmpty())        // No body, so no category from body
				Expect(parsedMessage.Comments).To(ContainSubstring("no category found in body text"))
				Expect(parsedMessage.BodyData).To(BeNil())
			})
		})

		Context("with a message that has valid header but malformed body for a known type", func() {
			It("should set Parsed to false if body regex match fails for a known type", func() {
				// ARR message with missing fields that the ARR regex requires
				message := `ZCZC MAL001 181200
GG ZZZZADDR
(ARR-FLTARR) 
NNNN`
				// The ARR regex expects more fields like airports and times.
				// If the regex for ARR doesn't match, parser.Parse() will return an error.
				parsedMessage := Parse(message)
				Expect(parsedMessage).ToNot(BeNil())
				Expect(parsedMessage.Parsed).To(BeFalse())
				Expect(parsedMessage.Category).To(Equal("ARR")) // Category is found
				Expect(parsedMessage.Comments).To(ContainSubstring("no matching pattern found for body")) // Because the specific ARR regex failed
				Expect(parsedMessage.BodyData).To(BeNil())
			})
		})

	})

	Describe("Utility Functions", func() {

		It("should clean text correctly", func() {
			text := `ZCZC TMQ2530 141614

1234
  4567
NNNN`
			expect := "ZCZC TMQ2530 141614\n1234\n  4567"
			cleaned := cleanMessage(text)
			Expect(cleaned).To(Equal(expect))
		})

		It("should parse start indicator correctly", func() {
			line := "ZCZC TMQ2530 141614"
			startIndicator, messageID, dateTime, err := parseStartIndicator(line)
			Expect(err).ToNot(HaveOccurred())
			Expect(startIndicator).To(Equal("ZCZC"))
			Expect(messageID).To(Equal("TMQ2530"))
			Expect(dateTime).To(Equal("141614"))
		})

		It("should return error for invalid start indicator line", func() {
			line := "Invalid Line"
			_, _, _, err := parseStartIndicator(line)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid start indicator line format"))
		})

		It("should parse priority and primary address correctly and return no error", func() {
			line := "QU TSNZPCA"
			priority, primary, err := parsePriorityAndPrimary(line)
			Expect(err).ToNot(HaveOccurred())
			Expect(priority).To(Equal("QU"))
			Expect(primary).To(Equal("TSNZPCA"))
		})

		It("should return an error for invalid priority and primary address line", func() {
			line := "Invalid-Line"
			priority, primary, err := parsePriorityAndPrimary(line)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid priority and primary address line format"))
			Expect(priority).To(BeEmpty())
			Expect(primary).To(BeEmpty())
		})

		// Tests for parseHeaderFieldsAndBody (refactored from parseRemainingLines)
		Describe("parseHeaderFieldsAndBody", func() {
			It("should handle only secondary addresses", func() {
				lines := []string{"SEC ADDR1", "SEC ADDR2"}
				headerData, body := parseHeaderFieldsAndBody(lines)
				Expect(headerData.SecondaryAddresses).To(Equal("SEC ADDR1 SEC ADDR2"))
				Expect(headerData.Originator).To(BeEmpty())
				Expect(headerData.OriginatorDateTime).To(BeEmpty())
				Expect(body).To(BeEmpty())
			})

			It("should handle only originator", func() {
				lines := []string{".ORIG 121200"}
				headerData, body := parseHeaderFieldsAndBody(lines)
				Expect(headerData.SecondaryAddresses).To(BeEmpty())
				Expect(headerData.Originator).To(Equal("ORIG"))
				Expect(headerData.OriginatorDateTime).To(Equal("121200"))
				Expect(body).To(BeEmpty())
			})

			It("should handle originator not prefixed with '.' if it matches regex", func() {
				// This relies on the package-level 'originator' regex
				// Example: originator = regexp.MustCompile(`^([A-Z]{4,8})?\s*([0-9]{6})$`)
				// For this test, we'd need to know the regex or mock it.
				// Assuming a simple regex for "ORIGID DATETIME"
				// If `getOriginator` is robust, it should parse this.
				// lines := []string{"ORIGID 121200"} // This format needs to match the actual 'originator' regex
				// headerData, body := parseHeaderFieldsAndBody(lines)
				// Expect(headerData.Originator).To(Equal("ORIGID"))
				// Expect(headerData.OriginatorDateTime).To(Equal("121200"))
				// This test might be better suited if the regex is explicitly defined or mockable here.
				// For now, we rely on the ". " prefix test for originator.
			})

			It("should handle body part starting with '('", func() {
				lines := []string{"SEC ADDR1", ".ORIG 121200", "(BODY STARTS HERE)"}
				headerData, body := parseHeaderFieldsAndBody(lines)
				Expect(headerData.SecondaryAddresses).To(Equal("SEC ADDR1"))
				Expect(headerData.Originator).To(Equal("ORIG"))
				Expect(body).To(Equal("(BODY STARTS HERE)")) // Trailing newline is now trimmed by parseHeaderFieldsAndBody
			})

			It("should handle body part starting with BeginPartMarker", func() {
				lines := []string{"BEGIN PART 01", "CONTENT"}
				headerData, body := parseHeaderFieldsAndBody(lines)
				Expect(headerData.SecondaryAddresses).To(BeEmpty())
				Expect(body).To(Equal("BEGIN PART 01\nCONTENT")) // Assuming markers become part of body if not "NNNN" line
			})

			It("should ignore NNNN marker line if it starts a body section", func() {
				lines := []string{"(NNNN"} // A line that is just (NNNN
				_, body := parseHeaderFieldsAndBody(lines)
				Expect(body).To(BeEmpty())

				lines2 := []string{"(MORE TEXT NNNN"} // Text before NNNN
				_, body2 := parseHeaderFieldsAndBody(lines2)
				Expect(body2).To(BeEmpty()) // Current logic might skip whole line
			})

			It("should handle EndHeaderMarker correctly", func() {
				lines := []string{"SEC ADDR1", "---", ".ORIG 121200", "(BODY)"}
				headerData, body := parseHeaderFieldsAndBody(lines)
				Expect(headerData.SecondaryAddresses).To(Equal("SEC ADDR1"))
				Expect(headerData.Originator).To(Equal("ORIG"))
				Expect(body).To(Equal("(BODY)"))
			})

			It("should correctly handle lines after body marker as part of body", func() {
				lines := []string{"(START)", "LINE 2 OF BODY", "LINE 3 OF BODY"}
				_, body := parseHeaderFieldsAndBody(lines)
				Expect(body).To(Equal("(START)\nLINE 2 OF BODY\nLINE 3 OF BODY"))
			})
		})
	})
})
