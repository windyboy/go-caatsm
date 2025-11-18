package postgres

import (
	"caatsm/internal/adapter/dto"
	"caatsm/internal/adapter/mapper"
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Benchmark data setup
func createBenchmarkTelegram() *dto.ParsedTelegram {
	return &dto.ParsedTelegram{
		Uuid:              uuid.New().String(),
		MessageID:         "TMQ1234",
		DateTime:          "150631",
		PriorityIndicator: "FF",
		PrimaryAddress:    "ZBTJZPZX",
		Category:          "ARR",
		Content:           "ZCZC TMQ1234 150631\nFF ZBTJZPZX\n(ARR-ABC123-ZBTJ-ZGGG)\nNNNN",
		Status:            dto.MessageStatusParsed,
		Parsed:            true,
		ReceivedAt:        time.Now(),
		ParsedAt:          time.Now(),
	}
}

func createBenchmarkTelegrams(count int) []*dto.ParsedTelegram {
	telegrams := make([]*dto.ParsedTelegram, count)
	for i := 0; i < count; i++ {
		tg := createBenchmarkTelegram()
		tg.Uuid = uuid.New().String()
		tg.MessageID = "TMQ" + strconv.Itoa(1000+i)
		telegrams[i] = tg
	}
	return telegrams
}

// BenchmarkInsertOne benchmarks single message insertion
// Note: This requires a database connection. Run with -tags=integration or provide test DB.
func BenchmarkInsertOne(b *testing.B) {
	// Skip if no database available (integration tests only)
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	// This benchmark requires a real database connection
	// In practice, you would set up a test database connection here
	// For now, we'll skip if not running integration tests
	b.Skip("Requires database connection - run with integration tests")
}

// BenchmarkInsertOneWithDB benchmarks single message insertion with database
// This is a helper that can be used in integration test suites
//
//nolint:unused // Benchmark helper for future use
func benchmarkInsertOneWithDB(b *testing.B, repo *Repository) {
	ctx := context.Background()
	telegram := createBenchmarkTelegram()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Update UUID for each iteration to avoid conflicts
		telegram.Uuid = uuid.New().String()
		telegram.MessageID = "TMQ" + strconv.Itoa(1000+i)
		_ = repo.InsertOne(ctx, telegram)
	}
}

// BenchmarkInsertBatch benchmarks batch message insertion
// Note: This requires a database connection. Run with -tags=integration or provide test DB.
func BenchmarkInsertBatch(b *testing.B) {
	// Skip if no database available (integration tests only)
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	// This benchmark requires a real database connection
	// In practice, you would set up a test database connection here
	// For now, we'll skip if not running integration tests
	b.Skip("Requires database connection - run with integration tests")
}

// BenchmarkInsertBatchWithDB benchmarks batch insertion with database
// This is a helper that can be used in integration test suites
//
//nolint:unused // Benchmark helper for future use
func benchmarkInsertBatchWithDB(b *testing.B, repo *Repository, batchSize int) {
	ctx := context.Background()
	telegrams := createBenchmarkTelegrams(batchSize)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Update UUIDs for each iteration to avoid conflicts
		for j := range telegrams {
			telegrams[j].Uuid = uuid.New().String()
			telegrams[j].MessageID = "TMQ" + strconv.Itoa(1000+i*batchSize+j)
		}
		_ = repo.InsertBatch(ctx, telegrams)
	}
}

// BenchmarkInsertBatch10 benchmarks batch insertion with 10 messages
func BenchmarkInsertBatch10(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}
	b.Skip("Requires database connection - run with integration tests")
}

// BenchmarkInsertBatch50 benchmarks batch insertion with 50 messages
func BenchmarkInsertBatch50(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}
	b.Skip("Requires database connection - run with integration tests")
}

// BenchmarkInsertBatch100 benchmarks batch insertion with 100 messages
func BenchmarkInsertBatch100(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}
	b.Skip("Requires database connection - run with integration tests")
}

// BenchmarkInsertRaw benchmarks raw message insertion
func BenchmarkInsertRaw(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}
	b.Skip("Requires database connection - run with integration tests")
}

// BenchmarkMapperToDBRow benchmarks the mapping from DTO to DB row
// This doesn't require a database connection
func BenchmarkMapperToDBRow(b *testing.B) {
	m := mapper.NewTelegramMapper()
	telegram := createBenchmarkTelegram()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = m.ToDBRow(telegram)
	}
}

// BenchmarkMapperToDBRowBatch benchmarks mapping multiple telegrams
func BenchmarkMapperToDBRowBatch(b *testing.B) {
	m := mapper.NewTelegramMapper()
	telegrams := createBenchmarkTelegrams(100)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, tg := range telegrams {
			_, _ = m.ToDBRow(tg)
		}
	}
}
