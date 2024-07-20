package parsers

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pattern Parser", func() {

	Describe("FindPatterns", func() {
		It("should return the correct BodyConfig based on the message body", func() {
			message := "(ARR-AB123-SSR1234-KJFK-KLAX)"
			bodyConfig := FindPatterns(message)
			Expect(bodyConfig).NotTo(BeNil())
			// Expect(bodyConfig.Name).To(Equal("ARR"))
		})

		It("should return nil if no pattern matches", func() {
			message := "(XYZ-123)"
			bodyConfig := FindPatterns(message)
			Expect(bodyConfig).To(BeNil())
		})
	})

	Describe("ParseBody", func() {
		It("should parse the message body and extract data based on patterns", func() {
			message := "(ARR-AB123-SSR1234-KJFK-KLAX)"
			parsedData := ParseBody(message)
			Expect(parsedData).NotTo(BeNil())
			Expect(parsedData["category"]).To(Equal("ARR"))
			Expect(parsedData["number"]).To(Equal("AB123"))
			Expect(parsedData["ssr"]).To(Equal("SSR1234"))
			Expect(parsedData["departure"]).To(Equal("KJFK"))
			Expect(parsedData["arrival"]).To(Equal("KLAX"))
		})

		It("should return nil if no patterns match", func() {
			message := "(XYZ-123)"
			parsedData := ParseBody(message)
			Expect(parsedData).To(BeNil())
		})
	})
})
