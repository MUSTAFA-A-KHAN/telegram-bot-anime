package modbot

import (
	"fmt"
	"log"
	"sort"
	"strconv"
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

	if message.Chat != nil && message.Chat.ID > 0 {
		trigger := strings.ToLower(strings.TrimSpace(message.Text))
		if rule, ok := GetGlobalRuleForTrigger(trigger); ok {
			sendRuleResponse(bot, message.Chat.ID, 0, rule)
			return
		}
	}

	// Handle regular messages for filtering and auto-responder
	handleFilters(bot, message, client)
}

func handleCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, client *mongo.Client) {
	chatID := message.Chat.ID

	// Messages from GroupAnonymousBot (anonymous admins) should not be processed.
	if message.From == nil || message.From.ID == groupAnonymousBotID {
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
	case "start", "menu":
		if chatID > 0 {
			sendGlobalRuleMenu(bot, chatID)
			return
		}
		// sendMessage(bot, chatID, "Use this command in a private chat with the bot to open the response menu.")
		return

	case "helpmod":
		sendModBotHelp(bot, chatID)
		return

	case "addrule":
		// Usage: /addrule <trigger_word> <response_text> OR /addrule <trigger_word> (as a reply to media)
		args := message.CommandArguments()
		parts := strings.SplitN(args, " ", 2)

		if len(parts) == 0 || parts[0] == "" {
			sendMessage(bot, chatID, "Usage:\n- `/addrule <word> <response>`\n- `/addrule <word>` (replying to a file/image)")
			return
		}

		trigger := normalizeRuleTrigger(parts[0])

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
		args := normalizeRuleTrigger(message.CommandArguments())
		if args == "" {
			sendMessage(bot, chatID, "Usage: `/delrule <word>`")
			return
		}

		if chatID > 0 && IsGlobalAdmin(userID) {
			removed := deleteRuleGlobally(client, args)
			if removed > 0 {
				sendMessage(bot, chatID, fmt.Sprintf("✅ Removed rule `%s` from all cached chats.", args))
			} else {
				sendMessage(bot, chatID, fmt.Sprintf("Rule not found for: `%s`", args))
			}
			return
		}

		ruleKey, exists := findRuleKeyByNormalizedTrigger(settings, args)
		if exists {
			delete(settings.Rules, ruleKey)
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

	case "purge", "delete":
		args := strings.TrimSpace(message.CommandArguments())
		if args == "" {
			sendMessage(bot, chatID, "Usage: `/purge <count>`\nDeletes the last <count> messages before this command.")
			return
		}

		count, err := strconv.Atoi(args)
		if err != nil || count <= 0 {
			sendMessage(bot, chatID, "Please provide a valid number greater than 0. Example: `/purge 5`")
			return
		}

		if count > 100 {
			count = 100
		}

		deletedCount := purgeLastMessages(bot, chatID, message.MessageID, count)
		sendMessage(bot, chatID, fmt.Sprintf("✅ Deleted %d message(s).", deletedCount))
		// Optionally remove the command message itself
		commandDelete := tgbotapi.NewDeleteMessage(chatID, message.MessageID)
		bot.DeleteMessage(commandDelete)

	// Global admin only commands
	// globalban - only group admins can execute, globalunban - only global admins
	case "globalban":
		if !isAdmin(bot, chatID, userID) {
			sendMessage(bot, chatID, "You must be an admin to use this command.")
			return
		}
		targetID, reason, err := parseTargetAndReason(message)
		if err != nil {
			sendMessage(bot, chatID, fmt.Sprintf("Usage: `/globalban <reason>` (reply to a user) or `/globalban <user_id> <reason>`"))
			return
		}
		if err := GlobalBanUser(client, targetID, reason, userID); err != nil {
			sendMessage(bot, chatID, fmt.Sprintf("Failed to globally ban user: %v", err))
			return
		}
		sendMessage(bot, chatID, fmt.Sprintf("🚫 User %d has been globally banned. Reason: %s", targetID, reason))

		// Kick from current chat
		kickConfig := tgbotapi.KickChatMemberConfig{
			ChatMemberConfig: tgbotapi.ChatMemberConfig{
				ChatID: chatID,
				UserID: targetID,
			},
		}
		bot.KickChatMember(kickConfig)

		// Try to kick from all other known chats
		for cid := range settingsCache {
			if cid == chatID {
				continue
			}
			otherKickConfig := tgbotapi.KickChatMemberConfig{
				ChatMemberConfig: tgbotapi.ChatMemberConfig{
					ChatID: cid,
					UserID: targetID,
				},
			}
			bot.KickChatMember(otherKickConfig) // Best-effort
		}

	case "globalunban":
		if !IsGlobalAdmin(userID) {
			sendMessage(bot, chatID, "❌ Only global admins can use this command.")
			return
		}
		targetID, _, err := parseTargetAndReason(message)
		if err != nil {
			sendMessage(bot, chatID, "Usage: `/globalunban` (reply to a user) or `/globalunban <user_id>`")
			return
		}
		if !IsGloballyBanned(targetID) {
			sendMessage(bot, chatID, fmt.Sprintf("User %d is not globally banned.", targetID))
			return
		}
		if err := GlobalUnbanUser(client, targetID); err != nil {
			sendMessage(bot, chatID, fmt.Sprintf("Failed to unban user: %v", err))
			return
		}
		sendMessage(bot, chatID, fmt.Sprintf("✅ User %d has been globally unbanned.", targetID))

	case "globalbannedlist":
		if !isAdmin(bot, chatID, userID) {
			sendMessage(bot, chatID, "You must be an admin to use this command.")
			return
		}
		bannedUsers := GetGloballyBannedUsers()
		if len(bannedUsers) == 0 {
			sendMessage(bot, chatID, "No globally banned users.")
			return
		}
		msg := "🚫 *Globally Banned Users:*\n"
		for userID, ban := range bannedUsers {
			msg += fmt.Sprintf("• `%d` - %s (by %d on %s)\n", userID, ban.Reason, ban.BannedBy, ban.BannedAt.Format("2006-01-02"))
		}
		sendMessage(bot, chatID, msg)

	case "addglobaladmin":
		if !IsGlobalAdmin(userID) {
			sendMessage(bot, chatID, "❌ Only global admins can add other global admins.")
			return
		}
		targetID, _, err := parseTargetAndReason(message)
		if err != nil {
			sendMessage(bot, chatID, "Usage: `/addglobaladmin` (reply to a user) or `/addglobaladmin <user_id>`")
			return
		}
		if err := AddGlobalAdmin(client, targetID, userID); err != nil {
			sendMessage(bot, chatID, fmt.Sprintf("Failed to add global admin: %v", err))
			return
		}
		sendMessage(bot, chatID, fmt.Sprintf("✅ User %d is now a global admin.", targetID))

	case "removeglobaladmin":
		if !IsGlobalAdmin(userID) {
			sendMessage(bot, chatID, "❌ Only global admins can remove global admins.")
			return
		}
		targetID, _, err := parseTargetAndReason(message)
		if err != nil {
			sendMessage(bot, chatID, "Usage: `/removeglobaladmin` (reply to a user) or `/removeglobaladmin <user_id>`")
			return
		}
		if !IsGlobalAdmin(targetID) {
			sendMessage(bot, chatID, fmt.Sprintf("User %d is not a global admin.", targetID))
			return
		}
		if err := RemoveGlobalAdmin(client, targetID); err != nil {
			sendMessage(bot, chatID, fmt.Sprintf("Failed to remove global admin: %v", err))
			return
		}
		sendMessage(bot, chatID, fmt.Sprintf("✅ User %d is no longer a global admin.", targetID))

	case "globaladminlist":
		if !isAdmin(bot, chatID, userID) {
			sendMessage(bot, chatID, "You must be an admin to use this command.")
			return
		}
		admins := GetGlobalAdmins()
		if len(admins) == 0 {
			sendMessage(bot, chatID, "No global admins configured.")
			return
		}
		msg := "👑 *Global Admins:*\n"
		for _, id := range admins {
			msg += fmt.Sprintf("• `%d`\n", id)
		}
		sendMessage(bot, chatID, msg)

	case "configglobalkeyboard":
		if !IsGlobalAdmin(userID) {
			sendMessage(bot, chatID, "❌ Only global admins can configure the global DM keyboard.")
			return
		}
		sendGlobalKeyboardConfigMenu(bot, chatID)

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

	switch {
	case callback.Data == "toggle_block_links":
		settings.BlockLinks = !settings.BlockLinks
		SaveChatSettings(client, settings)
		updateSettingsMenu(bot, callback.Message, settings)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, "Settings updated."))

	case callback.Data == "toggle_scam_detection":
		settings.ScamDetection = !settings.ScamDetection
		SaveChatSettings(client, settings)
		updateSettingsMenu(bot, callback.Message, settings)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, "Settings updated."))

	case strings.HasPrefix(callback.Data, "toggle_global_keyboard_item:"):
		if !IsGlobalAdmin(userID) {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, "Only global admins can change the global keyboard configuration."))
			return
		}
		indexText := strings.TrimPrefix(callback.Data, "toggle_global_keyboard_item:")
		index, err := strconv.Atoi(indexText)
		if err != nil || index < 0 {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, "Invalid keyboard item."))
			return
		}

		triggers := getGlobalRuleTriggers()
		if index >= len(triggers) {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, "Keyboard item no longer exists."))
			return
		}
		trigger := triggers[index]
		currentEnabled := isGlobalKeyboardItemEnabled(trigger)
		SetGlobalKeyboardMenuState(client, trigger, !currentEnabled)
		updateGlobalKeyboardConfigMenu(bot, callback.Message)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, "Keyboard menu updated."))

	case callback.Data == "menu_add_rule":
		// Enter the interactive flow but we already have the response
		SetInteractiveState(chatID, userID, AddRuleState{Step: 3}) // Step 3 means we are just waiting for the trigger word

		msg := tgbotapi.NewMessage(chatID, "What keyword should trigger this rule?")
		msg.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true, Selective: true}
		bot.Send(msg)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))

		// Delete the inline menu
		delMsg := tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID)
		bot.Send(delMsg)

	case callback.Data == "menu_add_scam":
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

	case callback.Data == "menu_cancel":
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

