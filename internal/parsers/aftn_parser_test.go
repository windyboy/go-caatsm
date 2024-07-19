package parsers

import (
	"caatsm/internal/domain"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AFTN Parser", func() {
	Describe("Parse", func() {
		It("should parse ARR messages correctly", func() {
			message := "(ARR-AB123-SSR1234-KJFK-KLAX)"
			parser := AFTNParser{}
			parsedMessage, err := parser.Parse(message)
			Expect(err).NotTo(HaveOccurred())
			Expect(parsedMessage).To(BeAssignableToTypeOf(&domain.ARR{}))
			arrMessage := parsedMessage.(*domain.ARR)
			Expect(arrMessage.Category).To(Equal("ARR"))
			Expect(arrMessage.AircraftID).To(Equal("AB123"))
			Expect(arrMessage.SSRModeAndCode).To(Equal("SSR1234"))
			Expect(arrMessage.DepartureAirport).To(Equal("KJFK"))
			Expect(arrMessage.ArrivalAirport).To(Equal("KLAX"))
		})

		It("should parse DEP messages correctly", func() {
			message := "(DEP-AB123-SSR1234-KJFK-1500-KLAX)"
			parser := AFTNParser{}
			parsedMessage, err := parser.Parse(message)
			Expect(err).NotTo(HaveOccurred())
			Expect(parsedMessage).To(BeAssignableToTypeOf(&domain.DEP{}))
			depMessage := parsedMessage.(*domain.DEP)
			Expect(depMessage.AircraftID).To(Equal("AB123"))
			Expect(depMessage.SSRModeAndCode).To(Equal("SSR1234"))
			Expect(depMessage.DepartureAirport).To(Equal("KJFK"))
			Expect(depMessage.DepartureTime).To(Equal("1500"))
			Expect(depMessage.Destination).To(Equal("KLAX"))
		})

		It("should parse FPL messages correctly", func() {
			message := `(FPL-CCA1532-IS
-A332/H
-SDE3FGHIJ4J5M1RWY/LB101
-ZSSS2035
-K0859S1040 PIAKS G330 PIMOL A539 BTO W82 DOGAR
-ZBAA0153 ZBYN
-PBN/A1B2B3B4B5D1L1 NAV/ABAS REG/B6513 EET/ZBPE0112 SEL/KMAL PER/C RIF/FRT N640 ZBYN RMK/TCAS EQUIPPED)`
			parser := AFTNParser{}
			parsedMessage, err := parser.Parse(message)
			Expect(err).NotTo(HaveOccurred())
			Expect(parsedMessage).To(BeAssignableToTypeOf(&domain.FPL{}))
			fplMessage := parsedMessage.(*domain.FPL)
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

		})

		It("should return an error for invalid message types", func() {
			message := "(XYZ-AB123-SSR1234-KJFK-KLAX)"
			parser := AFTNParser{}
			_, err := parser.Parse(message)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("invalid message type"))
		})
	})

	Describe("ParseAFTN", func() {
		It("should parse a valid AFTN message", func() {
			rawMessage := `ZCZC TMQ2611 151524
FF SENDERAA
151524 RECEIVERAA
(ARR-AB123-SSR1234-KJFK-KLAX)`

			aftnMessage, err := ParseAFTN(rawMessage)
			Expect(err).NotTo(HaveOccurred())
			Expect(aftnMessage).NotTo(BeNil())
			Expect(aftnMessage.Header.StartSignal).To(Equal("ZCZC"))
			Expect(aftnMessage.Header.SendID).To(Equal("TMQ2611"))
			Expect(aftnMessage.Header.SendTime).To(Equal("151524"))
			Expect(aftnMessage.Category).To(Equal("ARR"))
		})

		It("should return an error for invalid AFTN message format", func() {
			rawMessage := `ZCZC
TMQ2611
151524`

			_, err := ParseAFTN(rawMessage)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("invalid AFTN message format"))
		})
	})

	Describe("ValidateAFTN", func() {
		It("should validate a valid AFTN message", func() {
			aftnMessage := &domain.AFTN{
				Header: domain.Header{
					StartSignal: "ZCZC",
					SendID:      "TMQ2611",
					SendTime:    "151524",
				},
				PriorityAndSender: domain.PriorityAndSender{
					Priority: "FF",
					Sender:   "SENDERAA",
				},
				TimeAndReceiver: domain.TimeAndReceiver{
					Time:     "151524",
					Receiver: "RECEIVAA",
				},
				Category: "ARR",
			}
			err := ValidateAFTN(aftnMessage)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return an error for missing required fields", func() {
			aftnMessage := &domain.AFTN{
				Header: domain.Header{
					StartSignal: "",
					SendID:      "TMQ2611",
					SendTime:    "151524",
				},
				PriorityAndSender: domain.PriorityAndSender{
					Priority: "FF",
					Sender:   "SENDERAA",
				},
				TimeAndReceiver: domain.TimeAndReceiver{
					Time:     "151524",
					Receiver: "RECEIVAA",
				},
				Category: "ARR",
			}
			err := ValidateAFTN(aftnMessage)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("invalid AFTN message: missing fields"))
		})

		It("should return an error for invalid priority code", func() {
			aftnMessage := &domain.AFTN{
				Header: domain.Header{
					StartSignal: "ZCZC",
					SendID:      "TMQ2611",
					SendTime:    "151524",
				},
				PriorityAndSender: domain.PriorityAndSender{
					Priority: "ZZ",
					Sender:   "SENDERAA",
				},
				TimeAndReceiver: domain.TimeAndReceiver{
					Time:     "151524",
					Receiver: "RECEIVAA",
				},
				Category: "ARR",
			}
			err := ValidateAFTN(aftnMessage)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("invalid priority code"))
		})

		It("should return an error for invalid address format", func() {
			aftnMessage := &domain.AFTN{
				Header: domain.Header{
					StartSignal: "ZCZC",
					SendID:      "TMQ2611",
					SendTime:    "151524",
				},
				PriorityAndSender: domain.PriorityAndSender{
					Priority: "FF",
					Sender:   "INVALID",
				},
				TimeAndReceiver: domain.TimeAndReceiver{
					Time:     "151524",
					Receiver: "INVALID",
				},
				Category: "ARR",
			}
			err := ValidateAFTN(aftnMessage)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("invalid address format"))
		})
	})
})
