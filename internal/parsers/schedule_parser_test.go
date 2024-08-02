package parsers

import (
	"strings"

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

	Describe("Date Parser", func() {
		Context("parse : 31OCT", func() {
			message := "31OCT"
			data := extract(message, DateExpression)
			It("should return a valid date", func() {
				Expect(data).NotTo(BeNil())
				Expect(data[Date]).To(Equal("31OCT"))
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

		Context("CA1371/1372/1527", func() {
			It("3 number : CA1371 CA1372 CA1527", func() {
				message := "CA1371/1372/1527"
				data := getFlightNumbers(message)
				Expect(data).NotTo(BeNil())
				Expect(data).To(HaveLen(3))
				Expect(data).To(ContainElement("CA1371"))
				Expect(data).To(ContainElement("CA1372"))
				Expect(data).To(ContainElement("CA1527"))
			})
		})

		Context("CZ3301/2", func() {
			It("2 number : CZ3301 CZ3302", func() {
				message := "CZ3301/2"
				data := getFlightNumbers(message)
				Expect(data).NotTo(BeNil())
				Expect(data).To(HaveLen(2))
				Expect(data).To(ContainElement("CZ3301"))
				Expect(data).To(ContainElement("CZ3302"))
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

	Describe("FindWaypoint", func() {
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

		It("TSN/0645", func() {
			message := "TSN/0645"
			waypoint := ExtractWaypoint(message)
			Expect(waypoint).NotTo(BeNil())
			Expect(waypoint.Airport).To(Equal("TSN"))
			Expect(waypoint.DepartureTime).To(Equal("0645"))
		})
	})

	Describe("Waypoints", func() {
		Context("XIY/0415 TSN/0645 CGQ", func() {
			It("should return 3 waypoints", func() {
				points := strings.Split("XIY/0415 TSN/0645 CGQ", " ")
				waypoints := parseWaypoints(points)
				Expect(waypoints).NotTo(BeNil())
				Expect(waypoints).To(HaveLen(3))
				Expect(waypoints[0].Airport).To(Equal("XIY"))
				Expect(waypoints[0].DepartureTime).To(Equal("0415"))
				Expect(waypoints[1].Airport).To(Equal("TSN"))
				Expect(waypoints[1].DepartureTime).To(Equal("0645"))
				Expect(waypoints[2].Airport).To(Equal("CGQ"))

			})
		})
		Context("ICN 0235 TSN", func() {
			It("should return 2 waypoints", func() {
				points := strings.Split("ICN 0235 TSN", " ")
				waypoints := parseWaypoints(points)
				Expect(waypoints).NotTo(BeNil())
				Expect(waypoints).To(HaveLen(2))
				Expect(waypoints[0].Airport).To(Equal("ICN"))
				Expect(waypoints[0].DepartureTime).To(Equal("0235"))
				Expect(waypoints[1].Airport).To(Equal("TSN"))
			})
		})
	})

})

var _ = Describe("Parser Definition", func() {

	Context("MF", func() {
		It("valid def", func() {
			def := FindDef("MF")
			Expect(def).NotTo(BeNil())
			Expect(def.Airlines).To(ContainElement("MF"))
		})
	})
	Context("FM", func() {
		It("valid def", func() {
			def := FindDef("FM")
			Expect(def).NotTo(BeNil())
			Expect(def.Airlines).To(ContainElement("FM"))
		})
	})

	Context("8X", func() {
		It("valid def", func() {
			def := FindDef("8X")
			Expect(def).NotTo(BeNil())
			Expect(def.Airlines).To(ContainElement("8X"))
		})
	})

	Context("XX", func() {
		It("nil", func() {
			def := FindDef("XX")
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

	Context("JD", func() {
		It("1)  JD5195 B6727 ILS I(9) SYX/0800 1135/TSN", func() {
			lineText := "1)  JD5195 B6727 ILS I(9) SYX/0800 1135/TSN"
			def := FindDef("JD")
			Expect(def).NotTo(BeNil())
			schedule := ParseWithDef(lineText, def)
			Expect(schedule).NotTo(BeNil())
			Expect(schedule.Index).To(Equal("1)"))
			Expect(len(schedule.FlightNumber)).To(Equal(1))
			Expect(schedule.FlightNumber[0]).To(Equal("JD5195"))
			Expect(schedule.AircraftReg).To(Equal("B6727"))
			Expect(len(schedule.Waypoints)).To(Equal(2))
			Expect(schedule.Waypoints[0].Airport).To(Equal("SYX"))
			Expect(schedule.Waypoints[0].DepartureTime).To(Equal("0800"))
			Expect(schedule.Waypoints[1].Airport).To(Equal("TSN"))
			Expect(schedule.Waypoints[1].ArrivalTime).To(Equal("1135"))
		})
	})

	Context("GS", func() {
		It("01 GS7635 B3193 XIY0020(16APR) CGD", func() {
			lineText := "01 GS7635 B3193 XIY0020(16APR) CGD"
			def := FindDef("GS")
			Expect(def).NotTo(BeNil())
			schedule := ParseWithDef(lineText, def)
			Expect(schedule).NotTo(BeNil())
			Expect(schedule.Index).To(Equal("01"))
			Expect(len(schedule.FlightNumber)).To(Equal(1))
			Expect(schedule.FlightNumber[0]).To(Equal("GS7635"))
			Expect(schedule.AircraftReg).To(Equal("B3193"))
			Expect(len(schedule.Waypoints)).To(Equal(2))
			Expect(schedule.Waypoints[0].Airport).To(Equal("XIY"))
			Expect(schedule.Waypoints[0].DepartureTime).To(Equal("0020(16APR)"))
			Expect(schedule.Waypoints[1].Airport).To(Equal("CGD"))
		})
	})

	Context("Y8", func() {
		It("13 Y87444 B2578 ICN 0235 TSN", func() {
			lineText := "13 Y87444 B2578 ICN 0235 TSN"
			def := FindDef("Y8")
			Expect(def).NotTo(BeNil())
			schedule := ParseWithDef(lineText, def)
			Expect(schedule).NotTo(BeNil())
			Expect(schedule.Index).To(Equal("13"))
			Expect(len(schedule.FlightNumber)).To(Equal(1))
			Expect(schedule.FlightNumber[0]).To(Equal("Y87444"))
			Expect(schedule.AircraftReg).To(Equal("B2578"))
			Expect(len(schedule.Waypoints)).To(Equal(2))
			Expect(schedule.Waypoints[0].Airport).To(Equal("ICN"))
			Expect(schedule.Waypoints[0].DepartureTime).To(Equal("0235"))
			Expect(schedule.Waypoints[1].Airport).To(Equal("TSN"))
		})
	})

	Context("3U", func() {
		It("01)  31OCT 3U8863 B6598 CAT1 (10) CKG0010 0235TSN", func() {
			lineText := "01)  31OCT 3U8863 B6598 CAT1 (10) CKG0010 0235TSN"
			def := FindDef("3U")
			Expect(def).NotTo(BeNil())
			schedule := ParseWithDef(lineText, def)
			Expect(schedule).NotTo(BeNil())
			Expect(schedule.Index).To(Equal("01)"))
			Expect(schedule.Date).To(Equal("31OCT"))
			Expect(len(schedule.FlightNumber)).To(Equal(1))
			Expect(schedule.FlightNumber[0]).To(Equal("3U8863"))
			Expect(schedule.AircraftReg).To(Equal("B6598"))
			Expect(len(schedule.Waypoints)).To(Equal(2))
			Expect(schedule.Waypoints[0].Airport).To(Equal("CKG"))
			Expect(schedule.Waypoints[0].DepartureTime).To(Equal("0010"))
			Expect(schedule.Waypoints[1].Airport).To(Equal("TSN"))
			Expect(schedule.Waypoints[1].ArrivalTime).To(Equal("0235"))
		})
	})

	Context("CK", func() {
		It("01)H/Z CK261 B2076 PVG1535(30OCT) 1705TPE", func() {
			lineText := "01)H/Z CK261 B2076 PVG1535(30OCT) 1705TPE"
			def := FindDef("CK")
			Expect(def).NotTo(BeNil())
			schedule := ParseWithDef(lineText, def)
			Expect(schedule).NotTo(BeNil())
			Expect(schedule.Task).To(Equal("H/Z"))
			Expect(len(schedule.FlightNumber)).To(Equal(1))
			Expect(schedule.FlightNumber[0]).To(Equal("CK261"))
			Expect(schedule.AircraftReg).To(Equal("B2076"))
			Expect(len(schedule.Waypoints)).To(Equal(2))
			Expect(schedule.Waypoints[0].Airport).To(Equal("PVG"))
			Expect(schedule.Waypoints[0].DepartureTime).To(Equal("1535(30OCT)"))
			Expect(schedule.Waypoints[1].Airport).To(Equal("TPE"))
			Expect(schedule.Waypoints[1].ArrivalTime).To(Equal("1705"))
		})
	})

	Context("G5", func() {
		It("L01 W/Z G52665 B7762 (6) CKG/0725 CIH/0940 TSN", func() {
			lineText := "L01 W/Z G52665 B7762 (6) CKG/0725 CIH/0940 TSN"
			def := FindDef("G5")
			Expect(def).NotTo(BeNil())
			schedule := ParseWithDef(lineText, def)
			Expect(schedule).NotTo(BeNil())
			Expect(schedule.Index).To(Equal("L01"))
			Expect(schedule.Task).To(Equal("W/Z"))
			Expect(len(schedule.FlightNumber)).To(Equal(1))
			Expect(schedule.FlightNumber[0]).To(Equal("G52665"))
			Expect(schedule.AircraftReg).To(Equal("B7762"))
			Expect(len(schedule.Waypoints)).To(Equal(3))
			Expect(schedule.Waypoints[0].Airport).To(Equal("CKG"))
			Expect(schedule.Waypoints[0].DepartureTime).To(Equal("0725"))
			Expect(schedule.Waypoints[1].Airport).To(Equal("CIH"))
			Expect(schedule.Waypoints[1].DepartureTime).To(Equal("0940"))
			Expect(schedule.Waypoints[2].Airport).To(Equal("TSN"))
		})
	})

	Context("9C", func() {
		It("31OCT W/Z 9C8884 B6573 ILS1/1 (06) TSN0650 SYX", func() {
			lineText := "31OCT W/Z 9C8884 B6573 ILS1/1 (06) TSN0650 SYX"
			def := FindDef("9C")
			Expect(def).NotTo(BeNil())
			schedule := ParseWithDef(lineText, def)
			Expect(schedule).NotTo(BeNil())
			Expect(schedule.Date).To(Equal("31OCT"))
			Expect(schedule.Task).To(Equal("W/Z"))
			Expect(len(schedule.FlightNumber)).To(Equal(1))
			Expect(schedule.FlightNumber[0]).To(Equal("9C8884"))
			Expect(schedule.AircraftReg).To(Equal("B6573"))
			Expect(len(schedule.Waypoints)).To(Equal(2))
			Expect(schedule.Waypoints[0].Airport).To(Equal("TSN"))
			Expect(schedule.Waypoints[0].DepartureTime).To(Equal("0650"))
			Expect(schedule.Waypoints[1].Airport).To(Equal("SYX"))
		})
	})

	Context("ZH", func() {
		It("204) W/Z 31OCT ZH9783 B5670 CAT1 (10) SZX0045 0355TSN", func() {
			lineText := "204) W/Z 31OCT ZH9783 B5670 CAT1 (10) SZX0045 0355TSN"
			def := FindDef("ZH")
			Expect(def).NotTo(BeNil())
			schedule := ParseWithDef(lineText, def)
			Expect(schedule).NotTo(BeNil())
			Expect(schedule.Index).To(Equal("204)"))
			Expect(schedule.Task).To(Equal("W/Z"))
			Expect(len(schedule.FlightNumber)).To(Equal(1))
			Expect(schedule.FlightNumber[0]).To(Equal("ZH9783"))
			Expect(schedule.AircraftReg).To(Equal("B5670"))
			Expect(len(schedule.Waypoints)).To(Equal(2))
			Expect(schedule.Waypoints[0].Airport).To(Equal("SZX"))
			Expect(schedule.Waypoints[0].DepartureTime).To(Equal("0045"))
			Expect(schedule.Waypoints[1].Airport).To(Equal("TSN"))
			Expect(schedule.Waypoints[1].ArrivalTime).To(Equal("0355"))
		})
	})

	Context("8L", func() {
		It("L59 W/Z 8L9976 B6959 TSN/0510 CTU/0855 KMG", func() {
			lineText := "L59 W/Z 8L9976 B6959 TSN/0510 CTU/0855 KMG"
			def := FindDef("8L")
			Expect(def).NotTo(BeNil())
			schedule := ParseWithDef(lineText, def)
			Expect(schedule).NotTo(BeNil())
			Expect(schedule.Index).To(Equal("L59"))
			Expect(schedule.Task).To(Equal("W/Z"))
			Expect(len(schedule.FlightNumber)).To(Equal(1))
			Expect(schedule.FlightNumber[0]).To(Equal("8L9976"))
			Expect(schedule.AircraftReg).To(Equal("B6959"))
			Expect(len(schedule.Waypoints)).To(Equal(3))
			Expect(schedule.Waypoints[0].Airport).To(Equal("TSN"))
			Expect(schedule.Waypoints[0].DepartureTime).To(Equal("0510"))
			Expect(schedule.Waypoints[1].Airport).To(Equal("CTU"))
			Expect(schedule.Waypoints[1].DepartureTime).To(Equal("0855"))
			Expect(schedule.Waypoints[2].Airport).To(Equal("KMG"))
		})
	})

	Context("SC", func() {
		It("(1) SC4717 B3080 CRJ7 ILS I (6) TAO/2350 TSN", func() {
			lineText := "(1) SC4717 B3080 CRJ7 ILS I (6) TAO/2350 TSN"
			def := FindDef("SC")
			Expect(def).NotTo(BeNil())
			schedule := ParseWithDef(lineText, def)
			Expect(schedule).NotTo(BeNil())
			Expect(schedule.Index).To(Equal("(1)"))
			Expect(len(schedule.FlightNumber)).To(Equal(1))
			Expect(schedule.FlightNumber[0]).To(Equal("SC4717"))
			Expect(schedule.AircraftReg).To(Equal("B3080"))
			Expect(len(schedule.Waypoints)).To(Equal(2))
			Expect(schedule.Waypoints[0].Airport).To(Equal("TAO"))
			Expect(schedule.Waypoints[0].DepartureTime).To(Equal("2350"))
			Expect(schedule.Waypoints[1].Airport).To(Equal("TSN"))
		})
	})

})
