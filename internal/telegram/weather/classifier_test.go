package weather

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Classifier", func() {
	Describe("Classify", func() {
		It("identifies METAR, SPECI, and TAF reports", func() {
			t1, ok1 := Classify("METAR KJFK 251200Z 35012KT 10SM FEW020 25/18 Q1013=")
			Expect(ok1).To(BeTrue())
			Expect(t1).To(Equal("METAR"))

			t2, ok2 := Classify("SPECI KORD 251215Z 27015G25KT 5SM -RA BKN030 OVC050 20/18 A2992=")
			Expect(ok2).To(BeTrue())
			Expect(t2).To(Equal("SPECI"))

			t3, ok3 := Classify("TAF KJFK 251200Z 2512/2612 35012KT 10SM FEW020=")
			Expect(ok3).To(BeTrue())
			Expect(t3).To(Equal("TAF"))

			t4, ok4 := Classify("taf amd kjfk 251200Z 2512/2612 35012KT CAVOK=")
			Expect(ok4).To(BeTrue())
			Expect(t4).To(Equal("TAF"))

			t5, ok5 := Classify("METAR COR KJFK 251200Z 35012KT CAVOK=")
			Expect(ok5).To(BeTrue())
			Expect(t5).To(Equal("METAR"))
		})

		It("returns false for non-weather and empty reports", func() {
			_, ok := Classify("(FPL-JAE7433-IS")
			Expect(ok).To(BeFalse())
			_, okEmpty := Classify("")
			Expect(okEmpty).To(BeFalse())
		})
	})

	Describe("HasValidEnding", func() {
		It("detects trailing '=' with optional whitespace", func() {
			Expect(HasValidEnding("METAR KJFK 251200Z 35012KT 10SM FEW020 25/18 Q1013=")).To(BeTrue())
			Expect(HasValidEnding("METAR KJFK 251200Z 35012KT 10SM FEW020 25/18 Q1013= ")).To(BeTrue())
			Expect(HasValidEnding("METAR KJFK 251200Z 35012KT 10SM FEW020 25/18 Q1013")).To(BeFalse())
		})
	})
})
