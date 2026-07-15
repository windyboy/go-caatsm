package weather

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("METAR Parser", func() {
	Describe("parseMetar", func() {
		It("should parse a standard METAR", func() {
			raw := "METAR KJFK 251200Z 35012KT 10SM FEW020 25/18 Q1013="
			metar, err := parseMetar(raw, "METAR")
			Expect(err).ToNot(HaveOccurred())
			Expect(metar).ToNot(BeNil())
			Expect(metar.StationID).To(Equal("KJFK"))
			Expect(metar.Wind).ToNot(BeNil())
			Expect(metar.Wind.Direction).To(Equal(350))
			Expect(metar.Wind.Speed).To(Equal(12))
			Expect(metar.Visibility).ToNot(BeNil())
			Expect(metar.Visibility.Distance).To(Equal(10.0))
			Expect(metar.Visibility.Unit).To(Equal("SM"))
			Expect(len(metar.Clouds)).To(Equal(1))
			Expect(metar.Clouds[0].Type).To(Equal("FEW"))
			Expect(metar.Temperature).ToNot(BeNil())
			Expect(metar.Temperature.Value).To(Equal(25.0))
			Expect(metar.Dewpoint).ToNot(BeNil())
			Expect(metar.Dewpoint.Value).To(Equal(18.0))
			Expect(metar.Altimeter).ToNot(BeNil())
			Expect(metar.Altimeter.Value).To(Equal(1013.0))
		})

		It("should parse METAR with variable wind", func() {
			raw := "METAR KORD 251200Z VRB05KT 10SM CLR 20/15 Q1013="
			metar, err := parseMetar(raw, "METAR")
			Expect(err).ToNot(HaveOccurred())
			Expect(metar.Wind).ToNot(BeNil())
			Expect(metar.Wind.Variable).To(BeTrue())
		})

		It("should parse METAR with gust", func() {
			raw := "METAR KJFK 251200Z 27015G25KT 10SM FEW020 25/18 Q1013="
			metar, err := parseMetar(raw, "METAR")
			Expect(err).ToNot(HaveOccurred())
			Expect(metar.Wind).ToNot(BeNil())
			Expect(metar.Wind.Gust).To(Equal(25))
		})

		It("should parse METAR with AUTO modifier", func() {
			raw := "METAR KJFK 251200Z AUTO 35012KT 10SM FEW020 25/18 Q1013="
			metar, err := parseMetar(raw, "METAR")
			Expect(err).ToNot(HaveOccurred())
			Expect(metar.Modifier).To(Equal("AUTO"))
		})

		It("should parse METAR with COR modifier", func() {
			raw := "METAR KJFK 251200Z COR 35012KT 10SM FEW020 25/18 Q1013="
			metar, err := parseMetar(raw, "METAR")
			Expect(err).ToNot(HaveOccurred())
			Expect(metar.Modifier).To(Equal("COR"))
		})

		It("should parse METAR with weather phenomena", func() {
			raw := "METAR KJFK 251200Z 35012KT 5SM -RA BKN030 OVC050 20/18 Q1013="
			metar, err := parseMetar(raw, "METAR")
			Expect(err).ToNot(HaveOccurred())
			Expect(len(metar.Phenomena)).To(BeNumerically(">", 0))
			Expect(metar.Phenomena[0].Intensity).To(Equal("-"))
			Expect(metar.Phenomena[0].Weather).To(Equal("RA"))
		})

		It("should parse METAR with multiple clouds", func() {
			raw := "METAR KJFK 251200Z 35012KT 10SM FEW020 SCT030 BKN100 25/18 Q1013="
			metar, err := parseMetar(raw, "METAR")
			Expect(err).ToNot(HaveOccurred())
			Expect(len(metar.Clouds)).To(Equal(3))
		})

		It("should parse METAR with CB clouds", func() {
			raw := "METAR KJFK 251200Z 35012KT 10SM SCT030CB 25/18 Q1013="
			metar, err := parseMetar(raw, "METAR")
			Expect(err).ToNot(HaveOccurred())
			Expect(len(metar.Clouds)).To(Equal(1))
			Expect(metar.Clouds[0].Modifier).To(Equal("CB"))
		})

		It("should parse METAR with altimeter in inHg", func() {
			raw := "METAR KJFK 251200Z 35012KT 10SM FEW020 25/18 A2992="
			metar, err := parseMetar(raw, "METAR")
			Expect(err).ToNot(HaveOccurred())
			Expect(metar.Altimeter).ToNot(BeNil())
			Expect(metar.Altimeter.Unit).To(Equal("A"))
			Expect(metar.Altimeter.Value).To(Equal(29.92))
		})

		It("should parse METAR with negative temperature", func() {
			raw := "METAR KJFK 251200Z 35012KT 10SM FEW020 M05/M10 Q1013="
			metar, err := parseMetar(raw, "METAR")
			Expect(err).ToNot(HaveOccurred())
			Expect(metar.Temperature).ToNot(BeNil())
			Expect(metar.Temperature.Value).To(Equal(-5.0))
			Expect(metar.Dewpoint).ToNot(BeNil())
			Expect(metar.Dewpoint.Value).To(Equal(-10.0))
		})

		It("should parse METAR with remarks", func() {
			raw := "METAR KJFK 251200Z 35012KT 10SM FEW020 25/18 Q1013 RMK TEST REMARKS="
			metar, err := parseMetar(raw, "METAR")
			Expect(err).ToNot(HaveOccurred())
			Expect(metar.Remarks).To(ContainSubstring("RMK"))
		})

		It("should handle unrecognized tokens as warnings", func() {
			raw := "METAR KJFK 251200Z 35012KT 10SM FEW020 25/18 Q1013 UNKNOWN TOKEN="
			metar, err := parseMetar(raw, "METAR")
			Expect(err).ToNot(HaveOccurred())
			Expect(len(metar.Warnings)).To(BeNumerically(">", 0))
		})

		It("should parse SPECI reports", func() {
			raw := "SPECI KORD 251215Z 27015G25KT 5SM -RA BKN030 OVC050 20/18 A2992="
			metar, err := parseMetar(raw, "SPECI")
			Expect(err).ToNot(HaveOccurred())
			Expect(string(metar.ReportType)).To(Equal("SPECI"))
		})
	})
})

