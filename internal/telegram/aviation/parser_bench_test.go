package aviation

import "testing"

var (
	benchARRMessage = `ZCZC TMQ2526 141605
FF ZBTJZPZX
141604 ZBACZQZX
(ARR-JAE7433/A0132-RKSI-ZBTJ1604)
NNNN`

	benchDEPMessage = `ZCZC DEP5678 120915
DD KLAXZPZX
120914 KSFOZQZX
(DEP-ABC5678/A1234-ZBTJ1440-ZGGG)
NNNN`

	benchCNLMessage = `ZCZC CNL9012 150631
FF ZBTJZPZX
150630 ZBTJUOBK
(CNL-CCA9012-ZBTJ-ZGGG)
NNNN`

	benchDLAMessage = `ZCZC DLA3456 150631
FF ZBTJZPZX
150630 ZBTJUOBK
(DLA-CCA3456-ZBTJ1600-ZGGG0200)
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
)

func BenchmarkParseARR(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse(benchARRMessage)
	}
}

func BenchmarkParseFPL(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse(benchFPLMessage)
	}
}

func BenchmarkParseMixed(b *testing.B) {
	messages := []string{
		benchARRMessage,
		benchDEPMessage,
		benchCNLMessage,
		benchDLAMessage,
		benchFPLMessage,
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse(messages[i%len(messages)])
	}
}
