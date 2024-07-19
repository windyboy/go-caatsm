package domain

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CPL", func() {
	var original CPL

	BeforeEach(func() {
		original = CPL{
			Category:                "CPL",
			AircraftID:              "ABCD1234",
			SSRModeAndCode:          "A1234",
			FlightRulesAndType:      "IFR",
			AircraftAndEquipment:    "B738",
			CruisingSpeedAndLevel:   "N0450F350",
			DepartureAirport:        "JFK",
			DepartureTime:           time.Now().Format("150405"), // HHMMSS format
			Route:                   "DCT GAYEL J95 BUF DCT",
			DestinationAndTotalTime: "LAX0500", // Example format
			OtherInfo:               "Test flight",
		}
	})

	Describe("Marshalling and Unmarshalling", func() {
		It("should marshal and unmarshal correctly", func() {
			data, err := json.Marshal(original)
			Expect(err).NotTo(HaveOccurred())

			var unmarshalled CPL
			err = json.Unmarshal(data, &unmarshalled)
			Expect(err).NotTo(HaveOccurred())

			Expect(unmarshalled).To(Equal(original))
		})
	})

	Describe("Validation", func() {
		It("should validate successfully for a valid CPL", func() {
			err := original.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail validation for missing required fields", func() {
			invalidCPL := CPL{
				Category: "CPL",
				// AircraftID is missing
				SSRModeAndCode:          "A1234",
				FlightRulesAndType:      "IFR",
				AircraftAndEquipment:    "B738",
				CruisingSpeedAndLevel:   "N0450F350",
				DepartureAirport:        "JFK",
				DepartureTime:           "150405",
				Route:                   "DCT GAYEL J95 BUF DCT",
				DestinationAndTotalTime: "LAX0500",
			}

			err := invalidCPL.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("aircraft id is required"))
		})
	})
})
