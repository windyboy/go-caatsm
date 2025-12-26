package aviation

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"strings"
)

var _ = Describe("Validation", func() {

	Describe("ValidateInputSize", func() {
		Context("with empty input", func() {
			It("should return validation error", func() {
				err := ValidateInputSize("")
				Expect(err).To(HaveOccurred())

				valErr, ok := err.(*ValidationError)
				Expect(ok).To(BeTrue())
				Expect(valErr.Field).To(Equal("input"))
				Expect(valErr.Message).To(ContainSubstring("empty input"))
			})
		})

		Context("with valid input size", func() {
			It("should accept input under limit", func() {
				input := strings.Repeat("A", 1000)
				err := ValidateInputSize(input)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should accept input at exact limit", func() {
				input := strings.Repeat("A", MaxTelegramSize)
				err := ValidateInputSize(input)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("with oversized input", func() {
			It("should reject input exceeding limit", func() {
				input := strings.Repeat("A", MaxTelegramSize+1)
				err := ValidateInputSize(input)
				Expect(err).To(HaveOccurred())

				valErr, ok := err.(*ValidationError)
				Expect(ok).To(BeTrue())
				Expect(valErr.Field).To(Equal("input"))
				Expect(valErr.Message).To(ContainSubstring("exceeds maximum size"))
				Expect(valErr.Message).To(ContainSubstring("1800"))
			})

			It("should reject very large input", func() {
				input := strings.Repeat("A", 10000)
				err := ValidateInputSize(input)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("ValidateBodySize", func() {
		Context("with valid body size", func() {
			It("should accept body under limit", func() {
				body := strings.Repeat("B", 1000)
				err := ValidateBodySize(body)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should accept body at exact limit", func() {
				body := strings.Repeat("B", MaxBodySize)
				err := ValidateBodySize(body)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should accept empty body", func() {
				err := ValidateBodySize("")
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("with oversized body", func() {
			It("should reject body exceeding limit", func() {
				body := strings.Repeat("B", MaxBodySize+1)
				err := ValidateBodySize(body)
				Expect(err).To(HaveOccurred())

				valErr, ok := err.(*ValidationError)
				Expect(ok).To(BeTrue())
				Expect(valErr.Field).To(Equal("body"))
				Expect(valErr.Message).To(ContainSubstring("exceeds maximum size"))
			})
		})
	})

	Describe("ValidateTokenCount", func() {
		Context("with valid token count", func() {
			It("should accept empty token list", func() {
				tokens := []Token{}
				err := ValidateTokenCount(tokens)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should accept token count under limit", func() {
				tokens := make([]Token, 100)
				err := ValidateTokenCount(tokens)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should accept token count at exact limit", func() {
				tokens := make([]Token, MaxTokenCount)
				err := ValidateTokenCount(tokens)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("with excessive token count", func() {
			It("should reject token count exceeding limit", func() {
				tokens := make([]Token, MaxTokenCount+1)
				err := ValidateTokenCount(tokens)
				Expect(err).To(HaveOccurred())

				valErr, ok := err.(*ValidationError)
				Expect(ok).To(BeTrue())
				Expect(valErr.Field).To(Equal("tokens"))
				Expect(valErr.Message).To(ContainSubstring("exceeds maximum"))
			})
		})
	})

	Describe("GetRequiredField", func() {
		Context("with existing non-empty field", func() {
			It("should return the field value", func() {
				data := map[string]string{
					"category": "ARR",
					"number":   "CES5470",
				}

				value, err := GetRequiredField(data, "category")
				Expect(err).ToNot(HaveOccurred())
				Expect(value).To(Equal("ARR"))
			})
		})

		Context("with missing field", func() {
			It("should return validation error", func() {
				data := map[string]string{
					"category": "ARR",
				}

				value, err := GetRequiredField(data, "number")
				Expect(err).To(HaveOccurred())
				Expect(value).To(Equal(""))

				valErr, ok := err.(*ValidationError)
				Expect(ok).To(BeTrue())
				Expect(valErr.Field).To(Equal("number"))
				Expect(valErr.Message).To(ContainSubstring("not found"))
			})
		})

		Context("with empty field value", func() {
			It("should return validation error", func() {
				data := map[string]string{
					"category": "",
				}

				value, err := GetRequiredField(data, "category")
				Expect(err).To(HaveOccurred())
				Expect(value).To(Equal(""))

				valErr, ok := err.(*ValidationError)
				Expect(ok).To(BeTrue())
				Expect(valErr.Field).To(Equal("category"))
				Expect(valErr.Message).To(ContainSubstring("is empty"))
			})
		})
	})

	Describe("GetOptionalField", func() {
		Context("with existing field", func() {
			It("should return the field value", func() {
				data := map[string]string{
					"ssr": "A1234",
				}

				value := GetOptionalField(data, "ssr")
				Expect(value).To(Equal("A1234"))
			})

			It("should return empty string for empty value", func() {
				data := map[string]string{
					"ssr": "",
				}

				value := GetOptionalField(data, "ssr")
				Expect(value).To(Equal(""))
			})
		})

		Context("with missing field", func() {
			It("should return empty string", func() {
				data := map[string]string{
					"category": "ARR",
				}

				value := GetOptionalField(data, "ssr")
				Expect(value).To(Equal(""))
			})
		})
	})

	Describe("ValidationError", func() {
		It("should format error message correctly", func() {
			err := &ValidationError{
				Field:   "test_field",
				Message: "test message",
			}

			Expect(err.Error()).To(Equal("validation error [test_field]: test message"))
		})
	})

	Describe("SanitizeErrorForClient", func() {
		Context("with nil error", func() {
			It("should return empty string", func() {
				result := SanitizeErrorForClient(nil)
				Expect(result).To(Equal(""))
			})
		})

		Context("with ValidationError", func() {
			It("should return the validation error message", func() {
				err := &ValidationError{
					Field:   "input",
					Message: "empty input",
				}

				result := SanitizeErrorForClient(err)
				Expect(result).To(Equal("validation error [input]: empty input"))
			})
		})

		Context("with short error message", func() {
			It("should return the error message as-is", func() {
				err := &ValidationError{
					Field:   "category",
					Message: "invalid format",
				}

				result := SanitizeErrorForClient(err)
				Expect(result).To(ContainSubstring("invalid format"))
			})
		})

		Context("with long error message containing sensitive data", func() {
			It("should truncate the message to prevent data leakage", func() {
				// Create a long error message that might contain sensitive telegram content
				sensitiveData := strings.Repeat("SENSITIVE_FLIGHT_DATA ", 20)
				err := &ValidationError{
					Field:   "body",
					Message: "invalid telegram format: " + sensitiveData,
				}

				result := SanitizeErrorForClient(err)
				// Should be truncated to 200 chars + "..."
				Expect(len(result)).To(BeNumerically("<=", 203))
			})
		})
	})
})
