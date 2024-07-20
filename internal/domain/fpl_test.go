package domain

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("FPL", func() {
	var original FPL

	BeforeEach(func() {
		original = FPL{
			Category:                "FPL",
			FlightNumber:            "AB123",
			ReferenceData:           "Ref123", // Example reference data
			AircraftID:              "ABCD1234",
			SSRModeAndCode:          "A1234",
			FlightRulesAndType:      "IFR",
			CruisingSpeedAndLevel:   "N0450F350",
			DepartureAirport:        "JFK",
			DepartureTime:           time.Now().Format("150405"), // HHMMSS format
			Route:                   "DCT GAYEL J95 BUF DCT",
			DestinationAndTotalTime: "LAX0500", // Example format
			AlternateAirport:        "SFO",     // Example alternate airport
			OtherInfo:               "Test flight",
			SupplementaryInfo:       "Supplementary information",
			EstimatedArrivalTime:    "0153", // Example estimated arrival time
			PBN:                     "A1B2B3B4B5D1L1",
			NavigationEquipment:     "NAV/ABAS",
			EstimatedElapsedTime:    "EET/ZBPE0112",
			SELCALCode:              "KMAL",
			PerformanceCategory:     "C",
			RerouteInformation:      "RIF/FRT N640 ZBYN", // Example reroute information
			Remarks:                 "RMK/TCAS EQUIPPED", // Example remarks
		}
	})

	Describe("Marshalling and Unmarshalling", func() {
		It("should marshal and unmarshal correctly", func() {
			data, err := json.Marshal(original)
			Expect(err).NotTo(HaveOccurred())

			var unmarshalled FPL
			err = json.Unmarshal(data, &unmarshalled)
			Expect(err).NotTo(HaveOccurred())

			Expect(unmarshalled).To(Equal(original))
		})
	})

	Describe("Validation", func() {
		It("should validate successfully for a valid FPL", func() {
			err := original.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail validation for missing required fields", func() {
			invalidFPL := FPL{
				Category: "FPL",
				// FlightNumber is missing
				AircraftID:              "ABCD1234",
				SSRModeAndCode:          "A1234",
				FlightRulesAndType:      "IFR",
				CruisingSpeedAndLevel:   "N0450F350",
				DepartureAirport:        "JFK",
				DepartureTime:           "150405",
				Route:                   "DCT GAYEL J95 BUF DCT",
				DestinationAndTotalTime: "LAX0500",
			}

			err := invalidFPL.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("flight number is required"))
		})
	})
})
