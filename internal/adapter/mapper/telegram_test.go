package mapper

import (
	"time"

	"caatsm/internal/adapter/dto"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TelegramMapper", func() {
	var mapper *TelegramMapper

	BeforeEach(func() {
		mapper = NewTelegramMapper()
	})

	Describe("ToDBRow", func() {
		It("generates a UUID when missing", func() {
			msg := &dto.ParsedTelegram{}

			row, err := mapper.ToDBRow(msg)
			Expect(err).NotTo(HaveOccurred())

			value, ok := row[0].(uuid.UUID)
			Expect(ok).To(BeTrue())
			Expect(value).NotTo(Equal(uuid.Nil))
		})
	})

	Describe("FromDBRow", func() {
		It("round-trips telegram data", func() {
			now := time.Now().UTC()
			original := &dto.ParsedTelegram{
				Uuid:               uuid.NewString(),
				MessageID:          "TMQ1324",
				DateTime:           "150631",
				PriorityIndicator:  "FF",
				PrimaryAddress:     "ZBTJZPZX",
				SecondaryAddresses: "150630 ZBACZQZX",
				Originator:         "ORIGIN",
				OriginatorDateTime: "150630",
				Category:           "FPL",
				Content:            "raw telegram",
				BodyData:           map[string]string{"key": "value"},
				ReceivedAt:         now,
				ParsedAt:           now,
				DispatchedAt:       now,
				NeedDispatch:       true,
				Status:             dto.MessageStatusParsed,
			}

			row, err := mapper.ToDBRow(original)
			Expect(err).NotTo(HaveOccurred())
			Expect(row).To(HaveLen(17))

			roundTrip, err := mapper.FromDBRow(row)
			Expect(err).NotTo(HaveOccurred())

			Expect(roundTrip.Uuid).To(Equal(original.Uuid))
			Expect(roundTrip.MessageID).To(Equal(original.MessageID))
			Expect(roundTrip.NeedDispatch).To(Equal(original.NeedDispatch))
			Expect(roundTrip.Status).To(Equal(original.Status))
		})
	})
})
