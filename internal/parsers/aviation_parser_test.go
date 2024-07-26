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
			Expect(parsedHeader.SecondaryAddresses).To(Equal([]string{"QU PEKUDCA TSNUOCA TSNZPCA TSNUFCA"}))
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
			Expect(parsedHeader.SecondaryAddresses).To(Equal([]string{"QU PEKUDCA TSNUOCA TSNZPCA TSNUFCA"}))
			Expect(parsedHeader.Originator).To(Equal("SELOZKE"))
			Expect(parsedHeader.OriginatorDateTime).To(Equal("170999"))
		})

	})

	Describe("ParseBody", func() {

		Context("with ARR body (ARR-CES5470-ZBTJ-ZSHC1614)", func() {
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
			It("should parse the body (ARR-AB123/A1234-KJFK-KLAX1234) correctly", func() {
				body := " (ARR-AB123/A1234-KJFK-KLAX1234)"
				parser := NewBodyParser(body)
				category, parsedBody, err := parser.Parse()
				Expect(err).ToNot(HaveOccurred())
				Expect(parsedBody).ToNot(BeNil())
				Expect(category).To(Equal("ARR"))
				Expect(parsedBody).To(BeAssignableToTypeOf(&domain.ARR{}))
				arrMessage := parsedBody.(*domain.ARR)
				Expect(arrMessage.Category).To(Equal("ARR"))
				Expect(arrMessage.AircraftID).To(Equal("AB123"))
				Expect(arrMessage.SSRModeAndCode).To(Equal("A1234"))
				Expect(arrMessage.DepartureAirport).To(Equal("KJFK"))
				Expect(arrMessage.ArrivalAirport).To(Equal("KLAX"))
			})

			It("should parse the body (ARR-JAE7433/A0132-RKSI-ZBTJ1604) correctly", func() {
				body := " (ARR-JAE7433/A0132-RKSI-ZBTJ1604)"
				parser := NewBodyParser(body)
				category, parsedBody, err := parser.Parse()
				Expect(err).ToNot(HaveOccurred())
				Expect(parsedBody).ToNot(BeNil())
				Expect(category).To(Equal("ARR"))
				Expect(parsedBody).To(BeAssignableToTypeOf(&domain.ARR{}))
				arrMessage := parsedBody.(*domain.ARR)
				Expect(arrMessage.Category).To(Equal("ARR"))
				Expect(arrMessage.AircraftID).To(Equal("JAE7433"))
				Expect(arrMessage.SSRModeAndCode).To(Equal("A0132"))
				Expect(arrMessage.DepartureAirport).To(Equal("RKSI"))
				Expect(arrMessage.ArrivalAirport).To(Equal("ZBTJ"))
			})

		})

		Context("with DEP body", func() {
			// parser := NewBodyParser()
			It("should parse the body (DEP-CYZ9017/A5633-ZBTJ1638-ZSPD) correctly", func() {
				body := "(DEP-CYZ9017/A5633-ZBTJ1638-ZSPD)"
				parser := NewBodyParser(body)
				category, parsedBody, err := parser.Parse()
				Expect(err).ToNot(HaveOccurred())
				Expect(parsedBody).ToNot(BeNil())
				Expect(category).To(Equal("DEP"))
				Expect(parsedBody).To(BeAssignableToTypeOf(&domain.DEP{}))
				depMessage := parsedBody.(*domain.DEP)
				Expect(depMessage.Category).To(Equal("DEP"))
				Expect(depMessage.AircraftID).To(Equal("CYZ9017"))
				Expect(depMessage.SSRModeAndCode).To(Equal("A5633"))
				Expect(depMessage.DepartureAirport).To(Equal("ZBTJ"))
				Expect(depMessage.DepartureTime).To(Equal("1638"))
				Expect(depMessage.Destination).To(Equal("ZSPD"))
			})
		})

		Context("with FPL body", func() {
			// parser := NewBodyParser()
			It("should parse the body correctly", func() {
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
			// parser := NewBodyParser()
			It("should parse the body correctly", func() {
				body := "(CNL-YZR7979-ZSPD-ZBTJ)"
				parser := NewBodyParser(body)
				category, parsedBody, err := parser.Parse()
				Expect(err).ToNot(HaveOccurred())
				Expect(parsedBody).ToNot(BeNil())
				Expect(category).To(Equal("CNL"))
				Expect(parsedBody).To(BeAssignableToTypeOf(&domain.CNL{}))
				cnlMessage := parsedBody.(*domain.CNL)
				Expect(cnlMessage.AircraftID).To(Equal("YZR7979"))
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
				Expect(parsedMessage.SecondaryAddresses).To(BeNil())
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
				Expect(parsedMessage.SecondaryAddresses).To(BeNil())
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
		})

		It("should parse priority and primary address correctly", func() {
			line := "QU TSNZPCA"
			priority, primary := parsePriorityAndPrimary(line)
			Expect(priority).To(Equal("QU"))
			Expect(primary).To(Equal("TSNZPCA"))
		})

		It("should return empty strings for invalid priority and primary address line", func() {
			line := "Invalid-Line"
			priority, primary := parsePriorityAndPrimary(line)
			Expect(priority).To(BeEmpty())
			Expect(primary).To(BeEmpty())
		})

		It("should parse remaining lines correctly", func() {
			lines := []string{"QU PEKUDCA TSNUOCA TSNZPCA TSNUFCA", ".SELOZKE 170999", "BEGIN PART 01"}
			secondaryAddresses, originator, originatorDateTime, bodyAndFooter := parseRemainingLines(lines)
			Expect(secondaryAddresses).To(Equal([]string{"QU PEKUDCA TSNUOCA TSNZPCA TSNUFCA"}))
			Expect(originator).To(Equal("SELOZKE"))
			Expect(originatorDateTime).To(Equal("170999"))
			Expect(bodyAndFooter).To(Equal("BEGIN PART 01\n"))
		})
	})
})
