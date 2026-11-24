package pipeline

import (
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type externalPermanentErr struct{}

func (externalPermanentErr) Error() string     { return "external permanent" }
func (externalPermanentErr) IsPermanent() bool { return true }

var _ = Describe("Permanent errors", func() {
	It("wraps errors and reports permanence", func() {
		base := errors.New("boom")
		perr := Permanent(base)

		Expect(perr).NotTo(BeNil())
		Expect(IsPermanent(perr)).To(BeTrue())
		Expect(errors.Is(perr, base)).To(BeTrue())
		Expect(errors.Is(base, perr)).To(BeFalse())
	})

	It("recognizes external errors implementing IsPermanent() bool", func() {
		wrapped := fmt.Errorf("publish failed: %w", externalPermanentErr{})
		Expect(IsPermanent(wrapped)).To(BeTrue())
	})

	It("treats nil as non-permanent", func() {
		Expect(Permanent(nil)).To(BeNil())
		Expect(IsPermanent(nil)).To(BeFalse())
	})
})
