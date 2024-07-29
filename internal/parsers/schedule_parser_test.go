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
})
