package weather

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TAF Parser", func() {
	Describe("parseTaf", func() {
		It("should parse a simple TAF", func() {
			raw := "TAF KJFK 251200Z 2512/2612 35012KT 10SM FEW020="
			taf, err := parseTaf(raw)
			Expect(err).ToNot(HaveOccurred())
			Expect(taf).ToNot(BeNil())
			Expect(taf.StationID).To(Equal("KJFK"))
			Expect(len(taf.Periods)).To(BeNumerically(">", 0))
		})

		It("should parse TAF with FM period", func() {
			raw := "TAF KJFK 251200Z 2512/2612 35012KT 10SM FEW020 FM251800 36015KT 10SM SCT030="
			taf, err := parseTaf(raw)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(taf.Periods)).To(BeNumerically(">=", 2))
			Expect(taf.Periods[1].Type).To(Equal("FM"))
		})

		It("should parse TAF with TEMPO period", func() {
			raw := "TAF KJFK 251200Z 2512/2612 35012KT 10SM FEW020 TEMPO2512/2515 27015G25KT 5SM -RA="
			taf, err := parseTaf(raw)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(taf.Periods)).To(BeNumerically(">=", 1))
			// Find TEMPO period
			found := false
			for _, period := range taf.Periods {
				if period.Type == "TEMPO" {
					found = true
					Expect(period.Wind).ToNot(BeNil())
					Expect(len(period.Phenomena)).To(BeNumerically(">", 0))
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("should parse TAF with BECMG period", func() {
			raw := "TAF KJFK 251200Z 2512/2612 35012KT 10SM FEW020 BECMG2512/2515 36015KT="
			taf, err := parseTaf(raw)
			Expect(err).ToNot(HaveOccurred())
			// Find BECMG period
			found := false
			for _, period := range taf.Periods {
				if period.Type == "BECMG" {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("should parse TAF with PROB", func() {
			raw := "TAF KJFK 251200Z 2512/2612 35012KT 10SM FEW020 PROB30 TEMPO2512/2515 27015G25KT 5SM -RA="
			taf, err := parseTaf(raw)
			Expect(err).ToNot(HaveOccurred())
			// Find period with probability
			found := false
			for _, period := range taf.Periods {
				if period.Probability > 0 {
					found = true
					Expect(period.Probability).To(Equal(30))
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("should parse TAF with multiple periods", func() {
			raw := "TAF KJFK 251200Z 2512/2612 35012KT 10SM FEW020 FM251800 36015KT 10SM SCT030 TEMPO2520/2602 27015G25KT 5SM -RA="
			taf, err := parseTaf(raw)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(taf.Periods)).To(BeNumerically(">=", 2))
		})

		It("should parse TAF with remarks", func() {
			raw := "TAF KJFK 251200Z 2512/2612 35012KT 10SM FEW020 RMK TEST REMARKS="
			taf, err := parseTaf(raw)
			Expect(err).ToNot(HaveOccurred())
			Expect(taf.Remarks).To(ContainSubstring("RMK"))
		})

		It("should handle unrecognized tokens as warnings", func() {
			raw := "TAF KJFK 251200Z 2512/2612 35012KT 10SM FEW020 UNKNOWN TOKEN="
			taf, err := parseTaf(raw)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(taf.Warnings)).To(BeNumerically(">", 0))
		})
	})
})

