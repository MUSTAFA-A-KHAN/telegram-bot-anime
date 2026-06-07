package translator

import "testing"

func BenchmarkMarkdownToTelegramHTML(b *testing.B) {
	text := "This is a **bold** and _italic_ text with some `code` and \r\n```\nblock\n```\n\n### Heading"
	for i := 0; i < b.N; i++ {
		markdownToTelegramHTML(text)
	}
}
