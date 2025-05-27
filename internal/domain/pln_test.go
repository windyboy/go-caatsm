package domain_test

import (
	"caatsm/internal/domain"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPln(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PlnDomain Suite") // Changed Suite name for clarity
}

var _ = Describe("ScheduleLine", func() { // Changed Describe to be more specific
	Describe("Validate", func() {
		var sl domain.ScheduleLine

		BeforeEach(func() {
			sl = domain.ScheduleLine{
				Date:         "30OCT2024", // Date is a string
				FlightNumber: []string{"FN123"},
				AircraftReg:  "REG123",
				Waypoints: []domain.WayPoint{ // Waypoint, not Waypoint
					{Airport: "WPT1", ArrivalTime: "1030", DepartureTime: "1035"},
					{Airport: "WPT2", ArrivalTime: "1100", DepartureTime: "1105"},
				},
				// Optional fields
				Index:           "1",
				Task:            "H/G",
				PassengerConfig: "C12Y150",
				ILS:             "CATII",
				Comments:        "Test comment",
				Reference:       "Test ref",
			}
		})

		Context("when all mandatory fields are present", func() {
			It("should return nil", func() {
				Expect(sl.Validate()).To(BeNil())
			})
		})

		Context("when Date is missing", func() {
			It("should return an error", func() {
				sl.Date = ""
				Expect(sl.Validate()).To(MatchError("date is required"))
			})
		})

		Context("when FlightNumber slice is empty", func() {
			It("should return an error", func() {
				sl.FlightNumber = []string{}
				Expect(sl.Validate()).To(MatchError("flight number is required"))
			})
		})
        
		Context("when FlightNumber slice is nil", func() {
			It("should return an error", func() {
				sl.FlightNumber = nil
				Expect(sl.Validate()).To(MatchError("flight number is required"))
			})
		})

		Context("when AircraftReg is missing", func() {
			It("should return an error", func() {
				sl.AircraftReg = ""
				// Based on pln.go, error is "aircraft registration is required"
				Expect(sl.Validate()).To(MatchError("aircraft registration is required"))
			})
		})

		// Note: Current ScheduleLine.Validate() in pln.go does not validate:
		// - If strings within FlightNumber slice are empty.
		// - The contents or presence of Waypoints.
		// Tests for these would be added if ScheduleLine.Validate() is enhanced.
	})
})
