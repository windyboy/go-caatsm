package parsers

import (
	"caatsm/internal/config"
	"regexp"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestParsers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Parsers Suite")
}

var _ = Describe("Pattern Parser", func() {
	var testConfig *config.Config

	BeforeEach(func() {
		testConfig = &config.Config{
			Body: []config.BodyConfig{
				{
					Name: "FPL",
					Patterns: []config.PatternConfig{
						{
							Pattern:    `^\((?P<type>[A-Z]{3})-(?P<number>[A-Z0-9]+)\)$`,
							Expression: regexp.MustCompile(`^\((?P<type>[A-Z]{3})-(?P<number>[A-Z0-9]+)\)$`),
						},
					},
				},
			},
		}
	})

	Describe("FindPatterns", func() {
		It("should return the correct BodyConfig based on the message body", func() {
			message := "(FPL-AB123)"
			bodyConfig := FindPatterns(message, testConfig)
			Expect(bodyConfig).NotTo(BeNil())
			Expect(bodyConfig.Name).To(Equal("FPL"))
		})

		It("should return nil if no pattern matches", func() {
			message := "(XYZ-123)"
			bodyConfig := FindPatterns(message, testConfig)
			Expect(bodyConfig).To(BeNil())
		})
	})

	Describe("ParseBody", func() {
		It("should parse the message body and extract data based on patterns", func() {
			message := "(FPL-AB123)"
			parsedData := ParseBody(message, testConfig)
			Expect(parsedData).NotTo(BeNil())
			Expect(parsedData["type"]).To(Equal("FPL"))
			Expect(parsedData["number"]).To(Equal("AB123"))
		})

		It("should return nil if no patterns match", func() {
			message := "(XYZ-123)"
			parsedData := ParseBody(message, testConfig)
			Expect(parsedData).To(BeNil())
		})
	})
})
