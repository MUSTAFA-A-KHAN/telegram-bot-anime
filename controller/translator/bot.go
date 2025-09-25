package translator

//Flash activity
//Add synonyms

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

			switch message.Text {
			case "start":
				startHandler(bot, update)
			case "sayUK":
				// Get the text from the message being replied to
				text := message.ReplyToMessage.Text

				// Call ReadItLoud to convert the text to speech (assuming it returns an Audio struct)
				Voice := translator.ReadItLoudUK(text)
				tgbotapi.NewMessage(chatID, Voice)

				// Example 1: Send a local file
				file, err := os.Open("outputUK.mp3")
				if err != nil {
					log.Fatal(err)
				}
				defer file.Close()

				doc := tgbotapi.NewDocument(chatID, tgbotapi.FileReader{
					Name:   "outputUK.mp3",
					Reader: file,
				})

				_, err = bot.Send(doc)
				if err != nil {
					log.Fatal(err)
				}
			case "sayUKFemale":
				// Get the text from the message being replied to
				text := message.ReplyToMessage.Text

				// Call ReadItLoud to convert the text to speech (assuming it returns an Audio struct)
				Voice := translator.ReadItLoudUKFemale(text)
				tgbotapi.NewMessage(chatID, Voice)

				// Example 1: Send a local file
				file, err := os.Open("outputUKFemale.mp3")
				if err != nil {
					log.Fatal(err)
				}
				defer file.Close()

				doc := tgbotapi.NewDocument(chatID, tgbotapi.FileReader{
					Name:   "outputUKFemale.mp3",
					Reader: file,
				})

				_, err = bot.Send(doc)
				if err != nil {
					log.Fatal(err)
				}
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
			case "sayMale":
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
			case "sayFemale":
				// Get the text from the message being replied to
				text := message.ReplyToMessage.Text

				// Call ReadItLoud to convert the text to speech (assuming it returns an Audio struct)
				Voice := translator.ReadItLoudFemale(text)
				tgbotapi.NewMessage(chatID, Voice)

				// Example 1: Send a local file
				file, err := os.Open("outputFemale.mp3")
				if err != nil {
					log.Fatal(err)
				}
				defer file.Close()

				doc := tgbotapi.NewDocument(chatID, tgbotapi.FileReader{
					Name:   "outputFemale.mp3",
					Reader: file,
				})

				_, err = bot.Send(doc)
				if err != nil {
					log.Fatal(err)
				}
			case "ar":
				// Translate to English
				text = message.ReplyToMessage.Text

				// Translate to Arabic
				arabicTranslation := translator.TranslateToArabic(text)

				response := fmt.Sprintf(" %s\n", arabicTranslation)

				msg := tgbotapi.NewMessage(chatID, response)
				_, err := bot.Send(msg)
				if err != nil {
					log.Printf("Error sending translation: %v", err)
				}
			case "en":
				text = message.ReplyToMessage.Text

				// Translate to English
				englishTranslation := translator.TranslateToEnglish(text)

				response := fmt.Sprintf("English: %s\n", englishTranslation)

				msg := tgbotapi.NewMessage(chatID, response)
				_, err := bot.Send(msg)
				if err != nil {
					log.Printf("Error sending translation: %v", err)
				}
			case "ru":
				text = message.ReplyToMessage.Text

				// Translate to English
				russianTranslation := translator.TranslateToRussian(text)

				response := fmt.Sprintf("Russian: %s\n", russianTranslation)

				msg := tgbotapi.NewMessage(chatID, response)
				_, err := bot.Send(msg)
				if err != nil {
					log.Printf("Error sending translation: %v", err)
				}
			case "fr":
				text = message.ReplyToMessage.Text

				// Translate to English
				frenchTranslation := translator.TranslateToFrench(text)

				response := fmt.Sprintf("French: %s\n", frenchTranslation)

				msg := tgbotapi.NewMessage(chatID, response)
				_, err := bot.Send(msg)
				if err != nil {
					log.Printf("Error sending translation: %v", err)
				}
			case "edge":
				msg := tgbotapi.NewMessage(chatID, "what ever you are gonna write here will be reposnded ")
				_, err := bot.Send(msg)
				if err != nil {
					log.Printf("Error sending translation: %v", err)
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
			case "syn":
				text = message.ReplyToMessage.Text

				// Translate to English
				synonyms := translator.GetSynonyms(text)

				response := fmt.Sprintf("Synonyms for the word %s \n:%s", text, synonyms)

				msg := tgbotapi.NewMessage(chatID, response)
				_, err := bot.Send(msg)
				if err != nil {
					log.Printf("Error sending translation: %v", err)
				}
			case "anto":
				text = message.ReplyToMessage.Text

				// Translate to English
				antonyms := translator.GetAntonyms(text)

				response := fmt.Sprintf("Antonyms for the word %s \n:%s", text, antonyms)

				msg := tgbotapi.NewMessage(chatID, response)
				_, err := bot.Send(msg)
				if err != nil {
					log.Printf("Error sending translation: %v", err)
				}
			case "define":
				text = message.ReplyToMessage.Text

				// Translate to English
				definition := translator.GetDefinition(text)

				response := fmt.Sprintf(" %s \n:%s", text, definition)

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
				}
				// else if message.Text != "" {
				// 	// Handle text messages
				// 	log.Printf("[%s] %s", message.From.UserName, message.Text)
				// 	textHandler(bot, update)
				// }
			}

		}
	}
}
