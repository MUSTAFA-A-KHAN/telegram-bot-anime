package categorybot

import (
	"fmt"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/service"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/view"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"go.mongodb.org/mongo-driver/mongo"
)

func showShop(bot *tgbotapi.BotAPI, chatID int64, userID int, client *mongo.Client) {
	var rows [][]tgbotapi.InlineKeyboardButton

	// Add inventory button
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🎒 My Inventory", "inventory"),
	))

	// Group emojis into rows of 3
	var currentRow []tgbotapi.InlineKeyboardButton
	for i, item := range service.ShopItems {
		btn := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%s - %d 🪙", item.Emoji, item.Price),
			fmt.Sprintf("buy_emoji_%s", item.Emoji),
		)
		currentRow = append(currentRow, btn)

		if (i+1)%3 == 0 || i == len(service.ShopItems)-1 {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(currentRow...))
			currentRow = nil
		}
	}

	markup := tgbotapi.NewInlineKeyboardMarkup(rows...)

	points := repository.GetCurrentPoints(client, userID)
	text := fmt.Sprintf("🛒 *Welcome to the Emoji Shop!*\n\nSpend your Wordle Points here to buy custom emojis that will appear next to your name on leaderboards.\n\n💰 *Your Balance:* %d 🪙", points)

	view.SendMessageWithButtons(bot, chatID, text, markup)
}
