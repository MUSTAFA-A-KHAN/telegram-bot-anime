package translator

import (
	"testing"
)

func BenchmarkAnalyzeEordle(b *testing.B) {
	t := NewTextTranslator()
	puzzle := `
	🟩 🟩 ⬜ ⬜ ⬜ HELLO
	🟩 🟩 🟨 ⬜ ⬜ HEART
	`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t.AnalyzeEordle(puzzle)
	}
}
