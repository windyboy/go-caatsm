package postgres

import (
	"caatsm/internal/telegram"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Row Mapper", func() {
	It("generates a UUID when missing and rejects nil messages", func() {
		_, err := toDBRow(nil)
		Expect(err).To(HaveOccurred())

		row, err := toDBRow(&telegram.Message{})
		Expect(err).NotTo(HaveOccurred())
		Expect(row).To(HaveLen(len(telegramColumns)))

		value, ok := row[0].(uuid.UUID)
		Expect(ok).To(BeTrue())
		Expect(value).NotTo(Equal(uuid.Nil))
	})

	It("round-trips telegram fields accurately", func() {
		now := time.Now().UTC()
		original := &telegram.Message{
			UUID:               uuid.NewString(),
			MessageID:          "TMQ1324",
			DateTime:           "150631",
			PriorityIndicator:  "FF",
			PrimaryAddress:     "ZBTJZPZX",
			SecondaryAddresses: "150630 ZBACZQZX",
			Originator:         "ZBTJYOYX",
			OriginatorDateTime: "150630",
			Category:           "FPL",
			Content:            "raw telegram",
			BodyData:           map[string]string{"key": "value"},
			ReceivedAt:         now,
			ParsedAt:           now,
			DispatchedAt:       now,
			NeedDispatch:       true,
			Status:             telegram.StatusParsed,
		}

		row, err := toDBRow(original)
		Expect(err).NotTo(HaveOccurred())
		Expect(row).To(HaveLen(17))

		roundTrip, err := fromDBRow(row)
		Expect(err).NotTo(HaveOccurred())
		Expect(roundTrip.UUID).To(Equal(original.UUID))
		Expect(roundTrip.MessageID).To(Equal(original.MessageID))
		Expect(roundTrip.NeedDispatch).To(Equal(original.NeedDispatch))
		Expect(roundTrip.Status).To(Equal(original.Status))
	})
})
