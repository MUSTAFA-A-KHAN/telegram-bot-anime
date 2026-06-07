package wordlebot

import (
	"fmt"
	"log"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"go.mongodb.org/mongo-driver/mongo"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/controller/wordlebot/image_generator"
)

func RefreshActiveGameMessage(bot *tgbotapi.BotAPI, chatID int64, messageID int, client *mongo.Client) {
	ws := GetOrCreateWordleState(chatID)
	ws.RLock()
	defer ws.RUnlock()

	if !ws.Active {
		return
	}

	buttons := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Change Layout ⚙️", "setting_wordle_view_new"),
		),
	)

	settings := GetChatSettings(chatID, client)
	isImage := settings.WordleViewType == "image"

	if isImage && len(ws.Guesses) > 0 {
		imgData, err := image_generator.GenerateWordleImage(ws.Guesses, ws.Word)
		if err == nil {
			deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
			bot.Send(deleteMsg)

			photo := tgbotapi.NewPhotoUpload(chatID, tgbotapi.FileBytes{Name: "wordle.png", Bytes: imgData})
			photo.ReplyMarkup = buttons
			bot.Send(photo)
			return
		}
		log.Printf("Failed to generate wordle image: %v", err)
	}

	var board string
	if len(ws.Guesses) > 0 {
		board = buildWordleBoard(ws)
	} else {
		board = fmt.Sprintf("🐊 🖼 *Wordle started!* ✨\n\n• The word consists of 5 letters.\n• You have %d attempts.\n\n💡 Hints:\n🟩 Correct letter in the right spot\n🟨 Correct letter but in the wrong spot\n🟥 Letter is not in the word\n\nSend a 5-letter word to guess.", ws.MaxAttempts)
	}

	// Try to edit the text message. If it was an image message before, Telegram API will fail editMessageText.
	// In that case, we can try to send a new message and delete the old one.
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, board)
	editMsg.ReplyMarkup = &buttons
	editMsg.ParseMode = tgbotapi.ModeMarkdown

	_, err := bot.Send(editMsg)
	if err != nil {
		// If it fails (e.g. changing from image to text), delete and resend
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
		bot.Send(deleteMsg)

		msg := tgbotapi.NewMessage(chatID, board)
		msg.ReplyMarkup = buttons
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
	}
}
