package parsers

import (
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
			parsedHeader := ParseHeader(message)
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
			parsedHeader := ParseHeader(message)
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
NNNN`

			parsedMessage := ParseHeader(message)
			Expect(parsedMessage.StartIndicator).To(Equal("ZCZC"))
			Expect(parsedMessage.MessageID).To(Equal("NOTAM1234"))
			Expect(parsedMessage.DateTime).To(Equal("230715"))
			Expect(parsedMessage.PriorityIndicator).To(Equal("GG"))
			Expect(parsedMessage.PrimaryAddress).To(Equal("EDDNZEZN"))
			Expect(parsedMessage.SecondaryAddresses).To(Equal([]string{"GG EDDNYNYX"}))
			Expect(parsedMessage.Originator).To(Equal("BERLINTWR"))
			Expect(parsedMessage.OriginatorDateTime).To(Equal("230714"))
			Expect(parsedMessage.BodyAndFooter).To(ContainSubstring("Q) EDMM/QOATT/IV/BO/A/000/999/4814N01120E005\nA) EDDM\nB) 2307150600 C) 2307151800\nE) AERODROME CONTROL TOWER HOURS OF SERVICE\n    0600-1800 DUE TO MAINTENANCE\nNNNN"))
		})
	})
})
