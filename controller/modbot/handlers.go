package modbot

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"go.mongodb.org/mongo-driver/mongo"
)

type AdminCacheEntry struct {
	Admins    map[int]bool
	ExpiresAt time.Time
}

var (
	adminCache = make(map[int64]AdminCacheEntry)
	adminMutex sync.RWMutex
)

// isAdmin checks if the user is an administrator in the group.
// It uses a temporary in-memory cache to prevent exhausting API rate limits.
func isAdmin(bot *tgbotapi.BotAPI, chatID int64, userID int) bool {
	if chatID > 0 { // Private chat
		return true
	}

	// Check cache first
	adminMutex.RLock()
	entry, exists := adminCache[chatID]
	adminMutex.RUnlock()

	if exists && time.Now().Before(entry.ExpiresAt) {
		return entry.Admins[userID]
	}

	// Cache miss or expired, fetch from API
	admins, err := bot.GetChatAdministrators(tgbotapi.ChatConfig{ChatID: chatID})
	if err != nil {
		log.Printf("Failed to get chat administrators for chat %d: %v", chatID, err)
		return false
	}

	newAdmins := make(map[int]bool)
	for _, admin := range admins {
		newAdmins[admin.User.ID] = true
	}

	// Cache for 5 minutes
	adminMutex.Lock()
	adminCache[chatID] = AdminCacheEntry{
		Admins:    newAdmins,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	adminMutex.Unlock()

	return newAdmins[userID]
}

func handleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message, dbClient interface{}) {
	client := dbClient.(*mongo.Client)

	if message.IsCommand() {
		handleCommand(bot, message, client)
		return
	}

	// Also handle commands in captions
	if message.Caption != "" {
		isCmd, cmd, args := ExtractCommandFromCaption(message)
		if isCmd {
			// We can slightly mutate the message so handleCommand works
			message.Text = "/" + cmd + " " + args
			handleCommand(bot, message, client)
			return
		}
	}

	// Handle regular messages for filtering and auto-responder
	handleFilters(bot, message, client)
}

func handleCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, client *mongo.Client) {
	chatID := message.Chat.ID

	// Anonymous messages are from admins/owners; skip command processing.
	if message.From == nil {
		return
	}

	userID := message.From.ID

	// Admins only
	if !isAdmin(bot, chatID, userID) {
		if message.Command() == "addrule" || message.Command() == "delrule" || message.Command() == "modsettings" {
			sendMessage(bot, chatID, "You must be an admin to use this command.")
		}
		return
	}

	settings := GetChatSettings(chatID)

	switch message.Command() {
	case "addrule":
		// Usage: /addrule <trigger_word> <response_text> OR /addrule <trigger_word> (as a reply to media)
		args := message.CommandArguments()
		parts := strings.SplitN(args, " ", 2)

		if len(parts) == 0 || parts[0] == "" {
			sendMessage(bot, chatID, "Usage:\n- `/addrule <word> <response>`\n- `/addrule <word>` (replying to a file/image)")
			return
		}

		trigger := strings.ToLower(strings.TrimSpace(parts[0]))

		// INTERACTIVE FLOW INIT
		if trigger == "" && message.ReplyToMessage == nil && message.Caption == "" {
			SetInteractiveState(chatID, userID, AddRuleState{Step: 1})

			msg := tgbotapi.NewMessage(chatID, "Please send the keyword for the new rule.")
			msg.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true, Selective: true}
			bot.Send(msg)
			return
		}

		rule := ModRuleDoc{TriggerWord: trigger}

		if message.ReplyToMessage != nil {
			// Check if it's a media reply
			if message.ReplyToMessage.Photo != nil && len(*message.ReplyToMessage.Photo) > 0 {
				photos := *message.ReplyToMessage.Photo
				rule.ResponseType = "photo"
				rule.ResponseFileID = photos[len(photos)-1].FileID
			} else if message.ReplyToMessage.Video != nil {
				rule.ResponseType = "video"
				rule.ResponseFileID = message.ReplyToMessage.Video.FileID

			} else if message.ReplyToMessage.Voice != nil {
				rule.ResponseType = "voice"
				rule.ResponseFileID = message.ReplyToMessage.Voice.FileID
			} else if message.ReplyToMessage.Document != nil {
				rule.ResponseType = "document"
				rule.ResponseFileID = message.ReplyToMessage.Document.FileID
			} else if message.ReplyToMessage.Animation != nil {
				rule.ResponseType = "animation"
				rule.ResponseFileID = message.ReplyToMessage.Animation.FileID
			} else if message.ReplyToMessage.Text != "" {
				rule.ResponseType = "text"
				rule.ResponseText = message.ReplyToMessage.Text
			} else {
				sendMessage(bot, chatID, "Unsupported media type for rule.")
				return
			}
		} else if message.Caption != "" {
			// Support direct caption rule addition without replying
			if message.Photo != nil && len(*message.Photo) > 0 {
				photos := *message.Photo
				rule.ResponseType = "photo"
				rule.ResponseFileID = photos[len(photos)-1].FileID
			} else if message.Video != nil {
				rule.ResponseType = "video"
				rule.ResponseFileID = message.Video.FileID
			} else if message.Voice != nil {
				rule.ResponseType = "voice"
				rule.ResponseFileID = message.Voice.FileID
			} else if message.Document != nil {
				rule.ResponseType = "document"
				rule.ResponseFileID = message.Document.FileID
			} else if message.Animation != nil {
				rule.ResponseType = "animation"
				rule.ResponseFileID = message.Animation.FileID
			} else {
				sendMessage(bot, chatID, "Unsupported media type for rule in caption.")
				return
			}
		} else if len(parts) > 1 {
			// Text rule
			rule.ResponseType = "text"
			rule.ResponseText = strings.TrimSpace(parts[1])
		} else {
			// They provided a trigger, but no response or reply
			SetInteractiveState(chatID, userID, AddRuleState{Step: 2, TriggerWord: trigger})

			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Great! Now send the text or media that should be sent when someone says `%s`.", trigger))
			msg.ParseMode = "Markdown"
			msg.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true, Selective: true}
			bot.Send(msg)
			return
		}

		settings.Rules[trigger] = rule
		SaveChatSettings(client, settings)
		sendMessage(bot, chatID, fmt.Sprintf("✅ Rule added for keyword: `%s`", trigger))

	case "delrule":
		args := strings.ToLower(strings.TrimSpace(message.CommandArguments()))
		if args == "" {
			sendMessage(bot, chatID, "Usage: `/delrule <word>`")
			return
		}

		if _, exists := settings.Rules[args]; exists {
			delete(settings.Rules, args)
			SaveChatSettings(client, settings)
			sendMessage(bot, chatID, fmt.Sprintf("✅ Rule removed for keyword: `%s`", args))
		} else {
			sendMessage(bot, chatID, fmt.Sprintf("Rule not found for: `%s`", args))
		}

	case "addscamword":
		args := strings.ToLower(strings.TrimSpace(message.CommandArguments()))
		if args == "" {
			sendMessage(bot, chatID, "Usage: `/addscamword <phrase>`")
			return
		}

		// Check if already exists
		for _, w := range settings.ScamKeywords {
			if w == args {
				sendMessage(bot, chatID, "Phrase is already in the scam filter.")
				return
			}
		}

		settings.ScamKeywords = append(settings.ScamKeywords, args)
		SaveChatSettings(client, settings)
		sendMessage(bot, chatID, fmt.Sprintf("✅ Added `%s` to scam filter.", args))

	case "delscamword":
		args := strings.ToLower(strings.TrimSpace(message.CommandArguments()))
		if args == "" {
			sendMessage(bot, chatID, "Usage: `/delscamword <phrase>`")
			return
		}

		found := false
		var newWords []string
		for _, w := range settings.ScamKeywords {
			if w == args {
				found = true
			} else {
				newWords = append(newWords, w)
			}
		}

		if found {
			settings.ScamKeywords = newWords
			SaveChatSettings(client, settings)
			sendMessage(bot, chatID, fmt.Sprintf("✅ Removed `%s` from scam filter.", args))
		} else {
			sendMessage(bot, chatID, "Phrase not found in the scam filter.")
		}

	case "adddomain":
		args := strings.ToLower(strings.TrimSpace(message.CommandArguments()))
		if args == "" {
			sendMessage(bot, chatID, "Usage: `/adddomain <domain>` (e.g. google.com)")
			return
		}

		for _, d := range settings.AllowedDomains {
			if d == args {
				sendMessage(bot, chatID, "Domain is already allowed.")
				return
			}
		}

		settings.AllowedDomains = append(settings.AllowedDomains, args)
		SaveChatSettings(client, settings)
		sendMessage(bot, chatID, fmt.Sprintf("✅ Allowed domain `%s`.", args))

	case "deldomain":
		args := strings.ToLower(strings.TrimSpace(message.CommandArguments()))
		if args == "" {
			sendMessage(bot, chatID, "Usage: `/deldomain <domain>`")
			return
		}

		found := false
		var newDomains []string
		for _, d := range settings.AllowedDomains {
			if d == args {
				found = true
			} else {
				newDomains = append(newDomains, d)
			}
		}

		if found {
			settings.AllowedDomains = newDomains
			SaveChatSettings(client, settings)
			sendMessage(bot, chatID, fmt.Sprintf("✅ Removed `%s` from allowed domains.", args))
		} else {
			sendMessage(bot, chatID, "Domain not found in allowed list.")
		}

	case "modsettings":
		sendSettingsMenu(bot, chatID, settings)
	}
}

