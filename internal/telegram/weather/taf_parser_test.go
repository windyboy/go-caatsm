package weather

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TAF Parser", func() {
	It("parses simple TAF and FM period boundaries", func() {
		raw := "TAF KJFK 251200Z 2512/2612 35012KT 10SM FEW020 FM251800 36015KT 10SM SCT030="
		taf, err := parseTaf(raw)
		Expect(err).ToNot(HaveOccurred())
		Expect(taf.StationID).To(Equal("KJFK"))
		Expect(taf.Periods).To(HaveLen(2))
		Expect(taf.Periods[0].Type).To(Equal("MAIN"))
		Expect(taf.Periods[0].ValidTo).To(BeTemporally("==", taf.Periods[1].ValidFrom))
		Expect(taf.Periods[1].Type).To(Equal("FM"))
		Expect(taf.Periods[1].ValidTo).To(BeTemporally("==", taf.ValidTo))
	})

	It("rolls FM, TEMPO, and BECMG into next month when day precedes validity start", func() {
		taf1, err := parseTaf("TAF KJFK 301200Z 3012/0112 35012KT 10SM FEW020 FM010600 36015KT 10SM SCT030=")
		Expect(err).ToNot(HaveOccurred())
		Expect(taf1.Periods[1].ValidFrom).To(BeTemporally(">", taf1.ValidFrom))

		taf2, err := parseTaf("TAF KJFK 301200Z 3012/0112 35012KT 10SM FEW020 TEMPO0102/0106 5SM -RA=")
		Expect(err).ToNot(HaveOccurred())
		Expect(taf2.Periods[1].ValidFrom).To(BeTemporally(">", taf2.ValidFrom))

		taf3, err := parseTaf("TAF KJFK 301200Z 3012/0112 35012KT 10SM FEW020 BECMG0102/0106 36015KT=")
		Expect(err).ToNot(HaveOccurred())
		Expect(taf3.Periods[1].ValidFrom).To(BeTemporally(">", taf3.ValidFrom))
	})

	It("supports both compact and space-separated TEMPO/BECMG and PROB boundaries inside FM", func() {
		raw := "TAF KJFK 251200Z 2512/2612 35012KT 10SM FEW020 FM251800 36015KT 10SM SCT030 PROB30 TEMPO 2520/2602 27015G25KT 5SM -RA BECMG 2604/2606 18010KT RMK TEST="
		taf, err := parseTaf(raw)
		Expect(err).ToNot(HaveOccurred())
		Expect(taf.Warnings).To(BeEmpty())
		Expect(taf.Periods).To(HaveLen(4))
		Expect(taf.Periods[2].Type).To(Equal("TEMPO"))
		Expect(taf.Periods[2].Probability).To(Equal(30))
		Expect(taf.Periods[3].Type).To(Equal("BECMG"))
		Expect(taf.Remarks).To(ContainSubstring("RMK"))
	})

	It("supports TAF AMD/COR and standalone PROB30 periods", func() {
		raw := "TAF AMD KJFK 251200Z 2512/2612 35012KT 10SM FEW020 PROB30 2518/2522 3SM -RA BKN015="
		taf, err := parseTaf(raw)
		Expect(err).ToNot(HaveOccurred())
		Expect(taf.StationID).To(Equal("KJFK"))
		Expect(taf.Periods).To(HaveLen(2))
		Expect(taf.Periods[1].Type).To(Equal("PROB"))
		Expect(taf.Periods[1].Probability).To(Equal(30))
		Expect(taf.Periods[1].Phenomena).To(HaveLen(1))
		Expect(taf.Periods[1].Phenomena[0].Weather).To(Equal("RA"))
	})

	It("validates TAF validity calendar days while preserving ICAO hour 24 rollover", func() {
		febIssue := time.Date(2025, 2, 20, 5, 0, 0, 0, time.UTC)
		_, _, err := ParseTAFValidity("3106/3124", febIssue)
		Expect(err).To(HaveOccurred())

		febEndIssue := time.Date(2025, 2, 28, 5, 0, 0, 0, time.UTC)
		_, _, err = ParseTAFValidity("2806/2912", febEndIssue)
		Expect(err).To(HaveOccurred())

		validFrom, validTo, err := ParseTAFValidity("2806/2824", febEndIssue)
		Expect(err).ToNot(HaveOccurred())
		Expect(validFrom).To(Equal(time.Date(2025, 2, 28, 6, 0, 0, 0, time.UTC)))
		Expect(validTo).To(Equal(time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)))
	})
})
