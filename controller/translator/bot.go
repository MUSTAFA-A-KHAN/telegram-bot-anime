package translator

//Flash activity
//Add synonyms

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/controller/translator/utilities"
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
			cmd := normalizeCommand(message.Text, bot.Self.UserName)
			if message.ReplyToMessage != nil {
				switch cmd {
				case "start":
					startHandler(bot, update)
				case "sayaifemale":
					text := ""
					if message.ReplyToMessage != nil && len(message.ReplyToMessage.Photo) > 0 {

						photo := message.ReplyToMessage.Photo
						if len(photo) == 0 {
							log.Printf("Error: No photo found in reply")
							break
						}
						text = writeImage(chatID, bot, photo)
					} else if message.ReplyToMessage != nil {
						// Get the text from the message being replied to
						text = message.ReplyToMessage.Text
					}

					// Call ReadItLoud to convert the text to speech (assuming it returns an Audio struct)
					Voice := translator.TextToSpeechElevenLabsFemale(text)
					tgbotapi.NewMessage(chatID, Voice)

					// Example 1: Send a local file
					file, err := os.Open("outputElevenLabFemale.mp3")
					if err != nil {
						log.Print(err)
					}
					defer file.Close()

					doc := tgbotapi.NewVoice(chatID, tgbotapi.FileReader{
						Name:   "outputElevenLabFemale.mp3",
						Reader: file,
					})
					doc.ReplyToMessageID = message.MessageID

					_, err = bot.Send(doc)
					if err != nil {
						log.Print(err)
					}
				case "sayai":
					text := ""
					if message.ReplyToMessage != nil && len(message.ReplyToMessage.Photo) > 0 {

						photo := message.ReplyToMessage.Photo
						if len(photo) == 0 {
							log.Printf("Error: No photo found in reply")
							break
						}
						text = writeImage(chatID, bot, photo)
					} else {
						// Get the text from the message being replied to
						text = message.ReplyToMessage.Text
					}

					// Call ReadItLoud to convert the text to speech (assuming it returns an Audio struct)
					Voice := translator.TextToSpeechElevenLabs(text)
					tgbotapi.NewMessage(chatID, Voice)

					// Example 1: Send a local file
					file, err := os.Open("outputElevenLab.mp3")
					if err != nil {
						log.Print(err)
					}
					defer file.Close()

					doc := tgbotapi.NewVoice(chatID, tgbotapi.FileReader{
						Name:   "outputElevenLab.mp3",
						Reader: file,
					})
					doc.ReplyToMessageID = message.MessageID

					_, err = bot.Send(doc)
					if err != nil {
						log.Print(err)
					}
				case "sayaiuk":
					text := ""
					if message.ReplyToMessage != nil && len(message.ReplyToMessage.Photo) > 0 {

						photo := message.ReplyToMessage.Photo
						if len(photo) == 0 {
							log.Printf("Error: No photo found in reply")
							break
						}
						text = writeImage(chatID, bot, photo)
					} else {
						// Get the text from the message being replied to
						text = message.ReplyToMessage.Text
					}

					// Call ReadItLoud to convert the text to speech (assuming it returns an Audio struct)
					Voice := translator.TextToSpeechElevenLabsUK(text)
					tgbotapi.NewMessage(chatID, Voice)

					// Example 1: Send a local file
					file, err := os.Open("outputElevenLabUK.mp3")
					if err != nil {
						log.Print(err)
					}
					defer file.Close()

					doc := tgbotapi.NewVoice(chatID, tgbotapi.FileReader{
						Name:   "outputElevenLabUK.mp3",
						Reader: file,
					})
					doc.ReplyToMessageID = message.MessageID

					_, err = bot.Send(doc)
					if err != nil {
						log.Print(err)
					}
				case "sayuk":
					text := ""
					if message.ReplyToMessage != nil && len(message.ReplyToMessage.Photo) > 0 {

						photo := message.ReplyToMessage.Photo
						if len(photo) == 0 {
							log.Printf("Error: No photo found in reply")
							break
						}
						text = writeImage(chatID, bot, photo)
					} else {
						// Get the text from the message being replied to
						text = message.ReplyToMessage.Text
					}

					// Call ReadItLoud to convert the text to speech (assuming it returns an Audio struct)
					Voice := translator.ReadItLoudUK(text)
					tgbotapi.NewMessage(chatID, Voice)

					// Example 1: Send a local file
					file, err := os.Open("outputUK.mp3")
					if err != nil {
						log.Print(err)
					}
					defer file.Close()

					doc := tgbotapi.NewVoice(chatID, tgbotapi.FileReader{
						Name:   "outputUK.mp3",
						Reader: file,
					})
					doc.ReplyToMessageID = message.MessageID

					_, err = bot.Send(doc)
					if err != nil {
						log.Print(err)
					}
				case "sayukfemale":
					text := ""
					if message.ReplyToMessage != nil && len(message.ReplyToMessage.Photo) > 0 {

						photo := message.ReplyToMessage.Photo
						if len(photo) == 0 {
							log.Printf("Error: No photo found in reply")
							break
						}
						text = writeImage(chatID, bot, photo)
					} else {
						// Get the text from the message being replied to
						text = message.ReplyToMessage.Text
					}

					// Call ReadItLoud to convert the text to speech (assuming it returns an Audio struct)
					Voice := translator.ReadItLoudUKFemale(text)
					tgbotapi.NewMessage(chatID, Voice)

					// Example 1: Send a local file
					file, err := os.Open("outputUKFemale.mp3")
					if err != nil {
						log.Print(err)
					}
					defer file.Close()

					doc := tgbotapi.NewVoice(chatID, tgbotapi.FileReader{
						Name:   "outputUKFemale.mp3",
						Reader: file,
					})
					doc.ReplyToMessageID = message.MessageID

					_, err = bot.Send(doc)
					if err != nil {
						log.Print(err)
					}
				case "say":
					text := ""
					if message.ReplyToMessage != nil && len(message.ReplyToMessage.Photo) > 0 {

						photo := message.ReplyToMessage.Photo
						if len(photo) == 0 {
							log.Printf("Error: No photo found in reply")
							break
						}
						text = writeImage(chatID, bot, photo)
					} else {
						// Get the text from the message being replied to
						text = message.ReplyToMessage.Text
					}
					if strings.TrimSpace(text) == "" {
						msg := tgbotapi.NewMessage(chatID, "Please reply to a text message or a photo with readable text.")
						msg.ReplyToMessageID = message.MessageID
						if _, err := bot.Send(msg); err != nil {
							log.Printf("Error sending empty text warning: %v", err)
						}
						break
					}

					// Call ReadItLoud to convert the text to speech (assuming it returns an Audio struct)
					Voice := translator.ReadItLoud(text)
					if Voice == "" {
						msg := tgbotapi.NewMessage(chatID, "Could not generate voice for this message.")
						msg.ReplyToMessageID = message.MessageID
						if _, err := bot.Send(msg); err != nil {
							log.Printf("Error sending voice generation warning: %v", err)
						}
						break
					}
					tgbotapi.NewMessage(chatID, Voice)

					// Example 1: Send a local file
					file, err := os.Open("output.mp3")
					if err != nil {
						log.Print(err)
						break
					}
					defer file.Close()

					doc := tgbotapi.NewVoice(chatID, tgbotapi.FileReader{
						Name:   "output.mp3",
						Reader: file,
					})
					doc.ReplyToMessageID = message.MessageID

					_, err = bot.Send(doc)
					if err != nil {
						log.Print(err)
					}
				case "saymale", "sayMale":
					text := ""
					if message.ReplyToMessage != nil && len(message.ReplyToMessage.Photo) > 0 {

						photo := message.ReplyToMessage.Photo
						if len(photo) == 0 {
							log.Printf("Error: No photo found in reply")
							break
						}
						text = writeImage(chatID, bot, photo)
					} else {
						// Get the text from the message being replied to
						text = message.ReplyToMessage.Text
					}

					// Call ReadItLoud to convert the text to speech (assuming it returns an Audio struct)
					Voice := translator.ReadItLoud(text)
					tgbotapi.NewMessage(chatID, Voice)

					// Example 1: Send a local file
					file, err := os.Open("output.mp3")
					if err != nil {
						log.Print(err)
					}
					defer file.Close()

					voice := tgbotapi.NewVoice(chatID, tgbotapi.FileReader{
						Name:   "output.mp3",
						Reader: file,
					})
					voice.ReplyToMessageID = message.MessageID

					_, err = bot.Send(voice)
					if err != nil {
						log.Print(err)
					}
				case "sayfemale", "sayFemale":
					text := ""
					if message.ReplyToMessage != nil && len(message.ReplyToMessage.Photo) > 0 {

						photo := message.ReplyToMessage.Photo
						if len(photo) == 0 {
							log.Printf("Error: No photo found in reply")
							break
						}
						text = writeImage(chatID, bot, photo)
					} else {
						text = message.ReplyToMessage.Text
					}

					// Call ReadItLoud to convert the text to speech (assuming it returns an Audio struct)
					Voice := translator.ReadItLoudFemale(text)
					tgbotapi.NewMessage(chatID, Voice)

					// Example 1: Send a local file
					file, err := os.Open("outputFemale.mp3")
					if err != nil {
						log.Print(err)
					}
					defer file.Close()

					voice := tgbotapi.NewVoice(chatID, tgbotapi.FileReader{
						Name:   "outputFemale.mp3",
						Reader: file,
					})
					voice.ReplyToMessageID = message.MessageID

					_, err = bot.Send(voice)
					if err != nil {
						log.Print(err)
					}
				case "ar":
					text := ""
					if message.ReplyToMessage != nil && len(message.ReplyToMessage.Photo) > 0 {
						photo := message.ReplyToMessage.Photo
						if len(photo) == 0 {
							log.Printf("Error: No photo found in reply")
							break
						}
						extractedText := writeImage(chatID, bot, photo)
						// Translate the extracted text to Arabic
						arabicTranslation := translator.TranslateToArabic(extractedText)

						response := fmt.Sprintf("\n\nArabic Translation: %s", arabicTranslation)

						msg := tgbotapi.NewMessage(chatID, response)
						msg.ReplyToMessageID = message.MessageID
						if err = sendMarkdownMessage(bot, msg); err != nil {
							log.Printf("Error sending translation: %v", err)
						}
					} else if message.ReplyToMessage != nil {
						// Translate to English
						text = message.ReplyToMessage.Text

						// Translate to Arabic
						arabicTranslation := translator.TranslateToArabic(text)

						response := fmt.Sprintf(" %s\n", arabicTranslation)

						msg := tgbotapi.NewMessage(chatID, response)
						msg.ReplyToMessageID = message.MessageID
						if err := sendMarkdownMessage(bot, msg); err != nil {
							log.Printf("Error sending translation: %v", err)
						}
					}
				case "en":
					text := ""
					if message.ReplyToMessage != nil && len(message.ReplyToMessage.Photo) > 0 {
						photo := message.ReplyToMessage.Photo
						if len(photo) == 0 {
							log.Printf("Error: No photo found in reply")
							break
						}
						extractedText := writeImage(chatID, bot, photo)
						// Translate the extracted text to Arabic
						arabicTranslation := translator.TranslateToEnglish(extractedText)

						response := fmt.Sprintf("\n\nArabic Translation: %s", arabicTranslation)

						msg := tgbotapi.NewMessage(chatID, response)
						msg.ReplyToMessageID = message.MessageID
						if err = sendMarkdownMessage(bot, msg); err != nil {
							log.Printf("Error sending translation: %v", err)
						}
					} else if message.ReplyToMessage != nil {
						text = message.ReplyToMessage.Text

						// Translate to English
						englishTranslation := translator.TranslateToEnglish(text)

						response := fmt.Sprintf("English: %s\n", englishTranslation)

						msg := tgbotapi.NewMessage(chatID, response)
						msg.ReplyToMessageID = message.MessageID
						if err := sendMarkdownMessage(bot, msg); err != nil {
							log.Printf("Error sending translation: %v", err)
						}
					}
				case "ru":
					if message.ReplyToMessage != nil && len(message.ReplyToMessage.Photo) > 0 {
						photo := message.ReplyToMessage.Photo
						if len(photo) == 0 {
							log.Printf("Error: No photo found in reply")
							break
						}
						extractedText := writeImage(chatID, bot, photo)
						// Translate the extracted text to Arabic
						arabicTranslation := translator.TranslateToRussian(extractedText)

						response := fmt.Sprintf("\n\n Russian Translation: %s", arabicTranslation)

						msg := tgbotapi.NewMessage(chatID, response)
						msg.ReplyToMessageID = message.MessageID
						if err = sendMarkdownMessage(bot, msg); err != nil {
							log.Printf("Error sending translation: %v", err)
						}
					} else if message.ReplyToMessage != nil {
						text = message.ReplyToMessage.Text

						// Translate to English
						russianTranslation := translator.TranslateToRussian(text)

						response := fmt.Sprintf("Russian: %s\n", russianTranslation)

						msg := tgbotapi.NewMessage(chatID, response)
						msg.ReplyToMessageID = message.MessageID
						if err := sendMarkdownMessage(bot, msg); err != nil {
							log.Printf("Error sending translation: %v", err)
						}
					}
				case "fr":
					if message.ReplyToMessage != nil && len(message.ReplyToMessage.Photo) > 0 {
						photo := message.ReplyToMessage.Photo
						if len(photo) == 0 {
							log.Printf("Error: No photo found in reply")
							break
						}
						extractedText := writeImage(chatID, bot, photo)

						// Translate the extracted text to Arabic
						arabicTranslation := translator.TranslateToFrench(extractedText)

						response := fmt.Sprintf("\n\nArabic Translation: %s", arabicTranslation)

						msg := tgbotapi.NewMessage(chatID, response)
						msg.ReplyToMessageID = message.MessageID
						if err = sendMarkdownMessage(bot, msg); err != nil {
							log.Printf("Error sending translation: %v", err)
						}
					} else if message.ReplyToMessage != nil {
						text = message.ReplyToMessage.Text

						// Translate to English
						frenchTranslation := translator.TranslateToFrench(text)

						response := fmt.Sprintf("French: %s\n", frenchTranslation)

						msg := tgbotapi.NewMessage(chatID, response)
						msg.ReplyToMessageID = message.MessageID
						if err := sendMarkdownMessage(bot, msg); err != nil {
							log.Printf("Error sending translation: %v", err)
						}
					}
				case "abb":
					if message.ReplyToMessage != nil {
						text = message.ReplyToMessage.Text

						// Get the abbreviation
						abbreviation := translator.GetAbbreviation(text)

						response := fmt.Sprintf("%s: \n:%s", text, abbreviation)

						msg := tgbotapi.NewMessage(chatID, response)
						msg.ReplyToMessageID = message.MessageID
						if err := sendMarkdownMessage(bot, msg); err != nil {
							log.Printf("Error sending translation: %v", err)
						}
					}
				case "translate":
					// Translate to English
					text = message.ReplyToMessage.Text
					englishTranslation := translator.TranslateToEnglish(text)

					// Translate to Arabic
					arabicTranslation := translator.TranslateToArabic(text)

					response := fmt.Sprintf("English: %s\n\nArabic: %s", englishTranslation, arabicTranslation)

					msg := tgbotapi.NewMessage(chatID, response)
					msg.ReplyToMessageID = message.MessageID
					if err := sendMarkdownMessage(bot, msg); err != nil {
						log.Printf("Error sending translation: %v", err)
					}
				case "syn":
					text = message.ReplyToMessage.Text

					// Translate to English
					synonyms := translator.GetSynonyms(text)

					response := fmt.Sprintf("Synonyms for the word %s \n:%s", text, synonyms)

					msg := tgbotapi.NewMessage(chatID, response)
					msg.ReplyToMessageID = message.MessageID
					if err := sendMarkdownMessage(bot, msg); err != nil {
						log.Printf("Error sending translation: %v", err)
					}
				case "anto":
					text = message.ReplyToMessage.Text

					// Translate to English
					antonyms := translator.GetAntonyms(text)

					response := fmt.Sprintf("Antonyms for the word %s \n:%s", text, antonyms)

					msg := tgbotapi.NewMessage(chatID, response)
					msg.ReplyToMessageID = message.MessageID
					if err := sendMarkdownMessage(bot, msg); err != nil {
						log.Printf("Error sending translation: %v", err)
					}
				case "define":
					text = message.ReplyToMessage.Text

					// Translate to English
					definition := translator.GetDefinition(text)

					response := fmt.Sprintf(" %s \n:%s", text, definition)

					msg := tgbotapi.NewMessage(chatID, response)
					msg.ReplyToMessageID = message.MessageID
					if err := sendMarkdownMessage(bot, msg); err != nil {
						log.Printf("Error sending translation: %v", err)
					}
				case "write":
					if message.ReplyToMessage != nil && len(message.ReplyToMessage.Photo) > 0 {
						photo := message.ReplyToMessage.Photo
						if len(photo) == 0 {
							log.Printf("Error: No photo found in reply")
							break
						}
						extractedText := writeImage(chatID, bot, photo)

						response := extractedText

						msg := tgbotapi.NewMessage(chatID, response)
						msg.ReplyToMessageID = message.MessageID
						if err := sendMarkdownMessage(bot, msg); err != nil {
							log.Printf("Error sending translation: %v", err)
						}
					}
				case "test":
					// Prepare image file for upload
					photo := message.ReplyToMessage.Photo
					fmt.Println("-------------------------------", photo)
					imagePath, err := utilities.SaveImage(bot, chatID, photo[len(photo)-1])
					if err != nil {
						log.Printf("Error saving image: %v", err)
						break
					}
					imgToTxt := utilities.ImageToText(imagePath, APINinjas)
					fmt.Println(imgToTxt)
					fmt.Println(imagePath)
					msg := tgbotapi.NewMessage(chatID, imgToTxt)
					msg.ReplyToMessageID = message.MessageID
					if err := sendMarkdownMessage(bot, msg); err != nil {
						log.Printf("Error sending translation: %v", err)
					}
				case cmd:
					voiceID, err := utilities.Configurator("config.json", cmd)
					if err != nil {
						break
					}
					text := ""
					if message.ReplyToMessage != nil && len(message.ReplyToMessage.Photo) > 0 {

						photo := message.ReplyToMessage.Photo
						if len(photo) == 0 {
							log.Printf("Error: No photo found in reply")
							break
						}
						text = writeImage(chatID, bot, photo)
					} else {
						// Get the text from the message being replied to
						text = message.ReplyToMessage.Text
					}

					// Call ReadItLoud to convert the text to speech (assuming it returns an Audio struct)
					Voice := translator.ElevenLabsDyna(text, voiceID)
					tgbotapi.NewMessage(chatID, Voice)

					// Example 1: Send a local file
					file, err := os.Open(voiceID + ".mp3")
					if err != nil {
						log.Print(err)
					}
					defer file.Close()

					doc := tgbotapi.NewVoice(chatID, tgbotapi.FileReader{
						Name:   voiceID + ".mp3",
						Reader: file,
					})
					doc.ReplyToMessageID = message.MessageID

					_, err = bot.Send(doc)
					if err != nil {
						log.Print(err)
					}

					// t:=message.ReplyToMessage.Photo
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
			switch cmd {
			case "start":
				startHandler(bot, update)
			}
		}
	}
}

