package postgres

import (
	"caatsm/internal/telegram"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
)

func createBenchmarkTelegram() *telegram.Message {
	return &telegram.Message{
		UUID:              uuid.New().String(),
		MessageID:         "TMQ1234",
		DateTime:          "150631",
		PriorityIndicator: "FF",
		PrimaryAddress:    "ZBTJZPZX",
		Category:          "ARR",
		Content:           "ZCZC TMQ1234 150631\nFF ZBTJZPZX\n(ARR-ABC123-ZBTJ-ZGGG)\nNNNN",
		Status:            telegram.StatusParsed,
		Parsed:            true,
		ReceivedAt:        time.Now(),
		ParsedAt:          time.Now(),
	}
}

func BenchmarkToDBRow(b *testing.B) {
	msg := createBenchmarkTelegram()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = toDBRow(msg)
	}
}

func BenchmarkToDBRowBatch100(b *testing.B) {
	msgs := make([]*telegram.Message, 100)
	for i := range msgs {
		m := createBenchmarkTelegram()
		m.MessageID = "TMQ" + strconv.Itoa(1000+i)
		msgs[i] = m
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, m := range msgs {
			_, _ = toDBRow(m)
		}
	}
}
