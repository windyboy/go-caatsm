package aviation

import (
	"regexp"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Regex Timeout", func() {

	Describe("MatchWithTimeout", func() {
		Context("with simple pattern and normal input", func() {
			It("should match successfully within timeout", func() {
				re := regexp.MustCompile(`^(\w+)-(\w+)$`)
				input := "ARR-CES5470"

				match, err := MatchWithTimeout(re, input, DefaultRegexTimeout)
				Expect(err).ToNot(HaveOccurred())
				Expect(match).To(HaveLen(3))
				Expect(match[0]).To(Equal("ARR-CES5470"))
				Expect(match[1]).To(Equal("ARR"))
				Expect(match[2]).To(Equal("CES5470"))
			})
		})

		Context("with pattern that doesn't match", func() {
			It("should return nil match without error", func() {
				re := regexp.MustCompile(`^(\w+)-(\w+)$`)
				input := "INVALID FORMAT"

				match, err := MatchWithTimeout(re, input, DefaultRegexTimeout)
				Expect(err).ToNot(HaveOccurred())
				Expect(match).To(BeNil())
			})
		})

		Context("with complex ARR pattern", func() {
			It("should match ARR message within timeout", func() {
				input := "(ARR-CES5470/A1234-ZBTJ-ZSHC1614)"

				match, err := MatchWithTimeout(ArrPatternExpression, input, DefaultRegexTimeout)
				Expect(err).ToNot(HaveOccurred())
				Expect(match).ToNot(BeNil())
			})
		})

		Context("with complex DEP pattern", func() {
			It("should match DEP message within timeout", func() {
				input := "(DEP-CYZ9017/A5633-ZBTJ1638-ZSPD)"

				match, err := MatchWithTimeout(DepPatternExpression, input, DefaultRegexTimeout)
				Expect(err).ToNot(HaveOccurred())
				Expect(match).ToNot(BeNil())
			})
		})

		Context("with very short timeout", func() {
			It("should timeout on complex pattern", func() {
				// Use a very short timeout to force timeout
				veryShortTimeout := 1 * time.Nanosecond
				re := regexp.MustCompile(`^(.+)+$`)
				input := "aaaaaaaaaaaaaaaaaaaaaaaaaaaa!"

				match, err := MatchWithTimeout(re, input, veryShortTimeout)
				Expect(err).To(HaveOccurred())
				Expect(match).To(BeNil())

				valErr, ok := err.(*ValidationError)
				Expect(ok).To(BeTrue())
				Expect(valErr.Field).To(Equal("regex_timeout"))
				Expect(valErr.Message).To(ContainSubstring("exceeded timeout"))
			})
		})

		Context("with empty input", func() {
			It("should handle empty input gracefully", func() {
				re := regexp.MustCompile(`^(\w+)$`)
				input := ""

				match, err := MatchWithTimeout(re, input, DefaultRegexTimeout)
				Expect(err).ToNot(HaveOccurred())
				Expect(match).To(BeNil())
			})
		})

		Context("with named capture groups", func() {
			It("should preserve named groups in match result", func() {
				re := regexp.MustCompile(`^(?P<category>\w+)-(?P<number>\w+)$`)
				input := "ARR-CES5470"

				match, err := MatchWithTimeout(re, input, DefaultRegexTimeout)
				Expect(err).ToNot(HaveOccurred())
				Expect(match).To(HaveLen(3))
				Expect(match[0]).To(Equal("ARR-CES5470"))
			})
		})
	})
})
