package translator

import (
	"fmt"
	"io"
	"log"
	"net/http"
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

func textHandler(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	text := update.Message.Text

	// Translate to English
	englishTranslation := translator.TranslateToEnglish(text)

	// Translate to Arabic
	arabicTranslation := translator.TranslateToArabic(text)

	response := fmt.Sprintf("English: %s\n\nArabic: %s", englishTranslation, arabicTranslation)

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, response)
	_, err := bot.Send(msg)
	if err != nil {
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
	_, err = bot.Send(msg)
	if err != nil {
		log.Printf("Error sending voice translation: %v", err)
	}
}
