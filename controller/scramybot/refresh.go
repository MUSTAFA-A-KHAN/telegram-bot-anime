package scramybot

import (
	"fmt"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/view"
	tgbotapiv5Ovy "github.com/OvyFlash/telegram-bot-api"
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
		topText := "📝 WORD SCRAMBLE\n\n🦴 Make words using these letters\n\n"
		bottomText := fmt.Sprintf("\n\n🔎 Words with 4 or more letters are accepted. Longer words give more points!\n\nTotal: %d/%d", len(ss.FoundWords), ss.MaxWords)
		letters := getLetterString(ss.Letters, false)

		richMessage := tgbotapiv5Ovy.NewInputRichMessageBlocks(
			tgbotapiv5Ovy.InputRichBlockParagraph{
				Type: "paragraph",
				Text: topText,
			},
			tgbotapiv5Ovy.InputRichBlockSectionHeading{
				Type: "heading",
				Text: letters,
				Size: 1, // H1
			},
			tgbotapiv5Ovy.InputRichBlockParagraph{
				Type: "paragraph",
				Text: bottomText,
			},
		)

		ovyKeyboard := view.ConvertToOvyKeyboard(buttons)
		view.EditRichMessage(bot.Token, chatID, messageID, richMessage, ovyKeyboard)
	} else {
		msgText := fmt.Sprintf("📝 *WORD SCRAMBLE*\n\n🦴 Make words using these letters\n\n%s\n\n🔎 Words with 4 or more letters are accepted. Longer words give more points!\n\nTotal: %d/%d", getLetterString(ss.Letters, isSquared), len(ss.FoundWords), ss.MaxWords)

		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, msgText)
		editMsg.ReplyMarkup = &buttons
		editMsg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(editMsg)
	}
}
