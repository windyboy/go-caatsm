package weather

import (
	domainweather "caatsm/internal/domain/weather"
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

		It("should set final FM validTo to the TAF validity end", func() {
			raw := "TAF KJFK 251200Z 2512/2612 35012KT 10SM FEW020 FM251800 36015KT 10SM SCT030="
			taf, err := parseTaf(raw)
			Expect(err).ToNot(HaveOccurred())
			var fmPeriod *domainweather.TafPeriod
			for i := range taf.Periods {
				if taf.Periods[i].Type == "FM" {
					fmPeriod = &taf.Periods[i]
					break
				}
			}
			Expect(fmPeriod).ToNot(BeNil())
			Expect(fmPeriod.ValidTo).To(BeTemporally("==", taf.ValidTo))
		})

		It("should truncate main period at the first FM", func() {
			raw := "TAF KJFK 251200Z 2512/2612 35012KT 10SM FEW020 FM251800 36015KT 10SM SCT030="
			taf, err := parseTaf(raw)
			Expect(err).ToNot(HaveOccurred())
			var fmPeriod *domainweather.TafPeriod
			for i := range taf.Periods {
				if taf.Periods[i].Type == "FM" {
					fmPeriod = &taf.Periods[i]
					break
				}
			}
			Expect(fmPeriod).ToNot(BeNil())
			Expect(taf.Periods[0].Type).To(Equal("MAIN"))
			Expect(taf.Periods[0].ValidTo).To(BeTemporally("==", fmPeriod.ValidFrom))
		})

		It("should roll FM into next month when day precedes validity start", func() {
			raw := "TAF KJFK 301200Z 3012/0112 35012KT 10SM FEW020 FM010600 36015KT 10SM SCT030="
			taf, err := parseTaf(raw)
			Expect(err).ToNot(HaveOccurred())
			var fmPeriod *domainweather.TafPeriod
			for i := range taf.Periods {
				if taf.Periods[i].Type == "FM" {
					fmPeriod = &taf.Periods[i]
					break
				}
			}
			Expect(fmPeriod).ToNot(BeNil())
			Expect(fmPeriod.ValidFrom).To(BeTemporally(">", taf.ValidFrom))
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

		It("should roll TEMPO into next month when day precedes validity start", func() {
			raw := "TAF KJFK 301200Z 3012/0112 35012KT 10SM FEW020 TEMPO0102/0106 5SM -RA="
			taf, err := parseTaf(raw)
			Expect(err).ToNot(HaveOccurred())
			var tempoPeriod *domainweather.TafPeriod
			for i := range taf.Periods {
				if taf.Periods[i].Type == "TEMPO" {
					tempoPeriod = &taf.Periods[i]
					break
				}
			}
			Expect(tempoPeriod).ToNot(BeNil())
			Expect(tempoPeriod.ValidFrom).To(BeTemporally(">", taf.ValidFrom))
		})

		It("should roll BECMG into next month when day precedes validity start", func() {
			raw := "TAF KJFK 301200Z 3012/0112 35012KT 10SM FEW020 BECMG0102/0106 36015KT="
			taf, err := parseTaf(raw)
			Expect(err).ToNot(HaveOccurred())
			var becmgPeriod *domainweather.TafPeriod
			for i := range taf.Periods {
				if taf.Periods[i].Type == "BECMG" {
					becmgPeriod = &taf.Periods[i]
					break
				}
			}
			Expect(becmgPeriod).ToNot(BeNil())
			Expect(becmgPeriod.ValidFrom).To(BeTemporally(">", taf.ValidFrom))
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
