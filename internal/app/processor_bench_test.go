package app

import (
	"caatsm/internal/adapter/dto"
	"caatsm/internal/adapter/parser"
	"caatsm/internal/adapter/parser/weather"
	"caatsm/internal/infra/config"
	"caatsm/internal/infra/telemetry"
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

// Mock implementations for benchmarking
type mockRepository struct{}

func (m *mockRepository) InsertOne(ctx context.Context, msg *dto.ParsedTelegram) error {
	return nil
}

func (m *mockRepository) InsertBatch(ctx context.Context, msgs []*dto.ParsedTelegram) error {
	return nil
}

func (m *mockRepository) InsertRaw(ctx context.Context, msg *dto.ParsedTelegram) error {
	return nil
}

type mockPublisher struct{}

func (m *mockPublisher) Publish(message interface{}) error {
	return nil
}

// Benchmark data
var (
	benchARRRaw = []byte(`ZCZC TMQ2526 141605
FF ZBTJZPZX
141604 ZBACZQZX
(ARR-JAE7433/A0132-RKSI-ZBTJ1604)
NNNN`)

	benchDEPRaw = []byte(`ZCZC DEP5678 120915
DD KLAXZPZX
120914 KSFOZQZX
(DEP-ABC5678-A1234-ZBTJ1440-ZGGG)
NNNN`)

	benchFPLRaw = []byte(`ZCZC TMQ2617 142150
GG ZBTJZPZX
150551 ZBTJUOBK
(FPL-OKA2861-IS
-MA60/M-SHID/C
-ZBTJ0030
-K0420S0450 CG J1 FZ
-ZSYT0100  ZSQD ZYTL
-REG/B3710 SEL/ RMK/TCAS )
NNNN`)
)

// createBenchmarkProcessor creates a processor with mocks for benchmarking
func createBenchmarkProcessor() *MessageProcessor {
	weatherParser := weather.NewWeatherParser()
	aviationParser := parser.ProvideParser(weatherParser)
	mockRepo := &mockRepository{}
	mockPub := &mockPublisher{}
	logger := zap.NewNop()
	recorder := telemetry.NewNoop()
	cfg := &config.Config{
		AFTN: config.AFTNConfig{
			ValidationEnabled:          false,
			MessageGapThreshold:        2 * time.Minute,
			EnableSequenceGapDetection: true,
		},
	}

	return NewMessageProcessor(aviationParser, mockRepo, mockPub, recorder, logger, cfg)
}

// BenchmarkHandleARR benchmarks processing ARR messages end-to-end
func BenchmarkHandleARR(b *testing.B) {
	processor := createBenchmarkProcessor()
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = processor.Handle(ctx, benchARRRaw, "msg-arr-123")
	}
}

// BenchmarkHandleDEP benchmarks processing DEP messages end-to-end
func BenchmarkHandleDEP(b *testing.B) {
	processor := createBenchmarkProcessor()
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = processor.Handle(ctx, benchDEPRaw, "msg-dep-123")
	}
}

// BenchmarkHandleFPL benchmarks processing FPL messages end-to-end
func BenchmarkHandleFPL(b *testing.B) {
	processor := createBenchmarkProcessor()
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = processor.Handle(ctx, benchFPLRaw, "msg-fpl-123")
	}
}

// BenchmarkHandleMixed benchmarks processing a mix of message types
func BenchmarkHandleMixed(b *testing.B) {
	processor := createBenchmarkProcessor()
	ctx := context.Background()
	messages := [][]byte{benchARRRaw, benchDEPRaw, benchFPLRaw}
	msgIDs := []string{"msg-arr", "msg-dep", "msg-fpl"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		idx := i % len(messages)
		_ = processor.Handle(ctx, messages[idx], msgIDs[idx])
	}
}

// BenchmarkHandleParseOnly benchmarks parsing without persistence/publishing
// This isolates parser performance
func BenchmarkHandleParseOnly(b *testing.B) {
	weatherParser := weather.NewWeatherParser()
	aviationParser := parser.ProvideParser(weatherParser)
	// Use a repository that does nothing
	mockRepo := &mockRepository{}
	// Use a publisher that does nothing
	mockPub := &mockPublisher{}
	logger := zap.NewNop()
	recorder := telemetry.NewNoop()
	cfg := &config.Config{
		AFTN: config.AFTNConfig{
			ValidationEnabled:          false,
			MessageGapThreshold:        2 * time.Minute,
			EnableSequenceGapDetection: true,
		},
	}

	processor := NewMessageProcessor(aviationParser, mockRepo, mockPub, recorder, logger, cfg)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = processor.Handle(ctx, benchARRRaw, "msg-parse-only")
	}
}

