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
			SSRModeAndCode:       "A1234", // Optional
			DepartureAirport:     "JFK",
			NewDepartureTime:     time.Now().Add(1 * time.Hour).Format("150405"), // Mandatory
			ArrivalAirport:       "LAX",
			EstimatedElapsedTime: "0500", // Mandatory (formerly ArrivalTime)
			OtherInfo:            "Test flight delay", // Optional
		}
	})

	Describe("Marshalling and Unmarshalling", func() {
		It("should marshal and unmarshal correctly", func() {
			// Ensure all fields, especially the renamed one, are part of the test
			testDLA := DLA{
				Category:             "DLA",
				AircraftID:           "XYZ789",
				SSRModeAndCode:       "S7700",
				DepartureAirport:     "MIA",
				NewDepartureTime:     "100000",
				ArrivalAirport:       "ATL",
				EstimatedElapsedTime: "0230",
				OtherInfo:            "Delayed due to frogs",
			}
			data, err := json.Marshal(testDLA)
			Expect(err).NotTo(HaveOccurred())

			var unmarshalled DLA
			err = json.Unmarshal(data, &unmarshalled)
			Expect(err).NotTo(HaveOccurred())
			Expect(unmarshalled).To(Equal(testDLA))
		})
	})

	Describe("Validation", func() {
		It("should validate successfully for a valid DLA", func() {
			err := original.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		DescribeTable("when a mandatory field is missing",
			func(fieldToOmit string, expectedErrorMsgComponent string) {
				invalidDLA := original // Start with a valid one
				switch fieldToOmit {
				case "Category":
					invalidDLA.Category = ""
				case "AircraftID":
					invalidDLA.AircraftID = ""
				case "DepartureAirport":
					invalidDLA.DepartureAirport = ""
				case "NewDepartureTime":
					invalidDLA.NewDepartureTime = ""
				case "ArrivalAirport":
					invalidDLA.ArrivalAirport = ""
				case "EstimatedElapsedTime":
					invalidDLA.EstimatedElapsedTime = ""
				}
				err := invalidDLA.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(expectedErrorMsgComponent))
			},
			Entry("should fail if Category is missing", "Category", "DLA.Category: category is required"),
			Entry("should fail if AircraftID is missing", "AircraftID", "DLA.AircraftID: aircraft id is required"),
			Entry("should fail if DepartureAirport is missing", "DepartureAirport", "DLA.DepartureAirport: departure airport is required"),
			Entry("should fail if NewDepartureTime is missing", "NewDepartureTime", "DLA.NewDepartureTime: new departure time is required"),
			Entry("should fail if ArrivalAirport is missing", "ArrivalAirport", "DLA.ArrivalAirport: arrival airport is required"),
			Entry("should fail if EstimatedElapsedTime is missing", "EstimatedElapsedTime", "DLA.EstimatedElapsedTime: estimated elapsed time is required"),
		)

		It("should validate successfully if optional fields (SSRModeAndCode, OtherInfo) are missing", func() {
			validDLA := original
			validDLA.SSRModeAndCode = "" // Optional
			validDLA.OtherInfo = ""       // Optional
			err := validDLA.Validate()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
