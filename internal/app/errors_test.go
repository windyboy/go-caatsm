package app

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Permanent errors", func() {
	It("wraps errors and reports permanence", func() {
		base := errors.New("boom")
		perr := Permanent(base)

		Expect(perr).NotTo(BeNil())
		Expect(IsPermanent(perr)).To(BeTrue())
		Expect(errors.Is(perr, base)).To(BeTrue())
		Expect(errors.Is(base, perr)).To(BeFalse())
	})

	It("treats nil as non-permanent", func() {
		Expect(Permanent(nil)).To(BeNil())
		Expect(IsPermanent(nil)).To(BeFalse())
	})
})
