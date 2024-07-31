package parsers

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Schedule Parser", func() {

	Describe("Index Parser", func() {
		Context("parse : 83.", func() {
			message := "83."
			data := parse(message, IndexExpression)
			It("should return a valid index", func() {
				Expect(data).NotTo(BeNil())
				Expect(data[Index]).To(Equal("83."))
			})
		})

		Context("parse : (21)", func() {
			message := "(21)"
			data := parse(message, IndexExpression)
			It("should return a valid index", func() {
				Expect(data).NotTo(BeNil())
				Expect(data[Index]).To(Equal("(21)"))
			})
		})

		Context("parse : L59", func() {
			message := "L59"
			data := parse(message, IndexExpression)
			It("should return a valid index", func() {
				Expect(data).NotTo(BeNil())
				Expect(data[Index]).To(Equal("L59"))
			})
		})

		Context("parse : (205)", func() {
			message := "(205)"
			data := parse(message, IndexExpression)
			It("should return a valid index", func() {
				Expect(data).NotTo(BeNil())
				Expect(data[Index]).To(Equal("(205)"))
			})
		})

		Context("parse : L01", func() {
			message := "L01"
			data := parse(message, IndexExpression)
			It("should return a valid index", func() {
				Expect(data).NotTo(BeNil())
				Expect(data[Index]).To(Equal("L01"))
			})
		})

		Context("parse : 01)", func() {
			message := "01)"
			data := parse(message, IndexExpression)
			It("should return a valid index", func() {
				Expect(data).NotTo(BeNil())
				Expect(data[Index]).To(Equal("01)"))
			})
		})

	})

	Describe("Flight Number Parser", func() {
		Context("parse : FM9134", func() {
			message := "FM9134"
			data := parse(message, FlightNumberExpression)
			It("should return a valid flight number", func() {
				Expect(data).NotTo(BeNil())
				Expect(data[FlightNumber]).To(Equal("FM9134"))
			})
		})

		Context("parse : Y87969", func() {
			message := "Y87969"
			data := parse(message, FlightNumberExpression)
			It("should return a valid flight number", func() {
				Expect(data).NotTo(BeNil())
				Expect(data[FlightNumber]).To(Equal("Y87969"))
			})
		})

		Context("parse : CK261", func() {
			message := "CK261"
			data := parse(message, FlightNumberExpression)
			It("should return a valid flight number", func() {
				Expect(data).NotTo(BeNil())
				Expect(data[FlightNumber]).To(Equal("CK261"))
			})
		})

		Context("parse : 9C8812", func() {
			message := "9C8812"
			data := parse(message, FlightNumberExpression)
			It("should return a valid flight number", func() {
				Expect(data).NotTo(BeNil())
				Expect(data[FlightNumber]).To(Equal("9C8812"))
			})
		})
	})

	Describe("FindWaypoints", func() {
		It("should return the correct waypoints based on the message", func() {
			message := "1845(11JUN)TSN/2100"
			waypoint := ExtractWaypoint(message)
			Expect(waypoint).NotTo(BeNil())
			Expect(waypoint.ArrivalTime).To(Equal("1845(11JUN)"))
			Expect(waypoint.Airport).To(Equal("TSN"))
			Expect(waypoint.DepartureTime).To(Equal("2100"))
		})

		It("should return nil if no waypoints are found", func() {
			message := "18451TSN"
			waypoints := ExtractWaypoint(message)
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
				Expect(len(schedule.Waypoints)).To(Equal(2))
				Expect(schedule.Waypoints[0].Airport).To(Equal("TSN"))
				Expect(schedule.Waypoints[0].DepartureTime).To(Equal("0100"))
				Expect(schedule.Waypoints[1].Airport).To(Equal("SHA"))
				// Expect(schedule.PassengerConfig).To(Equal("1/1"))
				// Expect(schedule.ILS).To(Equal("ILS (00)"))
				// Expect(schedule.DepartureAirport).To(Equal("TSN"))
				// Expect(schedule.DepartureTime).To(Equal("0100"))
				// Expect(schedule.ScheduleInfo).To(Equal("SHA"))
			})

		})
	})
})
