package translator

import (
	"fmt"
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func Bot() {
	if BotToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable is not set")
	}

	bot, err := tgbotapi.NewBotAPI(BotToken)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			message := update.Message
			chatID := message.Chat.ID
			text := message.Text

			log.Printf("[%s] %s", message.From.UserName, message.Text)

			switch message.Command() {
			case "start":
				startHandler(bot, update)
			case "say":
				// Get the text from the message being replied to
				text := message.ReplyToMessage.Text

				// Call ReadItLoud to convert the text to speech (assuming it returns an Audio struct)
				Voice := translator.ReadItLoud(text)
				tgbotapi.NewMessage(chatID, Voice)

				// Example 1: Send a local file
				file, err := os.Open("output.mp3")
				if err != nil {
					log.Fatal(err)
				}
				defer file.Close()

				doc := tgbotapi.NewDocument(chatID, tgbotapi.FileReader{
					Name:   "output.mp3",
					Reader: file,
				})

				_, err = bot.Send(doc)
				if err != nil {
					log.Fatal(err)
				}

			case "translate":
				// Translate to English
				text = message.ReplyToMessage.Text
				englishTranslation := translator.TranslateToEnglish(text)

				// Translate to Arabic
				arabicTranslation := translator.TranslateToArabic(text)

				response := fmt.Sprintf("English: %s\n\nArabic: %s", englishTranslation, arabicTranslation)

				msg := tgbotapi.NewMessage(chatID, response)
				_, err := bot.Send(msg)
				if err != nil {
					log.Printf("Error sending translation: %v", err)
				}

			default:
				if message.Voice != nil {
					// Handle voice messages
					log.Printf("[%s] sent a voice message", message.From.UserName)
					voiceHandler(bot, update)
				} else if message.Text != "" {
					// Handle text messages
					log.Printf("[%s] %s", message.From.UserName, message.Text)
					textHandler(bot, update)
				}
			}

		}
	}
}
