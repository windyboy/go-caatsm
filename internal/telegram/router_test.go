package telegram

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Router", func() {
	var router *Router

	BeforeEach(func() {
		router = NewRouter()
	})

	It("routes METAR, SPECI, and TAF reports to weather parser", func() {
		metar, err := router.Parse("METAR KJFK 251200Z 35012KT 10SM FEW020 25/18 Q1013=")
		Expect(err).NotTo(HaveOccurred())
		Expect(metar.Protocol).To(Equal(ProtocolWeather))
		Expect(metar.Category).To(Equal("METAR"))
		Expect(metar.Parsed).To(BeTrue())
		Expect(metar.Status).To(Equal(StatusParsed))

		speci, err := router.Parse("SPECI KORD 251215Z 27015G25KT 5SM -RA BKN030 OVC050 20/18 A2992=")
		Expect(err).NotTo(HaveOccurred())
		Expect(speci.Category).To(Equal("SPECI"))

		taf, err := router.Parse("TAF KJFK 251200Z 2512/2612 35012KT 10SM FEW020=")
		Expect(err).NotTo(HaveOccurred())
		Expect(taf.Category).To(Equal("TAF"))
	})

	It("surfaces weather parser errors instead of falling back to aviation parser", func() {
		msg, err := router.Parse("METAR KJFK BADTIME=")
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(ErrBodyParse))
		Expect(err.Error()).To(ContainSubstring("failed to parse issue time"))
		Expect(msg).NotTo(BeNil())
		Expect(msg.Protocol).To(Equal(ProtocolWeather))
		Expect(msg.Category).To(Equal("METAR"))
		Expect(msg.Parsed).To(BeFalse())
		Expect(msg.Status).To(Equal(StatusBodyError))
	})

	It("routes AFTN aviation messages to aviation parser", func() {
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
		msg, err := router.Parse(raw)
		Expect(err).NotTo(HaveOccurred())
		Expect(msg.Protocol).To(Equal(ProtocolAFTN))
		Expect(msg.Category).To(Equal("FPL"))
		Expect(msg.Parsed).To(BeTrue())
		Expect(msg.Status).To(Equal(StatusParsed))
	})
})
