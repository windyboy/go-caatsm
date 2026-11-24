package aviation

import (
	"encoding/json"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Aviation Parser", func() {
	Describe("ParseHeader", func() {
		It("cleans text correctly", func() {
			message := `ZCZC TMQ2530 141614
GG ZBTJZXZX
141614 ZSHCZTZX
(ARR-CES5470-ZBTJ-ZSHC1614)
NNNN`
			Expect(cleanMessage(message)).To(Equal("ZCZC TMQ2530 141614\nGG ZBTJZXZX\n141614 ZSHCZTZX\n(ARR-CES5470-ZBTJ-ZSHC1614)"))
		})

		It("parses header with secondary addresses and BEGIN PART", func() {
			message := `
ZCZC TAF6789 160530
QU TSNZPCA
.
QU PEKUDCA TSNUOCA TSNZPCA TSNUFCA
.TAF WSSS 160500Z 1606/1712 20010KT 9999 SCT018
BECMG 1608/1610 24012KT 9999 SCT018

BEGIN PART 02

(FORECAST AMENDMENT)

NNNN`
			parsedHeader, err := ParseHeader(message)
			Expect(err).ToNot(HaveOccurred())
			Expect(parsedHeader.MessageID).To(Equal("TAF6789"))
			Expect(parsedHeader.DateTime).To(Equal("160530"))
			Expect(parsedHeader.PriorityIndicator).To(Equal("QU"))
			Expect(parsedHeader.PrimaryAddress).To(Equal("TSNZPCA"))
			Expect(parsedHeader.SecondaryAddresses).To(Equal(" QU PEKUDCA TSNUOCA TSNZPCA TSNUFCA"))
		})

		It("parses header with dot-prefixed originator", func() {
			message := `
ZCZC NOTAM1122 171000
QU TSNZPCA
.
QU PEKUDCA TSNUOCA TSNZPCA TSNUFCA
.SELOZKE 170999

BEGIN PART 01

(ALERT MESSAGE)

NNNN`
			parsedHeader, err := ParseHeader(message)
			Expect(err).ToNot(HaveOccurred())
			Expect(parsedHeader.MessageID).To(Equal("NOTAM1122"))
			Expect(parsedHeader.Originator).To(Equal("SELOZKE"))
			Expect(parsedHeader.OriginatorDateTime).To(Equal("170999"))
		})
	})

	Describe("BodyParser", func() {
		It("parses ARR without SSR", func() {
			category, body, err := NewBodyParser("(ARR-CES5470-ZBTJ-ZSHC1614)").Parse()
			Expect(err).ToNot(HaveOccurred())
			Expect(category).To(Equal("ARR"))
			arrMsg, ok := body.(*ARR)
			Expect(ok).To(BeTrue())
			Expect(arrMsg.AircraftID).To(Equal("CES5470"))
			Expect(arrMsg.DepartureAirport).To(Equal("ZBTJ"))
			Expect(arrMsg.ArrivalAirport).To(Equal("ZSHC"))
			Expect(arrMsg.ArrivalTime).To(Equal("1614"))
		})

		It("parses ARR with SSR", func() {
			category, body, err := NewBodyParser(" (ARR-JAE7433/A0132-RKSI-ZBTJ1604)").Parse()
			Expect(err).ToNot(HaveOccurred())
			Expect(category).To(Equal("ARR"))
			arrMsg := body.(*ARR)
			Expect(arrMsg.SSRModeAndCode).To(Equal("A0132"))
		})

		It("parses DEP body", func() {
			category, body, err := NewBodyParser("(DEP-CYZ9017/A5633-ZBTJ1638-ZSPD)").Parse()
			Expect(err).ToNot(HaveOccurred())
			Expect(category).To(Equal("DEP"))
			depMsg := body.(*DEP)
			Expect(depMsg.AircraftID).To(Equal("CYZ9017"))
			Expect(depMsg.DepartureTime).To(Equal("1638"))
			Expect(depMsg.Destination).To(Equal("ZSPD"))
		})

		It("parses CNL body", func() {
			category, body, err := NewBodyParser("(CNL-YZR7979-ZSPD-ZBTJ)").Parse()
			Expect(err).ToNot(HaveOccurred())
			Expect(category).To(Equal("CNL"))
			cnlMsg := body.(*CNL)
			Expect(cnlMsg.AircraftID).To(Equal("YZR7979"))
			Expect(cnlMsg.DestinationAirport).To(Equal("ZBTJ"))
		})

		It("parses DLA body", func() {
			category, body, err := NewBodyParser("(DLA-CSN3133-ZGGG0110-ZBTJ)").Parse()
			Expect(err).ToNot(HaveOccurred())
			Expect(category).To(Equal("DLA"))
			dlaMsg := body.(*DLA)
			Expect(dlaMsg.AircraftID).To(Equal("CSN3133"))
			Expect(dlaMsg.NewDepartureTime).To(Equal("0110"))
		})

		It("parses FPL body", func() {
			rawBody := `(FPL-CCA1532-IS
-A332/H
-SDE3FGHIJ4J5M1RWY/LB101
-ZSSS2035
-K0859S1040 PIAKS G330 PIMOL A539 BTO W82 DOGAR
-ZBAA0153 ZBYN
-PBN/A1B2B3B4B5D1L1 NAV/ABAS REG/B6513 EET/ZBPE0112 SEL/KMAL PER/C RIF/FRT N640 ZBYN RMK/TCAS EQUIPPED)`
			category, body, err := NewBodyParser(rawBody).Parse()
			Expect(err).ToNot(HaveOccurred())
			Expect(category).To(Equal("FPL"))
			fplMsg := body.(*FPL)
			Expect(fplMsg.FlightNumber).To(Equal("CCA1532"))
			Expect(fplMsg.PBN).To(Equal("A1B2B3B4B5D1L1"))
			Expect(fplMsg.EstimatedElapsedTime).To(Equal("ZBPE0112"))
			Expect(fplMsg.SELCALCode).To(Equal("KMAL"))
			Expect(fplMsg.PerformanceCategory).To(Equal("C"))
			Expect(fplMsg.RerouteInformation).To(Equal("FRT N640 ZBYN"))
			Expect(fplMsg.Remarks).To(Equal("TCAS EQUIPPED"))
		})
	})

	Describe("Parse", func() {
		It("parses a complete ARR message", func() {
			msg := `
ZCZC TMQ2526 141605
FF ZBTJZPZX
141604 ZBACZQZX
(ARR-JAE7433/A0132-RKSI-ZBTJ1604)
NNNN`
			res, err := Parse(msg)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).NotTo(BeNil())
			Expect(res.Parsed).To(BeTrue())
			Expect(res.Status).To(Equal(StatusParsed))
			Expect(res.Header.MessageID).To(Equal("TMQ2526"))
			Expect(res.Header.Originator).To(Equal("ZBACZQZX"))
			Expect(res.BodyData).To(BeAssignableToTypeOf(&ARR{}))
		})

		It("parses a complete FPL message with sparse Item 18 fields", func() {
			msg := `ZCZC TMQ2617 142150
GG ZBTJZPZX
150551 ZBTJUOBK
(FPL-OKA2861-IS
-MA60/M-SHID/C
-ZBTJ0030
-K0420S0450 CG J1 FZ
-ZSYT0100  ZSQD ZYTL
-REG/B3710 SEL/ RMK/TCAS )
NNNN`
			res, err := Parse(msg)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Parsed).To(BeTrue())
			fplMsg := res.BodyData.(*FPL)
			Expect(fplMsg.FlightNumber).To(Equal("OKA2861"))
			Expect(fplMsg.Register).To(Equal("B3710"))
			Expect(fplMsg.Remarks).To(Equal("TCAS"))
		})

		It("returns non-nil Result with StatusHeaderError on oversized input", func() {
			res, err := Parse(strings.Repeat("A", MaxTelegramSize+1))
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ErrHeaderParse))
			Expect(res).NotTo(BeNil())
			Expect(res.Parsed).To(BeFalse())
			Expect(res.Status).To(Equal(StatusHeaderError))
		})

		It("returns non-nil Result with StatusBodyError when body fails", func() {
			msg := "ZCZC TMQ2526 141605\nFF ZBTJZPZX\n141604 ZBACZQZX\n(INVALID-BODY)\nNNNN"
			res, err := Parse(msg)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ErrBodyParse))
			Expect(res).NotTo(BeNil())
			Expect(res.Parsed).To(BeFalse())
			Expect(res.Status).To(Equal(StatusBodyError))
		})
	})

	Describe("Models JSON & Validate", func() {
		It("marshals and validates ARR, DEP, CNL, DLA, FPL", func() {
			arr := ARR{Category: "ARR", AircraftID: "CCA123", DepartureAirport: "ZBAA", ArrivalAirport: "ZSSS", ArrivalTime: "1200"}
			Expect(arr.Validate()).To(Succeed())

			dep := DEP{Category: "DEP", AircraftID: "CCA123", DepartureAirport: "ZBAA", DepartureTime: "1000", Destination: "ZSSS"}
			Expect(dep.Validate()).To(Succeed())

			cnl := CNL{Category: "CNL", AircraftID: "CCA123", DepartureAirport: "ZBAA", DestinationAirport: "ZSSS"}
			Expect(cnl.Validate()).To(Succeed())

			dla := DLA{Category: "DLA", AircraftID: "CCA123", DepartureAirport: "ZBAA", ArrivalAirport: "ZSSS", ArrivalTime: "1300"}
			Expect(dla.Validate()).To(Succeed())
			blob, err := json.Marshal(dla)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(blob)).To(ContainSubstring(`"arrival_time":"1300"`))

			fpl := FPL{
				Category:                "FPL",
				FlightNumber:            "CCA123",
				AircraftID:              "B738/M",
				FlightRulesAndType:      "IS",
				CruisingSpeedAndLevel:   "K0850S1040",
				DepartureAirport:        "ZBAA",
				DepartureTime:           "0900",
				Route:                   "DCT",
				DestinationAndTotalTime: "ZSSS0200",
			}
			Expect(fpl.Validate()).To(Succeed())

			invalidFPL := fpl
			invalidFPL.FlightNumber = ""
			Expect(invalidFPL.Validate()).To(HaveOccurred())
		})
	})
})
