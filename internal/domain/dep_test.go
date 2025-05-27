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

		DescribeTable("when a mandatory field is missing",
			func(fieldToOmit string, expectedErrorMsgComponent string) {
				invalidDEP := original // Start with a valid one
				switch fieldToOmit {
				case "Category":
					invalidDEP.Category = ""
				case "AircraftID":
					invalidDEP.AircraftID = ""
				case "DepartureAirport":
					invalidDEP.DepartureAirport = ""
				case "DepartureTime":
					invalidDEP.DepartureTime = ""
				case "Destination":
					invalidDEP.Destination = ""
				case "EstimatedElapsedTime":
					invalidDEP.EstimatedElapsedTime = ""
				}
				err := invalidDEP.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(expectedErrorMsgComponent))
			},
			Entry("should fail if Category is missing", "Category", "DEP.Category: telegram category is required"),
			Entry("should fail if AircraftID is missing", "AircraftID", "DEP.AircraftID: aircraft id is required"),
			Entry("should fail if DepartureAirport is missing", "DepartureAirport", "DEP.DepartureAirport: departure airport is required"),
			Entry("should fail if DepartureTime is missing", "DepartureTime", "DEP.DepartureTime: departure time is required"),
			Entry("should fail if Destination is missing", "Destination", "DEP.Destination: destination is required"),
			Entry("should fail if EstimatedElapsedTime is missing", "EstimatedElapsedTime", "DEP.EstimatedElapsedTime: estimated elapsed time is required"),
		)

		It("should validate successfully if optional fields (SSRModeAndCode, AlternateAirport, OtherInfo) are missing", func() {
			validDEP := original
			validDEP.SSRModeAndCode = ""   // Optional
			validDEP.AlternateAirport = "" // Optional
			validDEP.OtherInfo = ""        // Optional
			err := validDEP.Validate()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
