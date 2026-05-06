package translator

import (
	"fmt"
	"html"
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

func markdownToTelegramHTML(text string) string {
	escaped := html.EscapeString(strings.ReplaceAll(text, "\r\n", "\n"))

	replacements := []struct {
		pattern string
		replace string
	}{
		{"(?s)```(?:[a-zA-Z0-9_-]+)?\\n?(.*?)```", `<pre>$1</pre>`},
		{`\*\*([^*\n]+)\*\*`, `<b>$1</b>`},
		{`__([^_\n]+)__`, `<b>$1</b>`},
		{`\*([^*\n]+)\*`, `<i>$1</i>`},
		{`_([^_\n]+)_`, `<i>$1</i>`},
		{"`([^`\n]+)`", `<code>$1</code>`},
	}

	for _, replacement := range replacements {
		escaped = regexp.MustCompile(replacement.pattern).ReplaceAllString(escaped, replacement.replace)
	}

	lines := strings.Split(escaped, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "### ") {
			lines[i] = strings.Replace(line, "### "+strings.TrimPrefix(trimmed, "### "), "<b>"+strings.TrimPrefix(trimmed, "### ")+"</b>", 1)
		} else if strings.HasPrefix(trimmed, "## ") {
			lines[i] = strings.Replace(line, "## "+strings.TrimPrefix(trimmed, "## "), "<b>"+strings.TrimPrefix(trimmed, "## ")+"</b>", 1)
		} else if strings.HasPrefix(trimmed, "# ") {
			lines[i] = strings.Replace(line, "# "+strings.TrimPrefix(trimmed, "# "), "<b>"+strings.TrimPrefix(trimmed, "# ")+"</b>", 1)
		}
	}

	return strings.Join(lines, "\n")
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
