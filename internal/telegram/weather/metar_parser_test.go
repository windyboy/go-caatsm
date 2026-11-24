package weather

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("METAR Parser", func() {
	It("parses a standard METAR", func() {
		raw := "METAR KJFK 251200Z 35012KT 10SM FEW020 25/18 Q1013="
		metar, err := parseMetar(raw, "METAR")
		Expect(err).ToNot(HaveOccurred())
		Expect(metar.StationID).To(Equal("KJFK"))
		Expect(metar.Wind.Direction).To(Equal(350))
		Expect(metar.Wind.Speed).To(Equal(12))
		Expect(metar.Visibility.Distance).To(Equal(10.0))
		Expect(metar.Visibility.Unit).To(Equal("SM"))
		Expect(metar.Clouds).To(HaveLen(1))
		Expect(metar.Temperature.Value).To(Equal(25.0))
		Expect(metar.Dewpoint.Value).To(Equal(18.0))
		Expect(metar.Altimeter.Value).To(Equal(1013.0))
	})

	It("parses variable wind, gusts, modifiers, phenomena, multiple clouds, inHg, negative temp, remarks, and warnings", func() {
		m1, err := parseMetar("METAR KORD 251200Z VRB05KT 10SM CLR 20/15 Q1013=", "METAR")
		Expect(err).ToNot(HaveOccurred())
		Expect(m1.Wind.Variable).To(BeTrue())

		m2, err := parseMetar("METAR KJFK 251200Z AUTO 27015G25KT 5SM -RA SCT030CB 25/18 A2992 RMK TEST=", "METAR")
		Expect(err).ToNot(HaveOccurred())
		Expect(m2.Modifier).To(Equal("AUTO"))
		Expect(m2.Wind.Gust).To(Equal(25))
		Expect(m2.Phenomena[0].Weather).To(Equal("RA"))
		Expect(m2.Clouds[0].Modifier).To(Equal("CB"))
		Expect(m2.Altimeter.Unit).To(Equal("A"))
		Expect(m2.Altimeter.Value).To(Equal(29.92))
		Expect(m2.Remarks).To(ContainSubstring("RMK"))

		m3, err := parseMetar("SPECI KORD 251215Z 27015KT 10SM FEW020 M05/M10 Q1013 UNKNOWN=", "SPECI")
		Expect(err).ToNot(HaveOccurred())
		Expect(string(m3.ReportType)).To(Equal("SPECI"))
		Expect(m3.Temperature.Value).To(Equal(-5.0))
		Expect(m3.Dewpoint.Value).To(Equal(-10.0))
		Expect(m3.Warnings).ToNot(BeEmpty())
	})
})
