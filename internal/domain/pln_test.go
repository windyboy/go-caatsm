package domain

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ScheduleLine", func() {
	var base ScheduleLine

	BeforeEach(func() {
		base = ScheduleLine{
			Index: "001",
			Date:  "30OCT",
			Task:  "H/G",
			FlightNumber: []string{
				"CA1014",
			},
			AircraftReg:     "B2458",
			PassengerConfig: "1/1",
			ILS:             "ILS(0)",
			Waypoints: []WayPoint{
				{Airport: "ZBTJ", DepartureTime: "0100", ArrivalTime: "0200"},
			},
			Comments: "all green",
		}
	})

	It("passes validation when all required fields exist", func() {
		Expect(base.Validate()).To(Succeed())
	})

	DescribeTable("required field validation",
		func(mutator func(line *ScheduleLine), expected string) {
			line := base
			mutator(&line)
			err := line.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(expected))
		},
		Entry("missing date", func(line *ScheduleLine) {
			line.Date = ""
		}, "date is required"),
		Entry("missing flight number", func(line *ScheduleLine) {
			line.FlightNumber = nil
		}, "flight number is required"),
		Entry("missing aircraft registration", func(line *ScheduleLine) {
			line.AircraftReg = ""
		}, "aircraft registration is required"),
	)
})
