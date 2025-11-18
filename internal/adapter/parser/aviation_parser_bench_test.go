package parser

import (
	"testing"
)

// Sample messages for benchmarking
var (
	benchARRMessage = `ZCZC TMQ2526 141605
FF ZBTJZPZX
141604 ZBACZQZX
(ARR-JAE7433/A0132-RKSI-ZBTJ1604)
NNNN`

	benchDEPMessage = `ZCZC DEP5678 120915
DD KLAXZPZX
120914 KSFOZQZX
(DEP-ABC5678-A1234-ZBTJ1440-ZGGG)
NNNN`

	benchCNLMessage = `ZCZC CNL9012 150631
FF ZBTJZPZX
(CNL-CCA9012-ZBTJ-ZGGG)
NNNN`

	benchDLAMessage = `ZCZC DLA3456 150631
FF ZBTJZPZX
(DLA-CCA3456-A1234-ZBTJ1600-ZGGG0200)
NNNN`

	benchFPLMessage = `ZCZC TMQ2617 142150
GG ZBTJZPZX
150551 ZBTJUOBK
(FPL-OKA2861-IS
-MA60/M-SHID/C
-ZBTJ0030
-K0420S0450 CG J1 FZ
-ZSYT0100  ZSQD ZYTL
-REG/B3710 SEL/ RMK/TCAS )
NNNN`

	benchComplexFPLMessage = `ZCZC FPL7890 150631
FF ZBTJZPZX
(FPL-JAE7433-IS
-B744/H-SXIRPZJWY/S
-ZBTJ1755
-K0926S0920 CG A326 VYK W80 HUR B339 GM A575 MANSA/K0919S0980
-EDDF0948 EDDK
-EET/ZMUB0100 UNKL0236
REG/B2422 SEL/JLAD
NAV/RNAV1 RNAV5 RNP4
RMK/AGCS EQUIPPED)
NNNN`
)

// BenchmarkParseARR benchmarks parsing ARR messages
func BenchmarkParseARR(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Parse(benchARRMessage)
	}
}

// BenchmarkParseDEP benchmarks parsing DEP messages
func BenchmarkParseDEP(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Parse(benchDEPMessage)
	}
}

// BenchmarkParseCNL benchmarks parsing CNL messages
func BenchmarkParseCNL(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Parse(benchCNLMessage)
	}
}

// BenchmarkParseDLA benchmarks parsing DLA messages
func BenchmarkParseDLA(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Parse(benchDLAMessage)
	}
}

// BenchmarkParseFPL benchmarks parsing simple FPL messages
func BenchmarkParseFPL(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Parse(benchFPLMessage)
	}
}

// BenchmarkParseComplexFPL benchmarks parsing complex FPL messages with extensive route and metadata
func BenchmarkParseComplexFPL(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Parse(benchComplexFPLMessage)
	}
}

// BenchmarkParseHeader benchmarks header parsing only
func BenchmarkParseHeader(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseHeader(benchARRMessage)
	}
}

// BenchmarkParseBody benchmarks body parsing only (ARR)
func BenchmarkParseBody(b *testing.B) {
	body := `(ARR-JAE7433/A0132-RKSI-ZBTJ1604)`
	parser := NewBodyParser(body)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = parser.Parse()
	}
}

// BenchmarkParseMixed benchmarks parsing a mix of message types
func BenchmarkParseMixed(b *testing.B) {
	messages := []string{
		benchARRMessage,
		benchDEPMessage,
		benchCNLMessage,
		benchDLAMessage,
		benchFPLMessage,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := messages[i%len(messages)]
		_, _ = Parse(msg)
	}
}