func sendGlobalRuleMenu(bot *tgbotapi.BotAPI, chatID int64) {
	triggers := getEnabledGlobalKeyboardTriggers(getGlobalRuleTriggers())
	if len(triggers) == 0 {
		sendMessage(bot, chatID, "No modbot rule responses are configured yet.")
		return
	}

	msg := tgbotapi.NewMessage(chatID, "📚 Choose a response from the keyboard:")
	msg.ReplyMarkup = buildRuleSelectionKeyboardFromTriggers(triggers)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Failed to send rule menu to chat %d: %v", chatID, err)
	}
}

func sendGlobalKeyboardConfigMenu(bot *tgbotapi.BotAPI, chatID int64) {
	triggers := getGlobalRuleTriggers()
	if len(triggers) == 0 {
		sendMessage(bot, chatID, "No modbot rule responses are configured yet.")
		return
	}

	msg := tgbotapi.NewMessage(chatID, "⚙️ *Global DM Keyboard Configuration*\nSelect which menu items are visible in the private keyboard.")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = getGlobalKeyboardConfigInlineKeyboard(triggers)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Failed to send global keyboard configuration menu to chat %d: %v", chatID, err)
	}
}

func deleteRuleGlobally(client *mongo.Client, trigger string) int {
	trigger = normalizeRuleTrigger(trigger)

	settingsMutex.RLock()
	chatIDs := make([]int64, 0, len(settingsCache))
	for chatID := range settingsCache {
		chatIDs = append(chatIDs, chatID)
	}
	settingsMutex.RUnlock()

	removed := 0
	for _, chatID := range chatIDs {
		settings := GetChatSettings(chatID)
		ruleKey, exists := findRuleKeyByNormalizedTrigger(settings, trigger)
		if !exists {
			continue
		}

		delete(settings.Rules, ruleKey)
		SaveChatSettings(client, settings)
		removed++
	}

	return removed
}

