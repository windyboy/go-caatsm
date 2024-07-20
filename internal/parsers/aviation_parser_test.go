package parsers

import (
	"caatsm/internal/domain"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Aviation Parser", func() {
	Describe("ParseHeader", func() {
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
			Expect(parsedHeader.StartIndicator).To(Equal("ZCZC"))
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
			Expect(parsedHeader.StartIndicator).To(Equal("ZCZC"))
			Expect(parsedHeader.MessageID).To(Equal("NOTAM1122"))
			Expect(parsedHeader.DateTime).To(Equal("171000"))
			Expect(parsedHeader.PriorityIndicator).To(Equal("QU"))
			Expect(parsedHeader.PrimaryAddress).To(Equal("TSNZPCA"))
			Expect(parsedHeader.SecondaryAddresses).To(Equal([]string{"QU PEKUDCA TSNUOCA TSNZPCA TSNUFCA"}))
			Expect(parsedHeader.Originator).To(Equal("SELOZKE"))
			Expect(parsedHeader.OriginatorDateTime).To(Equal("170999"))
		})

	})

	Context("with NOTAM message", func() {
		It("should parse the header correctly", func() {
			message := `
ZCZC NOTAM1234 230715
GG EDDNZEZN
.
GG EDDNYNYX
.BERLINTWR 230714

Q) EDMM/QOATT/IV/BO/A/000/999/4814N01120E005
A) EDDM
B) 2307150600 C) 2307151800
E) AERODROME CONTROL TOWER HOURS OF SERVICE
   0600-1800 DUE TO MAINTENANCE
NNNN
`

			parsedMessage, err := ParseHeader(message)
			Expect(err).ToNot(HaveOccurred())
			Expect(parsedMessage.StartIndicator).To(Equal("ZCZC"))
			Expect(parsedMessage.MessageID).To(Equal("NOTAM1234"))
			Expect(parsedMessage.DateTime).To(Equal("230715"))
			Expect(parsedMessage.PriorityIndicator).To(Equal("GG"))
			Expect(parsedMessage.PrimaryAddress).To(Equal("EDDNZEZN"))
			Expect(parsedMessage.SecondaryAddresses).To(Equal([]string{"GG EDDNYNYX"}))
			Expect(parsedMessage.Originator).To(Equal("BERLINTWR"))
			Expect(parsedMessage.OriginatorDateTime).To(Equal("230714"))
			// fmt.Print(parsedMessage.BodyAndFooter)
			Expect(parsedMessage.BodyAndFooter).To(Equal(`
Q) EDMM/QOATT/IV/BO/A/000/999/4814N01120E005
A) EDDM
B) 2307150600 C) 2307151800
E) AERODROME CONTROL TOWER HOURS OF SERVICE
0600-1800 DUE TO MAINTENANCE
NNNN
`))
		})
	})
	Describe("ParseBody", func() {
		Context("with ARR body", func() {
			parser := NewBodyParser()
			It("should parse the body (ARR-AB123-SSR1234-KJFK-KLAX) correctly", func() {
				body := "(ARR-AB123-SSR1234-KJFK-KLAX)"
				parsedBody, err := parser.Parse(body)
				Expect(err).ToNot(HaveOccurred())
				Expect(parsedBody).ToNot(BeNil())
				Expect(parsedBody).To(BeAssignableToTypeOf(&domain.ARR{}))
				arrMessage := parsedBody.(*domain.ARR)
				Expect(arrMessage.Category).To(Equal("ARR"))
				Expect(arrMessage.AircraftID).To(Equal("AB123"))
				Expect(arrMessage.SSRModeAndCode).To(Equal("SSR1234"))
				Expect(arrMessage.DepartureAirport).To(Equal("KJFK"))
				Expect(arrMessage.ArrivalAirport).To(Equal("KLAX"))
			})

		})

		Context("with DEP body", func() {
			parser := NewBodyParser()
			It("should parse the body (DEP-AB123-SSR1234-KJFK-1500-KLAX) correctly", func() {
				body := "(DEP-AB123-SSR1234-KJFK-1500-KLAX)"
				parsedBody, err := parser.Parse(body)
				Expect(err).ToNot(HaveOccurred())
				Expect(parsedBody).ToNot(BeNil())
				Expect(parsedBody).To(BeAssignableToTypeOf(&domain.DEP{}))
				depMessage := parsedBody.(*domain.DEP)
				Expect(depMessage.Category).To(Equal("DEP"))
				Expect(depMessage.AircraftID).To(Equal("AB123"))
				Expect(depMessage.SSRModeAndCode).To(Equal("SSR1234"))
				Expect(depMessage.DepartureAirport).To(Equal("KJFK"))
				Expect(depMessage.DepartureTime).To(Equal("1500"))
				Expect(depMessage.Destination).To(Equal("KLAX"))
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
				parsedBody, err := parser.Parse(body)
				Expect(err).ToNot(HaveOccurred())
				Expect(parsedBody).ToNot(BeNil())
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
				Expect(fplMessage.OtherInfo).To(Equal("PBN/A1B2B3B4B5D1L1 NAV/ABAS REG/B6513 EET/ZBPE0112 SEL/KMAL PER/C RIF/FRT N640 ZBYN"))
				Expect(fplMessage.SupplementaryInfo).To(Equal("RMK/TCAS EQUIPPED"))
				Expect(fplMessage.PBN).To(Equal("PBN/A1B2B3B4B5D1L1"))
				Expect(fplMessage.EstimatedElapsedTime).To(Equal("ZBPE0112"))
				Expect(fplMessage.SELCALCode).To(Equal("KMAL"))
				Expect(fplMessage.PerformanceCategory).To(Equal("C"))
				Expect(fplMessage.RerouteInformation).To(Equal("FRT N640 ZBYN"))
				Expect(fplMessage.Remarks).To(Equal("TCAS EQUIPPED"))
			})
		})
	})
})
