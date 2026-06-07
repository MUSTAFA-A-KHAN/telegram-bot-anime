package scramybot

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"go.mongodb.org/mongo-driver/mongo"
)

func RefreshActiveGameMessage(bot *tgbotapi.BotAPI, chatID int64, messageID int, client *mongo.Client) {
	ss := GetOrCreateScramyState(chatID)
	ss.RLock()
	defer ss.RUnlock()

	if !ss.Active {
		return
	}

	settings := GetChatSettings(chatID, client)
	isSquared := settings.ScramyLetterView == "squared"

	buttons := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Change Layout ⚙️", "setting_scramy_letters_new"),
		),
	)

	msg := fmt.Sprintf("📝 *WORD SCRAMBLE*\n\n🦴 Make words using these letters\n\n%s\n\n🔎 Words with 4 or more letters are accepted. Longer words give more points!\n\nTotal: %d/10", getLetterString(ss.Letters, isSquared), len(ss.FoundWords))

	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, msg)
	editMsg.ReplyMarkup = &buttons
	editMsg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(editMsg)
}
