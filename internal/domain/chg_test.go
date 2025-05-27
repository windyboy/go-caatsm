package domain

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CHG", func() {
	var original CHG

	BeforeEach(func() {
		original = CHG{
			Category:             "CHG",
			AircraftID:           "ABCD1234",
			SSRModeAndCode:       "A1234",
			DepartureAirport:     "JFK",
			DepartureTime:        time.Now().Format("150405"), // HHMMSS format
			ArrivalAirport:       "LAX",
			ArrivalTime:          time.Now().Add(5 * time.Hour).Format("150405"), // HHMMSS format
			EstimatedElapsedTime: "0500",                                         // Example time format
			ChangePart:           "Flight plan",
			OtherInfo:            "Test flight",
		}
	})

	Describe("Marshalling and Unmarshalling", func() {
		It("should marshal and unmarshal correctly", func() {
			data, err := json.Marshal(original)
			Expect(err).NotTo(HaveOccurred())

			var unmarshalled CHG
			err = json.Unmarshal(data, &unmarshalled)
			Expect(err).NotTo(HaveOccurred())

			Expect(unmarshalled).To(Equal(original))
		})
	})

	Describe("Validation", func() {
		It("should validate successfully for a valid CHG", func() {
			err := original.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		DescribeTable("when a mandatory field is missing",
			func(fieldToOmit string, expectedErrorMsgComponent string) {
				invalidCHG := original // Start with a valid one
				switch fieldToOmit {
				case "Category":
					invalidCHG.Category = ""
				case "AircraftID":
					invalidCHG.AircraftID = ""
				case "SSRModeAndCode":
					invalidCHG.SSRModeAndCode = ""
				case "DepartureAirport":
					invalidCHG.DepartureAirport = ""
				case "DepartureTime":
					invalidCHG.DepartureTime = ""
				case "ArrivalAirport":
					invalidCHG.ArrivalAirport = ""
				case "ArrivalTime":
					invalidCHG.ArrivalTime = ""
				case "EstimatedElapsedTime":
					invalidCHG.EstimatedElapsedTime = ""
				case "ChangePart":
					invalidCHG.ChangePart = ""
				}
				err := invalidCHG.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(expectedErrorMsgComponent))
			},
			Entry("should fail if Category is missing", "Category", "CHG.Category: category is required"),
			Entry("should fail if AircraftID is missing", "AircraftID", "CHG.AircraftID: aircraft id is required"),
			Entry("should fail if SSRModeAndCode is missing", "SSRModeAndCode", "CHG.SSRModeAndCode: ssr mode and code is required"),
			Entry("should fail if DepartureAirport is missing", "DepartureAirport", "CHG.DepartureAirport: departure airport is required"),
			Entry("should fail if DepartureTime is missing", "DepartureTime", "CHG.DepartureTime: departure time is required"),
			Entry("should fail if ArrivalAirport is missing", "ArrivalAirport", "CHG.ArrivalAirport: arrival airport is required"),
			Entry("should fail if ArrivalTime is missing", "ArrivalTime", "CHG.ArrivalTime: arrival time is required"),
			Entry("should fail if EstimatedElapsedTime is missing", "EstimatedElapsedTime", "CHG.EstimatedElapsedTime: estimated elapsed time is required"),
			Entry("should fail if ChangePart is missing", "ChangePart", "CHG.ChangePart: change part is required"),
		)

		It("should validate successfully if optional fields (AlternateAirport, OtherInfo) are missing", func() {
			validCHG := original
			validCHG.AlternateAirport = "" // Optional
			validCHG.OtherInfo = ""       // Optional
			err := validCHG.Validate()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
