package domain

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DLA", func() {
	var original DLA

	BeforeEach(func() {
		original = DLA{
			Category:             "DLA",
			AircraftID:           "ABCD1234",
			SSRModeAndCode:       "A1234",
			DepartureAirport:     "JFK",
			NewDepartureTime:     time.Now().Add(1 * time.Hour).Format("150405"), // HHMMSS format
			ArrivalAirport:       "LAX",
			EstimatedElapsedTime: "0500", // Example format
			OtherInfo:            "Test flight delay",
		}
	})

	Describe("Marshalling and Unmarshalling", func() {
		It("should marshal and unmarshal correctly", func() {
			data, err := json.Marshal(original)
			Expect(err).NotTo(HaveOccurred())

			var unmarshalled DLA
			err = json.Unmarshal(data, &unmarshalled)
			Expect(err).NotTo(HaveOccurred())

			Expect(unmarshalled).To(Equal(original))
		})
	})

	Describe("Validation", func() {
		It("should validate successfully for a valid DLA", func() {
			err := original.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail validation for missing required fields", func() {
			invalidDLA := DLA{
				Category: "DLA",
				// AircraftID is missing
				SSRModeAndCode:       "A1234",
				DepartureAirport:     "JFK",
				NewDepartureTime:     "150405",
				ArrivalAirport:       "LAX",
				EstimatedElapsedTime: "0500",
			}

			err := invalidDLA.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("aircraft id is required"))
		})
	})
})
