package parsers

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Schedule Parser", func() {

	Describe("FindWaypoints", func() {
		It("should return the correct waypoints based on the message", func() {
			message := "1845(11JUN)TSN/2100"
			waypoints := FindWaypoints(message)
			Expect(waypoints).NotTo(BeNil())
			Expect(waypoints[ArrivalTime]).To(Equal("1845(11JUN)"))
			Expect(waypoints[AirportCode]).To(Equal("TSN"))
			Expect(waypoints[DepartureTime]).To(Equal("2100"))
		})

		It("should return nil if no waypoints are found", func() {
			message := "1845TSN"
			waypoints := FindWaypoints(message)
			Expect(waypoints).To(BeNil())
		})
	})

	Describe("Parsing one line of schedule", func() {
		Context("parse : W/Z FM9134 B2688 1/1ILS (00) TSN0100 SHA", func() {
			lineText := "W/Z FM9134 B2688 1/1ILS (00) TSN0100 SHA"
			schedule := ParseLine(lineText)
			It("should return a valid schedule", func() {
				Expect(schedule).NotTo(BeNil())
				Expect(schedule.Task).To(Equal("W/Z"))
				// Expect(schedule.Date).To(Equal("TSN0100"))
				// Expect(schedule.Task).To(Equal("1/1"))
				Expect(schedule.FlightNumber).To(Equal("FM9134"))
				Expect(schedule.AircraftReg).To(Equal("B2688"))
				// Expect(schedule.PassengerConfig).To(Equal("1/1"))
				// Expect(schedule.ILS).To(Equal("ILS (00)"))
				// Expect(schedule.DepartureAirport).To(Equal("TSN"))
				// Expect(schedule.DepartureTime).To(Equal("0100"))
				// Expect(schedule.ScheduleInfo).To(Equal("SHA"))
			})

		})
	})
})
