package weather

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Classifier", func() {
	Describe("Classify", func() {
		It("should identify METAR reports", func() {
			reportType, ok := Classify("METAR KJFK 251200Z 35012KT 10SM FEW020 25/18 Q1013=")
			Expect(ok).To(BeTrue())
			Expect(reportType).To(Equal("METAR"))
		})

		It("should identify SPECI reports", func() {
			reportType, ok := Classify("SPECI KORD 251215Z 27015G25KT 5SM -RA BKN030 OVC050 20/18 A2992=")
			Expect(ok).To(BeTrue())
			Expect(reportType).To(Equal("SPECI"))
		})

		It("should identify TAF reports", func() {
			reportType, ok := Classify("TAF KJFK 251200Z 2512/2612 35012KT 10SM FEW020=")
			Expect(ok).To(BeTrue())
			Expect(reportType).To(Equal("TAF"))
		})

		It("should return false for non-weather reports", func() {
			_, ok := Classify("(FPL-JAE7433-IS")
			Expect(ok).To(BeFalse())
		})

		It("should return false for empty string", func() {
			_, ok := Classify("")
			Expect(ok).To(BeFalse())
		})
	})

	Describe("HasValidEnding", func() {
		It("should return true for reports ending with =", func() {
			Expect(HasValidEnding("METAR KJFK 251200Z 35012KT 10SM FEW020 25/18 Q1013=")).To(BeTrue())
		})

		It("should return false for reports without =", func() {
			Expect(HasValidEnding("METAR KJFK 251200Z 35012KT 10SM FEW020 25/18 Q1013")).To(BeFalse())
		})

		It("should handle whitespace", func() {
			Expect(HasValidEnding("METAR KJFK 251200Z 35012KT 10SM FEW020 25/18 Q1013= ")).To(BeTrue())
		})
	})
})

