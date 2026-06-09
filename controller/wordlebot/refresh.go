package wordlebot

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/controller/wordlebot/image_generator"
	"go.mongodb.org/mongo-driver/mongo"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/view"
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

	if isImage {
		imageData, err := image_generator.GenerateWordleImage(ws.Guesses, ws.Word, settings.WordleColor)
		if err == nil {
			err = view.EditMessageMediaWithStyledButtons(bot.Token, chatID, messageID, imageData, "wordle.png", &buttons)
			if err != nil {
				// Fallback if message wasn't a photo previously
				deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
				bot.Send(deleteMsg)

				photo := tgbotapi.NewPhotoUpload(chatID, tgbotapi.FileBytes{Name: "wordle.png", Bytes: imageData})
				photo.ReplyMarkup = buttons
				bot.Send(photo)
			}
			return
		}
		// fallback to text
	}

	// If not image, delete and resend text
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	bot.Send(deleteMsg)

	boardStr := buildWordleBoard(ws, settings.WordleColor)
	msgText := fmt.Sprintf("📝 *WORDLE*\n\n%s\n\nTotal: %d/6", boardStr, len(ws.Guesses))
	msg := tgbotapi.NewMessage(chatID, msgText)
	msg.ReplyMarkup = buttons
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)
}