func GetGlobalRuleForTrigger(trigger string) (ModRuleDoc, bool) {
	settingsMutex.RLock()
	defer settingsMutex.RUnlock()

	normalizedTrigger := normalizeRuleTrigger(trigger)
	for _, settings := range settingsCache {
		if ruleKey, ok := findRuleKeyByNormalizedTrigger(settings, normalizedTrigger); ok {
			return settings.Rules[ruleKey], true
		}
	}
	return ModRuleDoc{}, false
}

func getGlobalRuleTriggers() []string {
	settingsMutex.RLock()
	defer settingsMutex.RUnlock()

	triggers := make([]string, 0)
	seen := make(map[string]bool)
	for _, settings := range settingsCache {
		for trigger := range settings.Rules {
			if seen[trigger] {
				continue
			}
			seen[trigger] = true
			triggers = append(triggers, trigger)
		}
	}
	sort.Strings(triggers)
	return triggers
}

func buildRuleSelectionKeyboardFromTriggers(triggers []string) tgbotapi.ReplyKeyboardMarkup {
	rows := make([][]tgbotapi.KeyboardButton, 0)
	row := make([]tgbotapi.KeyboardButton, 0)

	for _, trigger := range triggers {
		row = append(row, tgbotapi.NewKeyboardButton(trigger))
		if len(row) == 3 {
			rows = append(rows, row)
			row = make([]tgbotapi.KeyboardButton, 0)
		}
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}

	return tgbotapi.NewReplyKeyboard(rows...)
}

