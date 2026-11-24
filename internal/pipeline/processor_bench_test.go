package pipeline

import (
	"caatsm/internal/telegram"
	"context"
	"testing"

	"go.uber.org/zap"
)

type benchStore struct{}

func (benchStore) InsertOne(context.Context, *telegram.Message) error     { return nil }
func (benchStore) InsertBatch(context.Context, []*telegram.Message) error { return nil }
func (benchStore) InsertRaw(context.Context, *telegram.Message) error     { return nil }

type benchPublisher struct{}

func (benchPublisher) Publish(context.Context, *telegram.Message) error { return nil }

var (
	benchARRRaw = []byte(`ZCZC TMQ2526 141605
FF ZBTJZPZX
141604 ZBACZQZX
(ARR-JAE7433/A0132-RKSI-ZBTJ1604)
NNNN`)

	benchDEPRaw = []byte(`ZCZC DEP5678 120915
DD KLAXZPZX
120914 KSFOZQZX
(DEP-ABC5678/A1234-ZBTJ1440-ZGGG)
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

func createBenchmarkProcessor() *Processor {
	return NewProcessor(
		telegram.NewRouter(),
		telegram.NewAFTNValidator(),
		benchStore{},
		benchPublisher{},
		nil,
		zap.NewNop(),
		Options{},
	)
}

func BenchmarkHandleARR(b *testing.B) {
	proc := createBenchmarkProcessor()
	ctx := context.Background()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = proc.Handle(ctx, benchARRRaw, "msg-arr-123")
	}
}

func BenchmarkHandleMixed(b *testing.B) {
	proc := createBenchmarkProcessor()
	ctx := context.Background()
	messages := [][]byte{benchARRRaw, benchDEPRaw, benchFPLRaw}
	msgIDs := []string{"msg-arr", "msg-dep", "msg-fpl"}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		idx := i % len(messages)
		_ = proc.Handle(ctx, messages[idx], msgIDs[idx])
	}
}
