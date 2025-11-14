package parsers

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pattern Parser", func() {

	Describe("FindPatterns", func() {
		DescribeTable("returns the correct BodyConfig", func(message string, expectedCategory string) {
			bodyConfig := FindPatterns(message)
			if expectedCategory == "" {
				Expect(bodyConfig).To(BeNil())
				return
			}
			Expect(bodyConfig).NotTo(BeNil())
			Expect(bodyConfig.Name).To(Equal(expectedCategory))
		},
			Entry("arrival message", "(ARR-AB123-SSR1234-KJFK-KLAX)", "ARR"),
			Entry("departure message", "(DEP-CYZ9017/A5633-ZBTJ1638-ZSPD)", "DEP"),
			Entry("flight plan message", "(FPL-CCA1532-IS-ZSSS2035-ZBAA0153)", "FPL"),
			Entry("unknown message", "(XYZ-123)", ""),
		)
	})

	Describe("ParseBody", func() {
		It("should parse the message body and extract data based on patterns", func() {
			message := "(ARR-AB123/A1234-KJFK-KLAX1234)"
			parsedData := ParseBody(message)
			Expect(parsedData).NotTo(BeNil())
			Expect(parsedData["category"]).To(Equal("ARR"))
			Expect(parsedData["number"]).To(Equal("AB123"))
			Expect(parsedData["ssr"]).To(Equal("A1234"))
			Expect(parsedData[DepartureCode]).To(Equal("KJFK"))
			Expect(parsedData[ArrivalCode]).To(Equal("KLAX"))
		})

		It("should return nil if no patterns match", func() {
			message := "(XYZ-123)"
			parsedData := ParseBody(message)
			Expect(parsedData).To(BeNil())
		})

		Describe("edge cases", func() {
			It("returns nil when category is missing", func() {
				message := "(--AB123/A1234-KJFK-KLAX1234)"
				Expect(ParseBody(message)).To(BeNil())
			})

			It("returns nil when mandatory fields are empty", func() {
				message := "(ARR-AB123//-KJFK-)"
				Expect(ParseBody(message)).To(BeNil())
			})
		})
	})
})
