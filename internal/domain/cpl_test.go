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

		DescribeTable("when a mandatory field is missing",
			func(fieldToOmit string, expectedErrorMsgComponent string) {
				invalidCPL := original // Start with a valid one
				switch fieldToOmit {
				case "Category":
					invalidCPL.Category = ""
				case "AircraftID":
					invalidCPL.AircraftID = ""
				case "SSRModeAndCode":
					invalidCPL.SSRModeAndCode = ""
				case "FlightRulesAndType":
					invalidCPL.FlightRulesAndType = ""
				case "AircraftAndEquipment":
					invalidCPL.AircraftAndEquipment = ""
				case "CruisingSpeedAndLevel":
					invalidCPL.CruisingSpeedAndLevel = ""
				case "DepartureAirport":
					invalidCPL.DepartureAirport = ""
				case "DepartureTime":
					invalidCPL.DepartureTime = ""
				case "Route":
					invalidCPL.Route = ""
				case "DestinationAndTotalTime":
					invalidCPL.DestinationAndTotalTime = ""
				}
				err := invalidCPL.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(expectedErrorMsgComponent))
			},
			Entry("should fail if Category is missing", "Category", "CPL.Category: category is required"),
			Entry("should fail if AircraftID is missing", "AircraftID", "CPL.AircraftID: aircraft id is required"),
			Entry("should fail if SSRModeAndCode is missing", "SSRModeAndCode", "CPL.SSRModeAndCode: ssr mode and code is required"),
			Entry("should fail if FlightRulesAndType is missing", "FlightRulesAndType", "CPL.FlightRulesAndType: flight rules and type is required"),
			Entry("should fail if AircraftAndEquipment is missing", "AircraftAndEquipment", "CPL.AircraftAndEquipment: aircraft and equipment is required"),
			Entry("should fail if CruisingSpeedAndLevel is missing", "CruisingSpeedAndLevel", "CPL.CruisingSpeedAndLevel: cruising speed and level is required"),
			Entry("should fail if DepartureAirport is missing", "DepartureAirport", "CPL.DepartureAirport: departure airport is required"),
			Entry("should fail if DepartureTime is missing", "DepartureTime", "CPL.DepartureTime: departure time is required"),
			Entry("should fail if Route is missing", "Route", "CPL.Route: route is required"),
			Entry("should fail if DestinationAndTotalTime is missing", "DestinationAndTotalTime", "CPL.DestinationAndTotalTime: destination and total time is required"),
		)

		It("should validate successfully if optional fields (AlternateAirport, OtherInfo) are missing", func() {
			validCPL := original
			validCPL.AlternateAirport = "" // Optional
			validCPL.OtherInfo = ""       // Optional
			err := validCPL.Validate()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
