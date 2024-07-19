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

		It("should fail validation for missing required fields", func() {
			invalidARR := ARR{
				Category: "ARR",
				// AircraftID is missing
				DepartureAirport: "JFK",
				DepartureTime:    "150405",
				ArrivalAirport:   "LAX",
				ArrivalTime:      "180405",
			}

			err := invalidARR.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("aircraft id is required"))
		})
	})
})
