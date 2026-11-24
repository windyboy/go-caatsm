package postgres

import (
	"caatsm/internal/telegram"
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var _ = Describe("Repository guards", func() {
	It("validates nil messages and empty batches safely", func() {
		repo := NewRepository(nil, nil, zap.NewNop())
		ctx := context.Background()

		Expect(repo.InsertOne(ctx, nil)).To(HaveOccurred())
		Expect(repo.InsertOne(ctx, &telegram.Message{})).To(HaveOccurred())
		Expect(repo.InsertBatch(ctx, nil)).To(Succeed())
		Expect(repo.InsertBatch(ctx, []*telegram.Message{})).To(Succeed())
		Expect(repo.InsertBatch(ctx, []*telegram.Message{{}})).To(HaveOccurred())
		Expect(repo.InsertRaw(ctx, nil)).To(HaveOccurred())
		Expect(repo.InsertRaw(ctx, &telegram.Message{})).To(HaveOccurred())
	})

	It("matches legacy and structured weather deduplication keys only when normalized content matches", func() {
		metarMsg := &telegram.Message{
			Protocol:  telegram.ProtocolWeather,
			Category:  "METAR",
			MessageID: "METAR-KJFK-20261125120000",
			Content:   "METAR KJFK 251200Z 35012KT 10SM FEW020 25/18 Q1013=",
		}
		Expect(weatherCandidateMessageIDs(metarMsg)).To(Equal([]string{
			"METAR-KJFK-20261125120000",
			"KJFK-20261125120000",
		}))

		corMsg := &telegram.Message{
			Protocol:  telegram.ProtocolWeather,
			Category:  "METAR",
			MessageID: "METAR-COR-KJFK-20261125120000",
			Content:   "METAR COR KJFK 251200Z 36015KT 10SM FEW020 25/18 Q1013=",
		}
		Expect(weatherCandidateMessageIDs(corMsg)).To(Equal([]string{
			"METAR-COR-KJFK-20261125120000",
			"KJFK-20261125120000",
		}))

		// Legacy replay with CRLF/surrounding whitespace is recognized as duplicate.
		existingLegacy := []string{"  METAR KJFK 251200Z 35012KT 10SM FEW020 25/18 Q1013=\r\n"}
		Expect(matchesWeatherContent(existingLegacy, metarMsg.Content)).To(BeTrue())

		// COR report against original legacy row has different content -> not a duplicate.
		Expect(matchesWeatherContent(existingLegacy, corMsg.Content)).To(BeFalse())

		// Two distinct COR reports at the same station and issue time -> not duplicates.
		secondCor := "METAR COR KJFK 251200Z 01018KT 10SM FEW020 25/18 Q1013="
		Expect(matchesWeatherContent([]string{corMsg.Content}, secondCor)).To(BeFalse())
		Expect(matchesWeatherContent([]string{corMsg.Content}, " "+corMsg.Content+"\n")).To(BeTrue())
	})
})