func handleCallbackQuery(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, dbClient interface{}) {
	client := dbClient.(*mongo.Client)
	chatID := callback.Message.Chat.ID
	userID := callback.From.ID

	if !isAdmin(bot, chatID, userID) {
		bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, "You must be an admin to configure settings."))
		return
	}

	settings := GetChatSettings(chatID)

	switch callback.Data {
	case "toggle_block_links":
		settings.BlockLinks = !settings.BlockLinks
		SaveChatSettings(client, settings)
		updateSettingsMenu(bot, callback.Message, settings)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, "Settings updated."))

	case "toggle_scam_detection":
		settings.ScamDetection = !settings.ScamDetection
		SaveChatSettings(client, settings)
		updateSettingsMenu(bot, callback.Message, settings)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, "Settings updated."))

	case "menu_add_rule":
		// Enter the interactive flow but we already have the response
		SetInteractiveState(chatID, userID, AddRuleState{Step: 3}) // Step 3 means we are just waiting for the trigger word

		msg := tgbotapi.NewMessage(chatID, "What keyword should trigger this rule?")
		msg.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true, Selective: true}
		bot.Send(msg)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))

		// Delete the inline menu
		delMsg := tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID)
		bot.Send(delMsg)

	case "menu_add_scam":
		if pendingMsg, ok := GetAndClearPendingRuleMessage(chatID, userID); ok && pendingMsg.Text != "" {
			keyword := strings.ToLower(strings.TrimSpace(pendingMsg.Text))

			// Check if already exists
			exists := false
			for _, w := range settings.ScamKeywords {
				if w == keyword {
					exists = true
					break
				}
			}

			if !exists {
				settings.ScamKeywords = append(settings.ScamKeywords, keyword)
				SaveChatSettings(client, settings)
				sendMessage(bot, chatID, fmt.Sprintf("✅ Added `%s` to scam filter.", keyword))
			} else {
				sendMessage(bot, chatID, "Phrase is already in the scam filter.")
			}
		}
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
		delMsg := tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID)
		bot.Send(delMsg)

	case "menu_cancel":
		GetAndClearPendingRuleMessage(chatID, userID) // Clear the pending message
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, "Cancelled"))
		delMsg := tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID)
		bot.Send(delMsg)
	}
}

func sendSettingsMenu(bot *tgbotapi.BotAPI, chatID int64, settings *ModChatSettings) {
	msg := tgbotapi.NewMessage(chatID, "⚙️ *Moderator Bot Settings*")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = getSettingsKeyboard(settings)
	bot.Send(msg)
}

func updateSettingsMenu(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, settings *ModChatSettings) {
	editMsg := tgbotapi.NewEditMessageReplyMarkup(msg.Chat.ID, msg.MessageID, getSettingsKeyboard(settings))
	bot.Send(editMsg)
}

func getSettingsKeyboard(settings *ModChatSettings) tgbotapi.InlineKeyboardMarkup {
	linkText := "🔴 Block Links: OFF"
	if settings.BlockLinks {
		linkText = "🟢 Block Links: ON"
	}

	scamText := "🔴 Scam Detection: OFF"
	if settings.ScamDetection {
		scamText = "🟢 Scam Detection: ON"
	}

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(linkText, "toggle_block_links"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(scamText, "toggle_scam_detection"),
		),
	)
}

func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("Failed to send message: %v", err)
	}
}
