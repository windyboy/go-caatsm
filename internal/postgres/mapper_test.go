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

	It("maps all 17 telegram fields to DB row columns", func() {
		now := time.Now().UTC()
		id := uuid.NewString()
		original := &telegram.Message{
			UUID:               id,
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
			ErrorReason:        "",
		}

		row, err := toDBRow(original)
		Expect(err).NotTo(HaveOccurred())
		Expect(row).To(HaveLen(17))
		Expect(row[0].(uuid.UUID).String()).To(Equal(id))
		Expect(row[1]).To(Equal("TMQ1324"))
		Expect(row[2]).To(Equal("150631"))
		Expect(row[3]).To(Equal("FF"))
		Expect(row[4]).To(Equal("ZBTJZPZX"))
		Expect(row[5]).To(Equal("150630 ZBACZQZX"))
		Expect(row[6]).To(Equal("ZBTJYOYX"))
		Expect(row[7]).To(Equal("150630"))
		Expect(row[8]).To(Equal("FPL"))
		Expect(row[9]).To(Equal("raw telegram"))
		Expect(string(row[10].([]byte))).To(ContainSubstring(`"key":"value"`))
		Expect(row[11]).To(Equal("parsed"))
		Expect(row[12]).To(Equal(""))
		Expect(row[13]).To(Equal(now))
		Expect(row[14]).To(Equal(now))
		Expect(row[15]).To(Equal(now))
		Expect(row[16]).To(BeTrue())
	})
})