func normalizeCommand(text, botUserName string) string {
	cmd := strings.Fields(strings.TrimSpace(text))
	if len(cmd) == 0 {
		return ""
	}

	normalized := strings.TrimPrefix(cmd[0], "/")
	if botUserName != "" {
		normalized = strings.TrimSuffix(normalized, "@"+botUserName)
	}

	lowerNormalized := strings.ToLower(normalized)
	if _, ok := builtInTranslatorCommands[lowerNormalized]; ok {
		return lowerNormalized
	}

	return normalized
}

var builtInTranslatorCommands = map[string]struct{}{
	"abb":         {},
	"anto":        {},
	"ar":          {},
	"define":      {},
	"en":          {},
	"fr":          {},
	"ru":          {},
	"say":         {},
	"sayai":       {},
	"sayaiuk":     {},
	"sayaifemale": {},
	"sayfemale":   {},
	"saymale":     {},
	"sayuk":       {},
	"sayukfemale": {},
	"start":       {},
	"syn":         {},
	"test":        {},
	"translate":   {},
	"write":       {},
}

func writeImage(chatID int64, bot *tgbotapi.BotAPI, photo []tgbotapi.PhotoSize) string {

	chatAction := tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping)
	bot.Send(chatAction)

	// Save the image and extract the image path
	imagePath, err := utilities.SaveImage(bot, chatID, photo[len(photo)-1])
	if err != nil {
		log.Printf("Error saving image: %v", err)
		return "Something went wrong :("
	}

	// Use WriteImage function to process the image and get the extracted text
	extractedText := translator.WriteImage("", imagePath)
	if extractedText == "" {
		log.Printf("No text found in image")
		return "No text found in image"
	}
	return extractedText
}
