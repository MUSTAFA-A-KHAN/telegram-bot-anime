package wordlebot

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/controller/wordlebot/image_generator"
	"go.mongodb.org/mongo-driver/mongo"
)

// RefreshActiveGameMessage updates the active wordle game message
func RefreshActiveGameMessage(bot *tgbotapi.BotAPI, chatID int64, messageID int, client *mongo.Client) {
	ws := GetOrCreateWordleState(chatID)
	ws.RLock()
	defer ws.RUnlock()

	if !ws.Active {
		return
	}

	settings := GetChatSettings(chatID, client)
	isImage := settings.WordleViewType == "image"

	buttons := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Change Layout ⚙️", "setting_wordle_view_new"),
		),
	)

	// Since we can't reliably edit text into photo, we will delete and resend.
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	bot.Send(deleteMsg)

	if isImage {
		imageData, err := image_generator.GenerateWordleImage(ws.Guesses, ws.Word)
		if err == nil {
			photo := tgbotapi.NewPhotoUpload(chatID, tgbotapi.FileBytes{Name: "wordle.png", Bytes: imageData})
			photo.ReplyMarkup = buttons
			bot.Send(photo)
			return
		}
		// fallback to text
	}

	boardStr := buildWordleBoard(ws)
	msgText := fmt.Sprintf("📝 *WORDLE*\n\n%s\n\nTotal: %d/6", boardStr, len(ws.Guesses))
	msg := tgbotapi.NewMessage(chatID, msgText)
	msg.ReplyMarkup = buttons
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)
}
