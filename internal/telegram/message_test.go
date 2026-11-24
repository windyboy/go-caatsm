package telegram

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Message", func() {
	It("initializes defaults and serializes status with lowercase json tag", func() {
		msg := NewMessage()
		Expect(msg.Parsed).To(BeFalse())
		Expect(msg.Status).To(Equal(StatusUnknown))
		Expect(msg.Protocol).To(Equal(ProtocolUnknown))

		blob, err := json.Marshal(msg)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(blob)).To(ContainSubstring(`"status":"unknown"`))
	})

	It("generates valid RFC 4122 v4 UUIDs", func() {
		u := NewUUID()
		Expect(u).To(MatchRegexp(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`))
	})
})
