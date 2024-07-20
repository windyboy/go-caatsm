package domain

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DEP", func() {
	var original DEP

	BeforeEach(func() {
		original = DEP{
			Category:             "DEP",
			AircraftID:           "ABCD1234",
			SSRModeAndCode:       "A1234",
			DepartureAirport:     "JFK",
			DepartureTime:        time.Now().Format("150405"), // HHMMSS format
			Destination:          "LAX",
			EstimatedElapsedTime: "0500", // Example format
			OtherInfo:            "Test flight",
		}
	})

	Describe("Marshalling and Unmarshalling", func() {
		It("should marshal and unmarshal correctly", func() {
			data, err := json.Marshal(original)
			Expect(err).NotTo(HaveOccurred())

			var unmarshalled DEP
			err = json.Unmarshal(data, &unmarshalled)
			Expect(err).NotTo(HaveOccurred())

			Expect(unmarshalled).To(Equal(original))
		})
	})

	Describe("Validation", func() {
		It("should validate successfully for a valid DEP", func() {
			err := original.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail validation for missing required fields", func() {
			invalidDEP := DEP{
				Category: "DEP",
				// AircraftID is missing
				DepartureAirport:     "JFK",
				DepartureTime:        "150405",
				Destination:          "LAX",
				EstimatedElapsedTime: "0500",
			}

			err := invalidDEP.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("aircraft id is required"))
		})
	})
})
