package translator

import (
	"strings"
	"testing"
)

func TestAnalyzeEordleConstraints(t *testing.T) {
	trans := &TextTranslator{}

	// Test 1: Standard Red squares
	inputRed := "🟩 🟥 🟥 🟥 🟥 GREEN\n🟨 🟥 🟥 🟥 🟥 YELLO\n🟥 🟥 🟥 🟥 🟥 REDDD"
	analysisRed := trans.AnalyzeEordle(inputRed)

	// Test 2: Standard Black squares
	inputBlack := "🟩 ⬛ ⬛ ⬛ ⬛ GREEN\n🟨 ⬛ ⬛ ⬛ ⬛ YELLO\n⬛ ⬛ ⬛ ⬛ ⬛ REDDD"
	analysisBlack := trans.AnalyzeEordle(inputBlack)

	// Test 3: Standard White squares
	inputWhite := "🟩 ⬜ ⬜ ⬜ ⬜ GREEN\n🟨 ⬜ ⬜ ⬜ ⬜ YELLO\n⬜ ⬜ ⬜ ⬜ ⬜ REDDD"
	analysisWhite := trans.AnalyzeEordle(inputWhite)

	if analysisRed != analysisBlack {
		t.Errorf("Black squares analysis did not match Red squares analysis.\nRed:\n%s\nBlack:\n%s", analysisRed, analysisBlack)
	}

	if analysisRed != analysisWhite {
		t.Errorf("White squares analysis did not match Red squares analysis.\nRed:\n%s\nWhite:\n%s", analysisRed, analysisWhite)
	}

	// Verify the constraints are correctly parsed
	if !strings.Contains(analysisRed, "Pattern: G____") {
		t.Errorf("Expected pattern G____, got: %s", analysisRed)
	}
	if !strings.Contains(analysisRed, "Present: G,Y") {
		t.Errorf("Expected present G,Y, got: %s", analysisRed)
	}
	if !strings.Contains(analysisRed, "Excluded: D,E,L,N,O,R") {
		t.Errorf("Expected excluded D,E,L,N,O,R, got: %s", analysisRed)
	}
}

func BenchmarkMarkdownToTelegramHTML(b *testing.B) {
	text := "This is a **bold** and _italic_ text with some `code` and \r\n```\nblock\n```\n\n### Heading"
	for i := 0; i < b.N; i++ {
		markdownToTelegramHTML(text)
	}
}
