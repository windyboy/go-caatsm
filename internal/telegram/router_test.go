package telegram

import (
	"caatsm/internal/telegram/weather"

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
		Expect(metar.MessageID).NotTo(Equal(taf.MessageID))
		Expect(metar.MessageID).To(HavePrefix("METAR-KJFK-"))
		Expect(taf.MessageID).To(HavePrefix("TAF-KJFK-"))

		metarCor, err := router.Parse("METAR COR KJFK 251200Z 35012KT 10SM FEW020 25/18 Q1013=")
		Expect(err).NotTo(HaveOccurred())
		Expect(metarCor.MessageID).To(HavePrefix("METAR-COR-KJFK-"))
		Expect(metarCor.MessageID).NotTo(Equal(metar.MessageID))

		tafAmd, err := router.Parse("TAF AMD KJFK 251200Z 2512/2612 35012KT 10SM FEW020=")
		Expect(err).NotTo(HaveOccurred())
		Expect(tafAmd.MessageID).To(HavePrefix("TAF-AMD-KJFK-"))
		Expect(tafAmd.MessageID).NotTo(Equal(taf.MessageID))
	})

	It("parses lowercase weather reports while preserving RMK remark casing", func() {
		msg, err := router.Parse("metar kjfk 251200z 35012kt cavok 12/08 q1013 rmk Ao2=")
		Expect(err).NotTo(HaveOccurred())
		Expect(msg.Protocol).To(Equal(ProtocolWeather))
		Expect(msg.Category).To(Equal("METAR"))
		Expect(msg.Parsed).To(BeTrue())
		Expect(msg.Status).To(Equal(StatusParsed))

		metar, ok := msg.BodyData.(*weather.Metar)
		Expect(ok).To(BeTrue())
		Expect(metar.StationID).To(Equal("KJFK"))
		Expect(metar.Visibility).NotTo(BeNil())
		Expect(metar.Visibility.Modifier).To(Equal("CAVOK"))
		Expect(metar.Temperature).NotTo(BeNil())
		Expect(metar.Temperature.Value).To(Equal(12.0))
		Expect(metar.Altimeter).NotTo(BeNil())
		Expect(metar.Altimeter.Value).To(Equal(1013.0))
		Expect(metar.Remarks).To(Equal("rmk Ao2"))
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

	It("keeps weather reports missing '=' terminator on the weather error path", func() {
		msg, err := router.Parse("METAR KJFK 251200Z 35012KT 10SM FEW020")
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(ErrBodyParse))
		Expect(err.Error()).To(ContainSubstring("missing '='"))
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
