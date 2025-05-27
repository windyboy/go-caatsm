package domain

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ALN", func() {
	var original ALN

	BeforeEach(func() {
		original = ALN{
			Category:           "AFTN",
			AircraftID:         "ABCD1234",
			SSRModeAndCode:     "A1234",
			FlightRulesAndType: "IFR",
			DepartureAirport:   "JFK",
			DepartureTime:      time.Now().Format("150405"), // HHMMSS format
			ArrivalAirport:     "LAX",
			ArrivalTime:        time.Now().Add(5 * time.Hour).Format("150405"), // HHMMSS format
			OtherInfo:          "Test flight",
		}
	})

	Describe("Marshalling and Unmarshalling", func() {
		It("should marshal and unmarshal correctly", func() {
			data, err := json.Marshal(original)
			Expect(err).NotTo(HaveOccurred())

			var unmarshalled ALN
			err = json.Unmarshal(data, &unmarshalled)
			Expect(err).NotTo(HaveOccurred())

			Expect(unmarshalled).To(Equal(original))
		})
	})

	Describe("Validation", func() {
		It("should validate successfully for a valid ALN", func() {
			err := original.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		DescribeTable("when a mandatory field is missing",
			func(fieldToOmit string, expectedErrorMsgComponent string) {
				invalidALN := original // Start with a valid one
				switch fieldToOmit {
				case "Category":
					invalidALN.Category = ""
				case "AircraftID":
					invalidALN.AircraftID = ""
				case "SSRModeAndCode":
					invalidALN.SSRModeAndCode = ""
				case "FlightRulesAndType":
					invalidALN.FlightRulesAndType = ""
				case "DepartureAirport":
					invalidALN.DepartureAirport = ""
				case "DepartureTime":
					invalidALN.DepartureTime = ""
				case "ArrivalAirport":
					invalidALN.ArrivalAirport = ""
				case "ArrivalTime":
					invalidALN.ArrivalTime = ""
				}
				err := invalidALN.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(expectedErrorMsgComponent))
			},
			Entry("should fail if Category is missing", "Category", "ALN.Category: telegram category is required"),
			Entry("should fail if AircraftID is missing", "AircraftID", "ALN.AircraftID: aircraft id is required"),
			Entry("should fail if SSRModeAndCode is missing", "SSRModeAndCode", "ALN.SSRModeAndCode: ssr mode and code is required"),
			Entry("should fail if FlightRulesAndType is missing", "FlightRulesAndType", "ALN.FlightRulesAndType: flight rules and type is required"),
			Entry("should fail if DepartureAirport is missing", "DepartureAirport", "ALN.DepartureAirport: departure airport is required"),
			Entry("should fail if DepartureTime is missing", "DepartureTime", "ALN.DepartureTime: departure time is required"),
			Entry("should fail if ArrivalAirport is missing", "ArrivalAirport", "ALN.ArrivalAirport: arrival airport is required"),
			Entry("should fail if ArrivalTime is missing", "ArrivalTime", "ALN.ArrivalTime: arrival time is required"),
		)

		It("should validate successfully if optional OtherInfo is missing", func() {
			validALN := original
			validALN.OtherInfo = "" // Optional field
			err := validALN.Validate()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