func buildRuleSelectionKeyboard(settings *ModChatSettings) tgbotapi.ReplyKeyboardMarkup {
	triggers := make([]string, 0, len(settings.Rules))
	for trigger := range settings.Rules {
		triggers = append(triggers, trigger)
	}
	sort.Strings(triggers)
	return buildRuleSelectionKeyboardFromTriggers(triggers)
}

func updateSettingsMenu(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, settings *ModChatSettings) {
	editMsg := tgbotapi.NewEditMessageReplyMarkup(msg.Chat.ID, msg.MessageID, getSettingsKeyboard(settings))
	bot.Send(editMsg)
}

func updateGlobalKeyboardConfigMenu(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	triggers := getGlobalRuleTriggers()
	editMsg := tgbotapi.NewEditMessageReplyMarkup(msg.Chat.ID, msg.MessageID, getGlobalKeyboardConfigInlineKeyboard(triggers))
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

func getGlobalKeyboardConfigInlineKeyboard(triggers []string) tgbotapi.InlineKeyboardMarkup {
	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(triggers))
	for i, trigger := range triggers {
		label := "✅ " + trigger
		if !isGlobalKeyboardItemEnabled(trigger) {
			label = "❌ " + trigger
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, fmt.Sprintf("toggle_global_keyboard_item:%d", i)),
		))
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("Failed to send message: %v", err)
	}
}

func purgeLastMessages(bot *tgbotapi.BotAPI, chatID int64, commandMessageID, count int) int {
	deleted := 0
	maxScan := count * 5 // or any reasonable limit

	for i := 1; deleted <= count && i <= maxScan; i++ {
		msgID := commandMessageID - i
		// if(isAdmin(bot,chatID, msgID)) {
		// }
		if msgID <= 0 {
			break
		}

		deleteMsg := tgbotapi.NewDeleteMessage(chatID, msgID)
		if _, err := bot.DeleteMessage(deleteMsg); err == nil {
			deleted++
		}
	}
	return deleted
}

func sendModBotHelp(bot *tgbotapi.BotAPI, chatID int64) {
	msgText := "*ModBot Commands*\n" +
		"`/addrule <word> <response>` - add keyword auto-response\n" +
		"`/addrule <word>` (reply to media) - add media response\n" +
		"`/delrule <word>` - remove a keyword response\n" +
		"`/addscamword <phrase>` - add a scam keyword\n" +
		"`/delscamword <phrase>` - remove a scam keyword\n" +
		"`/adddomain <domain>` - allow a domain\n" +
		"`/deldomain <domain>` - remove an allowed domain\n" +
		"`/purge <count>` - delete the last <count> messages before the command\n" +
		"`/globalban` / `/globalunban` / `/addglobaladmin` / `/removeglobaladmin` - global moderation commands\n"

	msg := tgbotapi.NewMessage(chatID, msgText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// parseTargetAndReason extracts the target user ID and reason from a command message.
// Target can be from replying to a message, or providing a user ID as first argument.
func parseTargetAndReason(message *tgbotapi.Message) (targetID int, reason string, err error) {
	args := strings.Fields(strings.TrimSpace(message.CommandArguments()))

	// If replying to a message, use that user's ID
	if message.ReplyToMessage != nil && message.ReplyToMessage.From != nil {
		targetID = message.ReplyToMessage.From.ID
		reason = strings.Join(args, " ")
		if reason == "" {
			reason = "Manual global ban by admin"
		}
		return targetID, reason, nil
	}

	// Otherwise, expect first arg is user ID
	if len(args) == 0 {
		return 0, "", fmt.Errorf("missing target")
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return 0, "", fmt.Errorf("invalid user ID: %v", err)
	}
	targetID = id

	if len(args) > 1 {
		reason = strings.Join(args[1:], " ")
	} else {
		reason = "Manual global ban by admin"
	}

	return targetID, reason, nil
}
