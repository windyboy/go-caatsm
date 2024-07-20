package domain

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AFTN", func() {
	var original AFTN

	BeforeEach(func() {
		original = AFTN{
			Header: Header{
				StartSignal: "ZCZC",
				SendID:      "TMQ2611",
				SendTime:    "151524",
			},
			PriorityAndSender: PriorityAndSender{
				Priority: "FF",
				Sender:   "ZBTJZPZX",
			},
			TimeAndReceiver: TimeAndReceiver{
				Time:     "151524",
				Receiver: "ZGGGZPZX",
			},
			Body:         "Test message",
			ReceivedTime: time.Now(),
			Category:     "Test",
			BodyData:     nil,
		}
	})

	Describe("Marshalling and Unmarshalling", func() {
		It("should marshal and unmarshal correctly", func() {
			data, err := json.Marshal(original)
			Expect(err).NotTo(HaveOccurred())

			var unmarshalled AFTN
			err = json.Unmarshal(data, &unmarshalled)
			Expect(err).NotTo(HaveOccurred())

			Expect(unmarshalled.Header).To(Equal(original.Header))
			Expect(unmarshalled.PriorityAndSender).To(Equal(original.PriorityAndSender))
			Expect(unmarshalled.TimeAndReceiver).To(Equal(original.TimeAndReceiver))
			Expect(unmarshalled.Body).To(Equal(original.Body))
			Expect(unmarshalled.Category).To(Equal(original.Category))
		})
	})
})
