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
				body := clean(message)
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
			parser := NewBodyParser()
			It("should parse the body correctly", func() {
				category, parsedBody, err := parser.Parse(body)
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
			parser := NewBodyParser()
			It("should parse the body (ARR-AB123/A1234-KJFK-KLAX1234) correctly", func() {
				body := " (ARR-AB123/A1234-KJFK-KLAX1234)"
				category, parsedBody, err := parser.Parse(body)
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
				category, parsedBody, err := parser.Parse(body)
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
			parser := NewBodyParser()
			It("should parse the body (DEP-CYZ9017/A5633-ZBTJ1638-ZSPD) correctly", func() {
				body := "(DEP-CYZ9017/A5633-ZBTJ1638-ZSPD)"
				category, parsedBody, err := parser.Parse(body)
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
			parser := NewBodyParser()
			It("should parse the body correctly", func() {
				body := `(FPL-CCA1532-IS
-A332/H
-SDE3FGHIJ4J5M1RWY/LB101
-ZSSS2035
-K0859S1040 PIAKS G330 PIMOL A539 BTO W82 DOGAR
-ZBAA0153 ZBYN
-PBN/A1B2B3B4B5D1L1 NAV/ABAS REG/B6513 EET/ZBPE0112 SEL/KMAL PER/C RIF/FRT N640 ZBYN RMK/TCAS EQUIPPED)`
				category, parsedBody, err := parser.Parse(body)
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
				parsedMessage, err := Parse(message)
				Expect(err).ToNot(HaveOccurred())
				Expect(parsedMessage).ToNot(BeNil())
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
	})
})
