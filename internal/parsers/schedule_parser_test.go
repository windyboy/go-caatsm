package parsers

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Schedule Parser", func() {

	Describe("Index Parser", func() {
		Context("parse : 83.", func() {
			It("should return a valid index", func() {
				message := "83."
				data := extract(message, IndexExpression)
				Expect(data).NotTo(BeNil())
				Expect(data[Index]).To(Equal("83."))
			})
		})

		Context("parse : (21)", func() {
			It("should return a valid index", func() {
				message := "(21)"
				data := extract(message, IndexExpression)
				Expect(data).NotTo(BeNil())
				Expect(data[Index]).To(Equal("(21)"))
			})
		})

		Context("parse : L59", func() {
			It("should return a valid index", func() {
				message := "L59"
				data := extract(message, IndexExpression)
				Expect(data).NotTo(BeNil())
				Expect(data[Index]).To(Equal("L59"))
			})
		})

		Context("parse : (205)", func() {
			It("should return a valid index", func() {
				message := "(205)"
				data := extract(message, IndexExpression)
				Expect(data).NotTo(BeNil())
				Expect(data[Index]).To(Equal("(205)"))
			})
		})

		Context("parse : L01", func() {
			It("should return a valid index", func() {
				message := "L01"
				data := extract(message, IndexExpression)
				Expect(data).NotTo(BeNil())
				Expect(data[Index]).To(Equal("L01"))
			})
		})

		Context("parse : 01)", func() {
			It("should return a valid index", func() {
				message := "01)"
				data := extract(message, IndexExpression)
				Expect(data).NotTo(BeNil())
				Expect(data[Index]).To(Equal("01)"))
			})
		})

	})

	Describe("Flight Number Parser", func() {
		Context("parse : FM9134", func() {
			It("should return a valid flight number", func() {
				message := "FM9134"
				data := extract(message, FlightNumberExpression)
				Expect(data).NotTo(BeNil())
				Expect(data[FlightNumber]).To(Equal("FM9134"))
			})
		})

		Context("parse : Y87969", func() {
			It("should return a valid flight number", func() {
				message := "Y87969"
				data := extract(message, FlightNumberExpression)
				Expect(data).NotTo(BeNil())
				Expect(data[FlightNumber]).To(Equal("Y87969"))
			})
		})

		Context("parse : CK261", func() {
			It("should return a valid flight number", func() {
				message := "CK261"
				data := extract(message, FlightNumberExpression)
				Expect(data).NotTo(BeNil())
				Expect(data[FlightNumber]).To(Equal("CK261"))
			})
		})

		Context("parse : 9C8812", func() {
			It("should return a valid flight number", func() {
				message := "9C8812"
				data := extract(message, FlightNumberExpression)
				Expect(data).NotTo(BeNil())
				Expect(data[FlightNumber]).To(Equal("9C8812"))
			})
		})
	})

	Describe("Schedule Date Parser", func() {
		Context("parse : 29OCT", func() {
			It("should return a valid date", func() {
				message := "29OCT"
				data := extract(message, DateExpression)
				Expect(data).NotTo(BeNil())
				Expect(data[Date]).To(Equal("29OCT"))
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

})

var _ = Describe("FindDef", func() {

	Context("find def for MF", func() {
		It("should return a valid def", func() {
			def := FindDef("MF")
			Expect(def).NotTo(BeNil())
			Expect(def.Airlines).To(ContainElement("MF"))
		})
	})
	Context("find def for FM", func() {
		It("should return a valid def", func() {
			def := FindDef("FM")
			Expect(def).NotTo(BeNil())
			Expect(def.Airlines).To(ContainElement("FM"))
		})
	})

	Context("find def for CK", func() {
		It("should return a valid def", func() {
			def := FindDef("CK")
			Expect(def).To(BeNil())
		})
	})
})

var _ = Describe("Parse Line with PreDef", func() {
	Context("FM", func() {
		It("W/Z FM9134 B2688 1/1ILS (00) TSN0100 SHA", func() {
			lineText := "W/Z FM9134 B2688 1/1ILS (00) TSN0100 SHA"
			def := FindDef("FM")
			schedule := ParseWithDef(lineText, def)
			Expect(schedule).NotTo(BeNil())
			Expect(schedule.Task).To(Equal("W/Z"))
			Expect(len(schedule.FlightNumber)).To(Equal(1))
			Expect(schedule.FlightNumber[0]).To(Equal("FM9134"))
			Expect(schedule.AircraftReg).To(Equal("B2688"))
			Expect(len(schedule.Waypoints)).To(Equal(2))
			Expect(schedule.Waypoints[0].Airport).To(Equal("TSN"))
			Expect(schedule.Waypoints[0].DepartureTime).To(Equal("0100"))
			Expect(schedule.Waypoints[1].Airport).To(Equal("SHA"))
		})

	})

	Context("MF", func() {
		It("01) MF8193 B5595 ILS(8) HGH1100 1305TSN", func() {
			lineText := "01) MF8193 B5595 ILS(8) HGH1100 1305TSN"
			def := FindDef("MF")
			Expect(def).NotTo(BeNil())
			schedule := ParseWithDef(lineText, def)
			Expect(schedule).NotTo(BeNil())
			Expect(schedule.Index).To(Equal("01)"))
			Expect(len(schedule.FlightNumber)).To(Equal(1))
			Expect(schedule.FlightNumber[0]).To(Equal("MF8193"))
			Expect(schedule.AircraftReg).To(Equal("B5595"))
			Expect(len(schedule.Waypoints)).To(Equal(2))
			Expect(schedule.Waypoints[0].Airport).To(Equal("HGH"))
			Expect(schedule.Waypoints[0].DepartureTime).To(Equal("1100"))
			Expect(schedule.Waypoints[1].Airport).To(Equal("TSN"))
			Expect(schedule.Waypoints[1].ArrivalTime).To(Equal("1305"))
		})
	})

	Context("8X", func() {
		It("L1:  29OCT  BK2735 B2863  ILS  IS (3/6)  TSN2350(28OCT)   HAK", func() {
			lineText := "L1:  29OCT  BK2735 B2863  ILS  IS (3/6)  TSN2350(28OCT)   HAK"
			def := FindDef("8X")
			Expect(def).NotTo(BeNil())
			schedule := ParseWithDef(lineText, def)
			Expect(schedule).NotTo(BeNil())
			Expect(schedule.Index).To(Equal("L1:"))
			Expect(schedule.Date).To(Equal("29OCT"))
			Expect(len(schedule.FlightNumber)).To(Equal(1))
			Expect(schedule.FlightNumber[0]).To(Equal("BK2735"))
			Expect(schedule.AircraftReg).To(Equal("B2863"))
			Expect(len(schedule.Waypoints)).To(Equal(2))
			Expect(schedule.Waypoints[0].Airport).To(Equal("TSN"))
			Expect(schedule.Waypoints[0].DepartureTime).To(Equal("2350(28OCT)"))
			Expect(schedule.Waypoints[1].Airport).To(Equal("HAK"))
		})
	})

	Context("HU", func() {
		It("L05 W/Z HU7205 B5406 (9) TSN/2355(30OCT) PVG", func() {
			lineText := "L05 W/Z HU7205 B5406 (9) TSN/2355(30OCT) PVG"
			def := FindDef("HU")
			Expect(def).NotTo(BeNil())
			schedule := ParseWithDef(lineText, def)
			Expect(schedule).NotTo(BeNil())
			Expect(schedule.Index).To(Equal("L05"))
			Expect(schedule.Task).To(Equal("W/Z"))
			Expect(len(schedule.FlightNumber)).To(Equal(1))
			Expect(schedule.FlightNumber[0]).To(Equal("HU7205"))
			Expect(schedule.AircraftReg).To(Equal("B5406"))
			Expect(len(schedule.Waypoints)).To(Equal(2))
			Expect(schedule.Waypoints[0].Airport).To(Equal("TSN"))
			Expect(schedule.Waypoints[0].DepartureTime).To(Equal("2355(30OCT)"))
			Expect(schedule.Waypoints[1].Airport).To(Equal("PVG"))
		})
	})

})
