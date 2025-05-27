package domain_test // Using _test package

import (
	"caatsm/internal/domain"
	"testing"
	"time" // Used in SITA struct for ReceivedTime

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSita(t *testing.T) { // Changed from TestConfig to TestSita for clarity
	RegisterFailHandler(Fail)
	RunSpecs(t, "SitaDomain Suite") // Changed suite name
}

var _ = Describe("SITAHeader Validation", func() { // Changed Describe to be more specific
	var sh domain.SITAHeader

	BeforeEach(func() {
		sh = domain.SITAHeader{
			StartSignal: "ZCZC",
			SendID:      "MSGID001",
			SendTime:    "151230", // DDHHMM
		}
	})

	Context("when header is valid", func() {
		It("should return nil", func() {
			Expect(sh.Validate()).To(BeNil())
		})
	})

	Context("when SendTime has invalid length", func() {
		It("should return an error", func() {
			sh.SendTime = "15123" // Too short
			// Corrected error message to match actual implementation
			Expect(sh.Validate()).To(MatchError("SITAHeader.SendTime: invalid send_time format, expected DDHHMM"))
		})
	})
	Context("when SendTime is empty", func() {
		It("should return an error", func() {
			sh.SendTime = "" // empty
			// Corrected error message
			Expect(sh.Validate()).To(MatchError("SITAHeader.SendTime: invalid send_time format, expected DDHHMM"))
		})
	})
})

var _ = Describe("SITA Validation", func() { // Changed Describe
	Describe("Validate", func() {
		var s domain.SITA

		BeforeEach(func() {
			s = domain.SITA{
				Header: domain.SITAHeader{
					StartSignal: "ZCZC",
					SendID:      "MSGID001",
					SendTime:    "151230",
				},
				PriorityAndSender: domain.PrioritySender{Priority: "QQ", Sender: "ADDR1"},
				TimeAndReceiver:   domain.TimeReceiver{Time: "151235", Receiver: "ADDR2"},
				Text:              "Some message text",
				ReceivedTime:      time.Now(),
				Category:          "FPL",
			}
		})

		Context("when SITA message has a valid header", func() {
			It("should return nil", func() {
				Expect(s.Validate()).To(BeNil())
			})
		})

		Context("when embedded SITAHeader is invalid", func() {
			It("should return an error from the header validation", func() {
				s.Header.SendTime = "invalid"
				// Corrected error message
				Expect(s.Validate()).To(MatchError("SITAHeader.SendTime: invalid send_time format, expected DDHHMM"))
			})
		})
	})
})
