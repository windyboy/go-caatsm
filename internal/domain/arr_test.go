package domain

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ARR", func() {
	var original ARR

	BeforeEach(func() {
		original = ARR{
			Category:         "ARR",
			AircraftID:       "ABCD1234",
			SSRModeAndCode:   "A1234",
			DepartureAirport: "JFK",
			DepartureTime:    time.Now().Format("150405"), // HHMMSS format
			ArrivalAirport:   "LAX",
			ArrivalTime:      time.Now().Add(5 * time.Hour).Format("150405"), // HHMMSS format
			OtherInfo:        "Test flight",
		}
	})

	Describe("Marshalling and Unmarshalling", func() {
		It("should marshal and unmarshal correctly", func() {
			data, err := json.Marshal(original)
			Expect(err).NotTo(HaveOccurred())

			var unmarshalled ARR
			err = json.Unmarshal(data, &unmarshalled)
			Expect(err).NotTo(HaveOccurred())

			Expect(unmarshalled).To(Equal(original))
		})
	})

	Describe("Validation", func() {
		It("should validate successfully for a valid ARR", func() {
			err := original.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		DescribeTable("when a mandatory field is missing",
			func(fieldToOmit string, expectedErrorMsgComponent string) {
				invalidARR := original // Start with a valid one
				switch fieldToOmit {
				case "Category":
					invalidARR.Category = ""
				case "AircraftID":
					invalidARR.AircraftID = ""
				case "DepartureAirport":
					invalidARR.DepartureAirport = ""
				case "DepartureTime":
					invalidARR.DepartureTime = ""
				case "ArrivalAirport":
					invalidARR.ArrivalAirport = ""
				case "ArrivalTime":
					invalidARR.ArrivalTime = ""
				}
				err := invalidARR.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(expectedErrorMsgComponent))
			},
			Entry("should fail if Category is missing", "Category", "ARR.Category: category is required"),
			Entry("should fail if AircraftID is missing", "AircraftID", "ARR.AircraftID: aircraft id is required"),
			Entry("should fail if DepartureAirport is missing", "DepartureAirport", "ARR.DepartureAirport: departure airport is required"),
			Entry("should fail if DepartureTime is missing", "DepartureTime", "ARR.DepartureTime: departure time is required"),
			Entry("should fail if ArrivalAirport is missing", "ArrivalAirport", "ARR.ArrivalAirport: arrival airport is required"),
			Entry("should fail if ArrivalTime is missing", "ArrivalTime", "ARR.ArrivalTime: arrival time is required"),
		)

		It("should validate successfully even if optional fields are missing", func() {
			validARR := ARR{ // Only mandatory fields
				Category:         "ARR",
				AircraftID:       "ABCD1234",
				DepartureAirport: "JFK",
				DepartureTime:    time.Now().Format("150405"),
				ArrivalAirport:   "LAX",
				ArrivalTime:      time.Now().Add(5 * time.Hour).Format("150405"),
			}
			// Explicitly make optional fields empty
			validARR.SSRModeAndCode = ""
			validARR.EstimatedElapsedTime = ""
			validARR.AlternateAirport = ""
			validARR.OtherInfo = ""

			err := validARR.Validate()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
