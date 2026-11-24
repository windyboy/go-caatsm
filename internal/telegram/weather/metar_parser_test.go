package weather

import (
	"time"

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

	It("parses rain phenomena RA/RASN, RVR warnings, CAVOK, mixed fractions, vicinity VCSH, METAR COR, and warnings before RMK", func() {
		m, err := parseMetar("METAR COR KJFK 251200Z 24015KT R24L/1200FT +TSRA BKN020 20/18 Q1013=", "METAR")
		Expect(err).ToNot(HaveOccurred())
		Expect(m.Modifier).To(Equal("COR"))
		Expect(m.Phenomena).To(HaveLen(1))
		Expect(m.Phenomena[0].Intensity).To(Equal("+"))
		Expect(m.Phenomena[0].Descriptor).To(Equal("TS"))
		Expect(m.Phenomena[0].Weather).To(Equal("RA"))
		Expect(m.Warnings).To(ContainElement("RVR not parsed: R24L/1200FT"))

		mCavok, err := parseMetar("METAR ZBAA 251200Z 18005MPS CAVOK 18/05 Q1020=", "METAR")
		Expect(err).ToNot(HaveOccurred())
		Expect(mCavok.Visibility).NotTo(BeNil())
		Expect(mCavok.Visibility.Modifier).To(Equal("CAVOK"))
		Expect(mCavok.Visibility.Distance).To(Equal(9999.0))

		mFrac, err := parseMetar("METAR KJFK 251200Z 24015KT 1 1/2SM RA VCSH EXTRA RMK SLP012=", "METAR")
		Expect(err).ToNot(HaveOccurred())
		Expect(mFrac.Visibility.Distance).To(Equal(1.5))
		Expect(mFrac.Visibility.Unit).To(Equal("SM"))
		Expect(mFrac.Phenomena).To(HaveLen(2))
		Expect(mFrac.Phenomena[0].Weather).To(Equal("RA"))
		Expect(mFrac.Phenomena[1].Intensity).To(Equal("VC"))
		Expect(mFrac.Phenomena[1].Descriptor).To(Equal("SH"))
		Expect(mFrac.Warnings).To(ContainElement("unrecognized token: EXTRA"))
		Expect(mFrac.Remarks).To(Equal("RMK SLP012"))
	})

	It("handles UTC backward and forward month rollovers and validates timestamps", func() {
		dec1 := time.Date(2025, 12, 1, 0, 5, 0, 0, time.UTC)
		prevMonth, err := parseTimeAt("302350Z", dec1)
		Expect(err).ToNot(HaveOccurred())
		Expect(prevMonth.Year()).To(Equal(2025))
		Expect(prevMonth.Month()).To(Equal(time.November))
		Expect(prevMonth.Day()).To(Equal(30))

		dec31 := time.Date(2025, 12, 31, 23, 50, 0, 0, time.UTC)
		nextMonth, err := parseTimeAt("010010Z", dec31)
		Expect(err).ToNot(HaveOccurred())
		Expect(nextMonth.Year()).To(Equal(2026))
		Expect(nextMonth.Month()).To(Equal(time.January))

		feb := time.Date(2025, 2, 18, 12, 0, 0, 0, time.UTC)
		_, badDay := parseTimeAt("311200Z", feb)
		Expect(badDay).To(HaveOccurred())

		_, badLeap := parseTimeAt("291200Z", time.Date(2025, 2, 15, 12, 0, 0, 0, time.UTC))
		Expect(badLeap).To(HaveOccurred())

		_, badHour := parseTimeAt("152500Z", feb)
		Expect(badHour).To(HaveOccurred())
	})
})
