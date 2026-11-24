package weather

import (
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
})
