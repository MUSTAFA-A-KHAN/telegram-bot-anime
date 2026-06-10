package collectible

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	model "github.com/MUSTAFA-A-KHAN/telegram-bot-anime/model/collectible"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
	collectibleRepo "github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository/collectible"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/service/collectible"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/view"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"go.mongodb.org/mongo-driver/mongo"
)

var pendingMarketplaceListing = make(map[int64]string) // map[userID]itemID
var pendingMarketplaceMutex sync.Mutex

func ShowHub(bot *tgbotapi.BotAPI, chatID int64, userID int, client *mongo.Client) {
	markup := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎁 Buy Pack (1000 🪙)", "collectible_buy_pack"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎒 My Collection", "collectible_inventory"),
			tgbotapi.NewInlineKeyboardButtonData("🏪 Marketplace", "collectible_market"),
		),
	)

	points := repository.GetCurrentPoints(client, userID)
	text := fmt.Sprintf("🌟 *Welcome to the Collectibles Hub!* 🌟\n\nOpen packs to discover unique anime-themed items. Trade them on the marketplace or build the ultimate collection!\n\n💰 *Your Balance:* %d 🪙", points)

	view.SendMessageWithButtons(bot, chatID, text, markup)
}

func HandleCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, client *mongo.Client) bool {
	data := callback.Data

	if data == "collectible_hub" {
		ShowHub(bot, callback.Message.Chat.ID, int(callback.From.ID), client)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
		return true
	} else if data == "collectible_buy_pack" {
		handleBuyPack(bot, callback, client)
		return true
	} else if data == "collectible_inventory" {
		showInventory(bot, callback, client)
		return true
	} else if data == "collectible_market" {
		showMarketplace(bot, callback, client)
		return true
	} else if strings.HasPrefix(data, "collectible_sell_") {
		itemID := strings.TrimPrefix(data, "collectible_sell_")
		promptSellPrice(bot, callback, itemID)
		return true
	} else if strings.HasPrefix(data, "collectible_buy_listing_") {
		listingID := strings.TrimPrefix(data, "collectible_buy_listing_")
		handleBuyListing(bot, callback, client, listingID)
		return true
	}

	return false
}

func handleBuyPack(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, client *mongo.Client) {
	item, template, err := collectible.OpenPack(client, int(callback.From.ID), callback.From.FirstName, callback.Message.Chat.ID)

	if err != nil {
		if err.Error() == "not enough points" {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, "Not enough points! Packs cost 1000 🪙."))
		} else {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, "Error opening pack: "+err.Error()))
		}
		return
	}

	text := fmt.Sprintf("✨ *Pack Opened!* ✨\n\nYou found:\n\n%s *%s #%d*\n⭐ Rarity: %s\n\nIt has been added to your collection!", template.Emoji, template.Name, item.SerialNumber, string(template.Rarity))

	markup := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎁 Buy Another", "collectible_buy_pack"),
			tgbotapi.NewInlineKeyboardButtonData("🎒 View Collection", "collectible_inventory"),
		),
	)

	if template.ImageURL != "" {
		photo := tgbotapi.NewPhotoShare(callback.Message.Chat.ID, template.ImageURL)
		photo.Caption = text
		photo.ParseMode = "Markdown"
		photo.ReplyMarkup = markup
		_, err := bot.Send(photo)
		if err != nil {
			log.Printf("send error: %v", err)
		}
	} else {
		view.SendMessageWithButtons(bot, callback.Message.Chat.ID, text, markup)
	}
	bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, "Pack opened!"))
}

func showInventory(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, client *mongo.Client) {
	items, templates, err := collectible.GetUserInventoryWithTemplates(client, int(callback.From.ID))
	if err != nil {
		bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, "Error loading inventory!"))
		return
	}

	// Pre-fetch listings to know which items are currently on the market
	listings, err := collectibleRepo.GetListings(client)
	var listingsErr bool
	if err != nil {
		listingsErr = true
	}
	listedItemsMap := make(map[string]bool)
	for _, l := range listings {
		listedItemsMap[l.ItemID] = true
	}

	if len(items) == 0 {
		markup := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Hub", "collectible_hub")),
		)
		editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, "🎒 *Your Collection is empty!*\n\nBuy some packs to get started.")
		editMsg.ReplyMarkup = &markup
		editMsg.ParseMode = "Markdown"
		bot.Send(editMsg)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	text := "🎒 *Your Collection*\n\n"
	var rows [][]tgbotapi.InlineKeyboardButton

	for _, item := range items {
		tmpl := templates[item.TemplateID]

		status := ""
		if !listingsErr && listedItemsMap[item.ID] {
			status = " [Listed on Market 🏪]"
		}

		text += fmt.Sprintf("%s *%s #%d* (%s)%s\n", tmpl.Emoji, tmpl.Name, item.SerialNumber, string(tmpl.Rarity), status)

		if !listedItemsMap[item.ID] {
			btnText := fmt.Sprintf("Sell: %s #%d", tmpl.Name, item.SerialNumber)
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(btnText, "collectible_sell_"+item.ID),
			))
		}
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Hub", "collectible_hub"),
	))

	markup := tgbotapi.NewInlineKeyboardMarkup(rows...)
	// editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	// editMsg.ReplyMarkup = &markup
	// editMsg.ParseMode = "Markdown"
	bot.Send(tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID))
	view.SendMessageWithButtons(bot, callback.Message.Chat.ID, text, markup)
	bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
}

