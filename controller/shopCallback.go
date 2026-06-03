package controller

import (
	"fmt"
	"strings"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/service"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"go.mongodb.org/mongo-driver/mongo"
)

func handleShopCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, client *mongo.Client) bool {
	data := callback.Data

	if data == "inventory" {
		showInventory(bot, callback, client)
		return true
	} else if data == "shop_main" {
		editShopMain(bot, callback)
		return true
	} else if strings.HasPrefix(data, "buy_emoji_") {
		emoji := strings.TrimPrefix(data, "buy_emoji_")
		handleBuyEmoji(bot, callback, client, emoji)
		return true
	} else if strings.HasPrefix(data, "equip_emoji_") {
		emoji := strings.TrimPrefix(data, "equip_emoji_")
		handleEquipEmoji(bot, callback, client, emoji)
		return true
	}

	return false
}

func editShopMain(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
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

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, "🛒 *Welcome to the Emoji Shop!*\n\nSpend your Wordle Points here to buy custom emojis that will appear next to your name on leaderboards.")
	editMsg.ReplyMarkup = &markup
	editMsg.ParseMode = "Markdown"
	bot.Send(editMsg)

	bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
}

func showInventory(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, client *mongo.Client) {
	purchased, err := repository.GetPurchasedEmojis(client, int(callback.From.ID))
	if err != nil {
		bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, "Error loading inventory!"))
		return
	}

	if len(purchased) == 0 {
		bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, "Your inventory is empty! Buy some emojis first."))
		return
	}

	equipped, err := repository.GetEquippedEmojis(client, int(callback.From.ID))
	if err != nil {
		bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, "Error loading equipped emojis!"))
		return
	}

	equippedMap := make(map[string]bool)
	for _, e := range equipped {
		equippedMap[e] = true
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	var currentRow []tgbotapi.InlineKeyboardButton

	for i, emoji := range purchased {
		text := emoji
		if equippedMap[emoji] {
			text += " ✅"
		}
		btn := tgbotapi.NewInlineKeyboardButtonData(text, fmt.Sprintf("equip_emoji_%s", emoji))
		currentRow = append(currentRow, btn)

		if (i+1)%3 == 0 || i == len(purchased)-1 {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(currentRow...))
			currentRow = nil
		}
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Shop", "shop_main"),
	))

	markup := tgbotapi.NewInlineKeyboardMarkup(rows...)

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, "🎒 *Your Inventory*\n\nTap an emoji to toggle (equip/unequip) it. It will appear next to your name in leaderboards!")
	editMsg.ReplyMarkup = &markup
	editMsg.ParseMode = "Markdown"
	bot.Send(editMsg)

	bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
}

func handleBuyEmoji(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, client *mongo.Client, emoji string) {
	var price int
	for _, item := range service.ShopItems {
		if item.Emoji == emoji {
			price = item.Price
			break
		}
	}

	if price == 0 {
		bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, "Emoji not found in shop!"))
		return
	}

	purchased, _ := repository.GetPurchasedEmojis(client, int(callback.From.ID))
	for _, e := range purchased {
		if e == emoji {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, "You already own this emoji!"))
			return
		}
	}

	points := repository.GetCurrentPoints(client, int(callback.From.ID))
	if points < price {
		bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, fmt.Sprintf("Not enough points! You need %d points.", price)))
		return
	}

	repository.DeductWordlePoints(client, int(callback.From.ID), callback.From.FirstName, callback.Message.Chat.ID, price)

	err := repository.PurchaseEmoji(client, int(callback.From.ID), emoji)
	if err != nil {
		bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, "Error purchasing emoji! Please try again."))
		return
	}

	bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, fmt.Sprintf("Successfully bought %s for %d points!", emoji, price)))

	// Refresh shop view to potentially show new points or just state
	showShop(bot, callback.Message.Chat.ID)
}

func handleEquipEmoji(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, client *mongo.Client, emoji string) {
	isEquipped, err := repository.ToggleEquipEmoji(client, int(callback.From.ID), emoji)
	if err != nil {
		bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, "Error toggling emoji!"))
		return
	}

	status := "unequipped"
	if isEquipped {
		status = "equipped"
	}

	bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, fmt.Sprintf("%s has been %s!", emoji, status)))
	showInventory(bot, callback, client)
}
