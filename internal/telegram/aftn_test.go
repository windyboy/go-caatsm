package telegram

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AFTN Validator", func() {
	Describe("ValidatePriorityIndicator", func() {
		It("accepts valid priorities and rejects invalid ones", func() {
			for _, p := range []string{"FF", "GG", "QU", "DD", "SS", "KK", "  ff  ", ""} {
				Expect(ValidatePriorityIndicator(p)).To(Succeed())
			}
			for _, bad := range []string{"XX", "F", "FFF"} {
				err := ValidatePriorityIndicator(bad)
				Expect(err).To(HaveOccurred())
				Expect(IsAFTNError(err)).To(BeTrue())
				Expect(GetAFTNErrorType(err)).To(Equal("priority_indicator"))
			}
		})
	})

	Describe("ValidateAddress", func() {
		It("accepts 4-char ICAO codes and 8-char AFTN routing addresses", func() {
			for _, addr := range []string{"ZBTJ", "KLAX", "ZBBBZPZX", "ZBTJYOYX", "1ABC", "  zbtj  ", ""} {
				Expect(ValidateAddress(addr)).To(Succeed())
			}
			for _, bad := range []string{"ZBT", "ZBTJX", "ZBTJYOY", "ZBBBZPZXA", "ZB-J", "ZB J"} {
				err := ValidateAddress(bad)
				Expect(err).To(HaveOccurred())
				Expect(IsAFTNError(err)).To(BeTrue())
				Expect(GetAFTNErrorType(err)).To(Equal("icao_address"))
			}
		})
	})

	Describe("ValidateDateTime", func() {
		It("accepts valid DDHHMM and rejects out-of-range values", func() {
			for _, dt := range []string{"151430", "010000", "312359", ""} {
				Expect(ValidateDateTime(dt)).To(Succeed())
			}
			for _, bad := range []string{"15A430", "15143", "1514301", "001430", "321430", "152430", "151460"} {
				err := ValidateDateTime(bad)
				Expect(err).To(HaveOccurred())
				Expect(GetAFTNErrorType(err)).To(Equal("datetime"))
			}
		})
	})

	Describe("ValidateAFTN", func() {
		It("accepts 8-char AFTN routing addresses and skips weather reports", func() {
			aftnMsg := &Message{
				Protocol:           ProtocolAFTN,
				PriorityIndicator:  "FF",
				PrimaryAddress:     "ZBBBZPZX",
				Originator:         "ZBTJYOYX",
				DateTime:           "151430",
				OriginatorDateTime: "151425",
			}
			Expect(ValidateAFTN(aftnMsg)).To(Succeed())

			weatherMsg := &Message{
				Protocol: ProtocolWeather,
				Category: "METAR",
				DateTime: "261125120000",
			}
			Expect(ValidateAFTN(weatherMsg)).To(Succeed())
			Expect(ValidateAFTN(nil)).To(Succeed())
		})

		It("preserves field granularity for single error and returns multiple_errors for multiple", func() {
			singleErrMsg := &Message{
				Protocol:          ProtocolAFTN,
				PriorityIndicator: "XX",
				PrimaryAddress:    "ZBBBZPZX",
				DateTime:          "151430",
			}
			err := ValidateAFTN(singleErrMsg)
			Expect(err).To(HaveOccurred())
			Expect(IsAFTNError(err)).To(BeTrue())
			Expect(GetAFTNErrorType(err)).To(Equal("priority_indicator"))

			wrapped := fmt.Errorf("outer: %w", err)
			Expect(IsAFTNError(wrapped)).To(BeTrue())
			Expect(GetAFTNErrorType(wrapped)).To(Equal("priority_indicator"))

			multiErrMsg := &Message{
				Protocol:          ProtocolAFTN,
				PriorityIndicator: "XX",
				PrimaryAddress:    "TOOLONG",
			}
			merr := ValidateAFTN(multiErrMsg)
			Expect(merr).To(HaveOccurred())
			Expect(GetAFTNErrorType(merr)).To(Equal("multiple_errors"))
		})
	})
})
