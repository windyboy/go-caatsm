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

		DescribeTable("when a mandatory field is missing",
			func(fieldToOmit string, expectedErrorMsgComponent string) {
				invalidFPL := original // Start with a valid one
				switch fieldToOmit {
				case "Category":
					invalidFPL.Category = ""
				case "FlightNumber":
					invalidFPL.FlightNumber = ""
				case "AircraftID":
					invalidFPL.AircraftID = ""
				case "SSRModeAndCode":
					invalidFPL.SSRModeAndCode = ""
				case "FlightRulesAndType":
					invalidFPL.FlightRulesAndType = ""
				case "CruisingSpeedAndLevel":
					invalidFPL.CruisingSpeedAndLevel = ""
				case "DepartureAirport":
					invalidFPL.DepartureAirport = ""
				case "DepartureTime":
					invalidFPL.DepartureTime = ""
				case "Route":
					invalidFPL.Route = ""
				case "DestinationAndTotalTime":
					invalidFPL.DestinationAndTotalTime = ""
				case "EstimatedArrivalTime":
					invalidFPL.EstimatedArrivalTime = ""
				case "PBN":
					invalidFPL.PBN = ""
				case "NavigationEquipment":
					invalidFPL.NavigationEquipment = ""
				case "EstimatedElapsedTime":
					invalidFPL.EstimatedElapsedTime = ""
				case "SELCALCode":
					invalidFPL.SELCALCode = ""
				case "PerformanceCategory":
					invalidFPL.PerformanceCategory = ""
				}
				err := invalidFPL.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(expectedErrorMsgComponent))
			},
			Entry("should fail if Category is missing", "Category", "FPL.Category: category is required"),
			Entry("should fail if FlightNumber is missing", "FlightNumber", "FPL.FlightNumber: flight number is required"),
			Entry("should fail if AircraftID is missing", "AircraftID", "FPL.AircraftID: aircraft id is required"),
			Entry("should fail if SSRModeAndCode is missing", "SSRModeAndCode", "FPL.SSRModeAndCode: SSR mode and code is required"),
			Entry("should fail if FlightRulesAndType is missing", "FlightRulesAndType", "FPL.FlightRulesAndType: flight rules and type is required"),
			Entry("should fail if CruisingSpeedAndLevel is missing", "CruisingSpeedAndLevel", "FPL.CruisingSpeedAndLevel: cruising speed and level is required"),
			Entry("should fail if DepartureAirport is missing", "DepartureAirport", "FPL.DepartureAirport: departure airport is required"),
			Entry("should fail if DepartureTime is missing", "DepartureTime", "FPL.DepartureTime: departure time is required"),
			Entry("should fail if Route is missing", "Route", "FPL.Route: route is required"),
			Entry("should fail if DestinationAndTotalTime is missing", "DestinationAndTotalTime", "FPL.DestinationAndTotalTime: destination and total time is required"),
			Entry("should fail if EstimatedArrivalTime is missing", "EstimatedArrivalTime", "FPL.EstimatedArrivalTime: estimated arrival time is required"),
			Entry("should fail if PBN is missing", "PBN", "FPL.PBN: PBN information is required"),
			Entry("should fail if NavigationEquipment is missing", "NavigationEquipment", "FPL.NavigationEquipment: navigation equipment is required"),
			Entry("should fail if EstimatedElapsedTime (Field 18) is missing", "EstimatedElapsedTime", "FPL.EstimatedElapsedTime: estimated elapsed time (EET) is required"),
			Entry("should fail if SELCALCode is missing", "SELCALCode", "FPL.SELCALCode: SELCAL code is required"),
			Entry("should fail if PerformanceCategory is missing", "PerformanceCategory", "FPL.PerformanceCategory: performance category is required"),
		)

		It("should validate successfully if optional fields are missing", func() {
			// Create an FPL with only mandatory fields based on FPL.Validate()
			// Note: This is a long list of "mandatory" fields for FPL.
			validFPL := FPL{
				Category:                "FPL",
				FlightNumber:            "VALID123",
				AircraftID:              "VALIDAC",
				SSRModeAndCode:          "VALSSR",
				FlightRulesAndType:      "VFR",
				CruisingSpeedAndLevel:   "N0100F100",
				DepartureAirport:        "AAAA",
				DepartureTime:           "1000",
				Route:                   "DCT",
				DestinationAndTotalTime: "BBBB0100",
				EstimatedArrivalTime:    "1100", // Derived from Dest+EET usually, but a separate field in struct
				PBN:                     "PBNINFO",
				NavigationEquipment:     "NAVINFO",
				EstimatedElapsedTime:    "EETINFO", // This is from OtherInfo (Field 18)
				SELCALCode:              "SELINFO",
				PerformanceCategory:     "A",
			}
			// Optional fields: ReferenceData, AlternateAirport, OtherInfo, SupplementaryInfo, Register, RerouteInformation, Remarks
			validFPL.ReferenceData = ""
			validFPL.AlternateAirport = ""
			validFPL.OtherInfo = "" // If this is empty, sub-fields like PBN, NAV etc. would not be parsable.
			                         // The validation logic implies PBN, NAV etc. are direct fields, not just from OtherInfo string.
			                         // For this test, if PBN etc. are mandatory, OtherInfo might implicitly be required to contain them.
			                         // However, the struct has them as separate fields.
			                         // Let's assume OtherInfo itself can be empty, but if PBN etc. are direct fields, they must be filled.
			validFPL.SupplementaryInfo = ""
			validFPL.Register = ""
			validFPL.RerouteInformation = ""
			validFPL.Remarks = ""

			err := validFPL.Validate()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
