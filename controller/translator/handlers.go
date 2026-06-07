package translator

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var translator = NewTextTranslator()

func startHandler(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	user := update.Message.From
	text := fmt.Sprintf("Hi %s! I'm a simple translator bot. Send me any text and I'll translate it to English and Arabic.", user.FirstName)

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("Error sending start message: %v", err)
	}
}

func sendMarkdownMessage(bot *tgbotapi.BotAPI, msg tgbotapi.MessageConfig) error {
	plainText := msg.Text
	msg.Text = markdownToTelegramHTML(msg.Text)
	msg.ParseMode = tgbotapi.ModeHTML
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending formatted message, retrying as plain text: %v", err)
		msg.Text = plainText
		msg.ParseMode = ""
		_, fallbackErr := bot.Send(msg)
		return fallbackErr
	}

	return nil
}

var markdownReplacements = []struct {
	re      *regexp.Regexp
	replace string
}{
	{regexp.MustCompile("(?s)```(?:[a-zA-Z0-9_-]+)?\\n?(.*?)```"), `<pre>$1</pre>`},
	{regexp.MustCompile(`\*\*([^*\n]+)\*\*`), `<b>$1</b>`},
	{regexp.MustCompile(`__([^_\n]+)__`), `<b>$1</b>`},
	{regexp.MustCompile(`\*([^*\n]+)\*`), `<i>$1</i>`},
	{regexp.MustCompile(`_([^_\n]+)_`), `<i>$1</i>`},
	{regexp.MustCompile("`([^`\n]+)`"), `<code>$1</code>`},
}

func markdownToTelegramHTML(text string) string {
	var sb strings.Builder
	sb.Grow(len(text) + len(text)/4)

	// Combine \r\n replacement and html.EscapeString
	for i := 0; i < len(text); i++ {
		if text[i] == '\r' && i+1 < len(text) && text[i+1] == '\n' {
			continue // skip \r
		}
		switch text[i] {
		case '&':
			sb.WriteString("&amp;")
		case '\'':
			sb.WriteString("&#39;")
		case '<':
			sb.WriteString("&lt;")
		case '>':
			sb.WriteString("&gt;")
		case '"':
			sb.WriteString("&#34;")
		default:
			sb.WriteByte(text[i])
		}
	}

	escaped := sb.String()

	for _, replacement := range markdownReplacements {
		escaped = replacement.re.ReplaceAllString(escaped, replacement.replace)
	}

	sb.Reset()
	sb.Grow(len(escaped) + len(escaped)/10)

	start := 0
	for i := 0; i <= len(escaped); i++ {
		if i == len(escaped) || escaped[i] == '\n' {
			line := escaped[start:i]
			if start > 0 {
				sb.WriteByte('\n')
			}

			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "### ") {
				content := strings.TrimPrefix(trimmed, "### ")
				sb.WriteString(strings.Replace(line, "### "+content, "<b>"+content+"</b>", 1))
			} else if strings.HasPrefix(trimmed, "## ") {
				content := strings.TrimPrefix(trimmed, "## ")
				sb.WriteString(strings.Replace(line, "## "+content, "<b>"+content+"</b>", 1))
			} else if strings.HasPrefix(trimmed, "# ") {
				content := strings.TrimPrefix(trimmed, "# ")
				sb.WriteString(strings.Replace(line, "# "+content, "<b>"+content+"</b>", 1))
			} else {
				sb.WriteString(line)
			}
			start = i + 1
		}
	}

	return sb.String()
}

func textHandler(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	text := update.Message.Text

	// Translate to English
	englishTranslation := translator.TranslateToEnglish(text)

	// Translate to Arabic
	arabicTranslation := translator.TranslateToArabic(text)

	response := fmt.Sprintf("English: %s\n\nArabic: %s", englishTranslation, arabicTranslation)

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, response)
	if err := sendMarkdownMessage(bot, msg); err != nil {
		log.Printf("Error sending translation: %v", err)
	}
}

func voiceHandler(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	voice := update.Message.Voice

	// Get file URL
	file, err := bot.GetFile(tgbotapi.FileConfig{FileID: voice.FileID})
	if err != nil {
		log.Printf("Error getting file: %v", err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Error processing voice message")
		bot.Send(msg)
		return
	}

	// Download the voice file
	fileURL := file.Link(BotToken)
	resp, err := http.Get(fileURL)
	if err != nil {
		log.Printf("Error downloading voice file: %v", err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Error downloading voice message")
		bot.Send(msg)
		return
	}
	defer resp.Body.Close()

	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading voice data: %v", err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Error reading voice message")
		bot.Send(msg)
		return
	}

	// Transcribe the audio
	transcribedText := translator.WriteITDown(audioData, voice.MimeType)

	if strings.Contains(transcribedText, "Error") || strings.Contains(transcribedText, "failed") {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Transcription failed: %s", transcribedText))
		bot.Send(msg)
		return
	}

	// Translate the transcribed text
	englishTranslation := translator.TranslateToEnglish(transcribedText)
	arabicTranslation := translator.TranslateToArabic(transcribedText)

	response := fmt.Sprintf("Transcribed: %s\n\nEnglish: %s\n\nArabic: %s", transcribedText, englishTranslation, arabicTranslation)

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, response)
	if err := sendMarkdownMessage(bot, msg); err != nil {
		log.Printf("Error sending voice translation: %v", err)
	}
}
