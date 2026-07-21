package scramybot

import (
	"fmt"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/view"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"go.mongodb.org/mongo-driver/mongo"
)

// RefreshActiveGameMessage updates the active scramy game message
func RefreshActiveGameMessage(bot *tgbotapi.BotAPI, chatID int64, messageID int, client *mongo.Client) {
	ss := GetOrCreateScramyState(chatID)
	ss.RLock()
	defer ss.RUnlock()

	if !ss.Active {
		return
	}

	settings := GetChatSettings(chatID, client)
	isSquared := settings.ScramyLetterView == "squared"
	isH1 := settings.ScramyLetterView == "h1"

	buttons := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Change Layout ⚙️", "setting_scramy_letters_new"),
		),
	)

	if isH1 {
		// When layout is h1, editing rich text via bot.Send standard may not work cleanly if they switch from markdown text message to a rich text.
		// Standard `editMessageText` does not accept rich messages. We will delete the previous message and send a new rich message,
		// or if we must edit, Telegram Bot API typically requires `editMessageText` but OvyFlash API does not seem to easily support `editRichMessage`.
		// To be safe, we will delete the old message and resend.
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
		bot.Send(deleteMsg)

		topText := "📝 WORD SCRAMBLE\n\n🦴 Make words using these letters\n\n"
		bottomText := fmt.Sprintf("\n\n🔎 Words with 4 or more letters are accepted. Longer words give more points!\n\nTotal: %d/%d", len(ss.FoundWords), ss.MaxWords)
		letters := getLetterString(ss.Letters, false)
		view.SendScramyRichMessage(bot.Token, chatID, topText, letters, bottomText, buttons)
	} else {
		msgText := fmt.Sprintf("📝 *WORD SCRAMBLE*\n\n🦴 Make words using these letters\n\n%s\n\n🔎 Words with 4 or more letters are accepted. Longer words give more points!\n\nTotal: %d/%d", getLetterString(ss.Letters, isSquared), len(ss.FoundWords), ss.MaxWords)

		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, msgText)
		editMsg.ReplyMarkup = &buttons
		editMsg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(editMsg)
	}
}