func showMarketplace(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, client *mongo.Client) {
	listings, items, templates, err := collectible.GetMarketplaceListingsWithDetails(client)
	if err != nil {
		bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, "Error loading marketplace!"))
		return
	}

	if len(listings) == 0 {
		markup := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Hub", "collectible_hub")),
		)
		editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, "🏪 *Marketplace is currently empty!*")
		editMsg.ReplyMarkup = &markup
		editMsg.ParseMode = "Markdown"
		bot.Send(editMsg)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	text := "🏪 *Marketplace*\n\n"
	var rows [][]tgbotapi.InlineKeyboardButton

	for _, listing := range listings {
		item := items[listing.ItemID]
		tmpl := templates[item.TemplateID]

		text += fmt.Sprintf("%s *%s #%d* - %d 🪙\n", tmpl.Emoji, tmpl.Name, item.SerialNumber, listing.Price)

		btnText := fmt.Sprintf("Buy: %s #%d (%d 🪙)", tmpl.Name, item.SerialNumber, listing.Price)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(btnText, "collectible_buy_listing_"+listing.ID),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Hub", "collectible_hub"),
	))

	markup := tgbotapi.NewInlineKeyboardMarkup(rows...)
	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	editMsg.ReplyMarkup = &markup
	editMsg.ParseMode = "Markdown"
	bot.Send(editMsg)
	bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
}

func promptSellPrice(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, itemID string) {
	pendingMarketplaceMutex.Lock()
	pendingMarketplaceListing[int64(callback.From.ID)] = itemID
	pendingMarketplaceMutex.Unlock()

	text := "💰 Please reply with the price in Coins (numbers only) to list this item on the marketplace. Or type /cancel to abort."
	view.SendMessage(bot, callback.Message.Chat.ID, text)
	bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
}

func handleBuyListing(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, client *mongo.Client, listingID string) {
	err := collectible.BuyItemFromMarketplace(client, listingID, int(callback.From.ID), callback.From.FirstName, callback.Message.Chat.ID)

	if err != nil {
		bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, "Purchase failed: "+err.Error()))
		return
	}

	bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, "Successfully purchased item!"))
	showMarketplace(bot, callback, client)
}

// CheckAndHandlePendingMarketplaceListing checks if a user is trying to list an item and processes the text as a price
func CheckAndHandlePendingMarketplaceListing(bot *tgbotapi.BotAPI, message *tgbotapi.Message, client *mongo.Client) bool {
	userID := int64(message.From.ID)

	pendingMarketplaceMutex.Lock()
	itemID, ok := pendingMarketplaceListing[userID]
	if ok {
		// Remove from state immediately to avoid holding the global lock
		delete(pendingMarketplaceListing, userID)
	}
	pendingMarketplaceMutex.Unlock()

	if !ok {
		return false
	}

	text := strings.TrimSpace(message.Text)
	if text == "/cancel" {
		view.SendMessage(bot, message.Chat.ID, "Marketplace listing cancelled.")
		return true
	}

	price, err := strconv.Atoi(text)
	if err != nil || price <= 0 {
		view.SendMessage(bot, message.Chat.ID, "Invalid price. Listing cancelled. Please use numbers only greater than 0.")
		return true
	}

	// Verify ownership before listing
	item, err := collectibleRepo.GetItemByID(client, itemID)
	if err != nil || item.OwnerID != int(message.From.ID) {
		view.SendMessage(bot, message.Chat.ID, "Failed to list item: Item not found or you don't own it.")
		return true
	}

	err = collectibleRepo.CreateListing(client, model.MarketListing{
		ItemID:   itemID,
		SellerID: int(message.From.ID),
		Price:    price,
	})

	if err != nil {
		log.Printf("Error creating marketplace listing: %v", err)
		view.SendMessage(bot, message.Chat.ID, "An error occurred while creating your listing.")
		return true
	}

	view.SendMessage(bot, message.Chat.ID, fmt.Sprintf("✅ Successfully listed your item for %d 🪙!", price))
	return true
}
