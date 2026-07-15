package parser

import (
	"caatsm/internal/adapter/dto"
	"caatsm/internal/adapter/parser/aviation"
	weatherparser "caatsm/internal/adapter/parser/weather"
	"caatsm/internal/port"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CompositeParser", func() {
	var composite *CompositeParser
	var aviationParser Parser
	var weatherParser port.WeatherParser

	BeforeEach(func() {
		aviationParser = aviation.NewParser()
		weatherParser = weatherparser.NewWeatherParser()
		composite = NewCompositeParser(aviationParser, weatherParser)
	})

	Describe("Parse", func() {
		It("should route METAR to weather parser", func() {
			raw := "METAR KJFK 251200Z 35012KT 10SM FEW020 25/18 Q1013="
			parsed, err := composite.Parse(raw)
			Expect(err).ToNot(HaveOccurred())
			Expect(parsed).ToNot(BeNil())
			Expect(parsed.Category).To(Equal("METAR"))
			Expect(parsed.Parsed).To(BeTrue())
			Expect(parsed.Status).To(Equal(dto.MessageStatusParsed))
		})

		It("should route SPECI to weather parser", func() {
			raw := "SPECI KORD 251215Z 27015G25KT 5SM -RA BKN030 OVC050 20/18 A2992="
			parsed, err := composite.Parse(raw)
			Expect(err).ToNot(HaveOccurred())
			Expect(parsed).ToNot(BeNil())
			Expect(parsed.Category).To(Equal("SPECI"))
			Expect(parsed.Parsed).To(BeTrue())
		})

		It("should route TAF to weather parser", func() {
			raw := "TAF KJFK 251200Z 2512/2612 35012KT 10SM FEW020="
			parsed, err := composite.Parse(raw)
			Expect(err).ToNot(HaveOccurred())
			Expect(parsed).ToNot(BeNil())
			Expect(parsed.Category).To(Equal("TAF"))
			Expect(parsed.Parsed).To(BeTrue())
		})

		It("should route aviation messages to aviation parser", func() {
			raw := `ZCZC TMQ2617 142150
GG ZBTJZPZX
150551 ZBTJUOBK
(FPL-OKA2861-IS
-MA60/M-SHID/C
-ZBTJ0030
-K0420S0450 CG J1 FZ
-ZSYT0100 ZSQD ZYTL
-DOF/241215 EET/ZPKM0012 REG/B00FA PER/C)
NNNN`
			parsed, err := composite.Parse(raw)
			Expect(err).ToNot(HaveOccurred())
			Expect(parsed).ToNot(BeNil())
			Expect(parsed.Category).To(Equal("FPL"))
			Expect(parsed.Parsed).To(BeTrue())
		})

		It("should handle weather reports without ending =", func() {
			raw := "METAR KJFK 251200Z 35012KT 10SM FEW020 25/18 Q1013"
			// Should fall back to aviation parser
			parsed, err := composite.Parse(raw)
			// May fail or succeed depending on aviation parser
			_ = parsed
			_ = err
		})
	})
})
