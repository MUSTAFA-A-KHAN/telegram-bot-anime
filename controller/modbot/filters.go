package modbot

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"go.mongodb.org/mongo-driver/mongo"
)

var urlRegex = regexp.MustCompile(`(?i)(https?:\/\/[^\s]+)|(www\.[^\s]+)|([a-zA-Z0-9-]+\.[a-zA-Z]{2,}(\/[^\s]*)?)`)

// groupAnonymousBotID is the Telegram user ID for GroupAnonymousBot,
// used when an admin sends a message anonymously as the group.
const groupAnonymousBotID = 1087968824

func handleFilters(bot *tgbotapi.BotAPI, message *tgbotapi.Message, client *mongo.Client) {
	chatID := message.Chat.ID

	// Messages sent via GroupAnonymousBot (anonymous admins) should not be filtered.
	if message.From == nil || message.From.ID == groupAnonymousBotID {
		return
	}

	userID := message.From.ID

	// Globally banned users: delete message and re-kick silently
	if IsGloballyBanned(userID) {
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, message.MessageID)
		bot.DeleteMessage(deleteMsg)
		// Re-kick the user from this chat as a safeguard
		bot.KickChatMember(tgbotapi.KickChatMemberConfig{
			ChatMemberConfig: tgbotapi.ChatMemberConfig{
				ChatID: chatID,
				UserID: userID,
			},
		})
		return
	}

	settings := GetChatSettings(chatID)

	text := strings.ToLower(strings.TrimSpace(message.Text))
	if text == "" && message.Caption != "" {
		text = strings.ToLower(strings.TrimSpace(message.Caption))
	}

	// Normalize text for scam/filter detection.
	// This handles Arabic letter obfuscation techniques such as:
	// - Tatweel/kashida characters (ـ) between letters: عـمـل -> عمل
	// - Different alef forms: أ, إ, آ -> ا
	// - Taa marbouta: ة -> ه
	// - Arabic Presentation Forms (initial/medial/final)
	// - Arabic diacritics (tashkeel)
	// - Zero-width characters
	// The original text is preserved for exact rule matching below.
	normalizedText := ArabicNormalize(text)

	// BOT MENTION OR REPLY MENU (Admins only)
	if isAdmin(bot, chatID, userID) && message.ReplyToMessage != nil {
		isBotMentioned := false
		botUsername := bot.Self.UserName

		// Check if the reply text mentions the bot
		if strings.Contains(text, "@"+strings.ToLower(botUsername)) {
			isBotMentioned = true
		}

		if isBotMentioned {
			SetPendingRuleMessage(chatID, userID, message.ReplyToMessage)

			// Show inline menu for this message
			msg := tgbotapi.NewMessage(chatID, "What would you like to do with the replied message?")
			msg.ReplyToMessageID = message.MessageID

			var keyboard [][]tgbotapi.InlineKeyboardButton

			// Always allow adding as a rule response
			keyboard = append(keyboard, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📝 Add as Rule Response", "menu_add_rule"),
			))

			// If it's text, we can also add it as a scam keyword
			if message.ReplyToMessage.Text != "" {
				keyboard = append(keyboard, tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("🚫 Add as Scam Keyword", "menu_add_scam"),
				))
			}

			keyboard = append(keyboard, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", "menu_cancel"),
			))

			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
			bot.Send(msg)
			return
		}
	}

	// INTERACTIVE FLOW HANDLING
	if state, exists := GetInteractiveState(chatID, userID); exists {
		handleInteractiveState(bot, message, client, settings, state)
		return
	}

	// 1. Auto-responder Rule Match (Exact Match)
	normalizedTrigger := normalizeRuleTrigger(text)
	if ruleKey, exists := findRuleKeyByNormalizedTrigger(settings, normalizedTrigger); exists {
		sendRuleResponse(bot, chatID, message.MessageID, settings.Rules[ruleKey])
		// We don't return here just in case the message also contained a bad link
	}

	// Don't filter group admins
	if isAdmin(bot, chatID, message.From.ID) {
		return
	}

	// 2. Scam Detection (using normalized text to catch obfuscated Arabic)
	if settings.ScamDetection {
		for _, keyword := range settings.ScamKeywords {
			if strings.Contains(normalizedText, keyword) {
				log.Printf("[ModBot] Scam detected in chat %d (user %d): original=%q, normalized=%q, keyword=%q",
					chatID, userID, text, normalizedText, keyword)
				handleViolation(bot, message, client, "Scam detection triggered")
				return
			}
		}
	}

	// 3. Link Blocking
	if settings.BlockLinks {
		urls := extractURLs(text)
		if message.Entities != nil {
			for _, entity := range *message.Entities {
				if entity.Type == "url" || entity.Type == "text_link" {
					if entity.URL != "" {
						urls = append(urls, extractURLs(entity.URL)...)
					}
				}
			}
		}

		if len(urls) > 0 {
			for _, foundURL := range urls {
				if !isAllowedDomain(foundURL, settings.AllowedDomains) {
					handleViolation(bot, message, client, "Unauthorized link detected")
					return // Message is deleted
				}
			}
		}
	}
}

func sendRuleResponse(bot *tgbotapi.BotAPI, chatID int64, replyToMessageID int, rule ModRuleDoc) {
	var msg tgbotapi.Chattable

	switch rule.ResponseType {
	case "text":
		textMsg := tgbotapi.NewMessage(chatID, rule.ResponseText)
		textMsg.ReplyToMessageID = replyToMessageID
		msg = textMsg
	case "photo":
		photoMsg := tgbotapi.NewPhotoShare(chatID, rule.ResponseFileID)
		photoMsg.ReplyToMessageID = replyToMessageID
		msg = photoMsg
	case "video":
		videoMsg := tgbotapi.NewVideoShare(chatID, rule.ResponseFileID)
		videoMsg.ReplyToMessageID = replyToMessageID
		msg = videoMsg
	case "voice":
		voiceMsg := tgbotapi.NewVoiceShare(chatID, rule.ResponseFileID)
		voiceMsg.ReplyToMessageID = replyToMessageID
		msg = voiceMsg
	case "document":
		docMsg := tgbotapi.NewDocumentShare(chatID, rule.ResponseFileID)
		docMsg.ReplyToMessageID = replyToMessageID
		msg = docMsg
	case "animation":
		animMsg := tgbotapi.NewAnimationShare(chatID, rule.ResponseFileID)
		animMsg.ReplyToMessageID = replyToMessageID
		msg = animMsg
	default:
		log.Printf("Unknown rule response type: %s", rule.ResponseType)
		return
	}

	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("Failed to send rule response: %v", err)
	}
}

func handleViolation(bot *tgbotapi.BotAPI, message *tgbotapi.Message, client *mongo.Client, reason string) {
	chatID := message.Chat.ID

	// Messages from GroupAnonymousBot (anonymous admins) should not be filtered.
	if message.From == nil || message.From.ID == groupAnonymousBotID {
		return
	}

	userID := message.From.ID

	// Delete message
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, message.MessageID)
	_, err := bot.DeleteMessage(deleteMsg)
	if err != nil {
		log.Printf("Failed to delete message: %v", err)
	}

	// Increment violations
	count := IncrementUserViolations(client, chatID, userID)

	// Report to group chat
	admins, _ := bot.GetChatAdministrators(tgbotapi.ChatConfig{ChatID: chatID})
	var adminMentions []string
	var adminIDs []int
	for _, admin := range admins {
		if !admin.User.IsBot {
			if admin.User.UserName != "" {
				adminMentions = append(adminMentions, fmt.Sprintf("@%s", admin.User.UserName))
			} else {
				adminMentions = append(adminMentions, admin.User.FirstName)
			}
			adminIDs = append(adminIDs, admin.User.ID)
		}
	}

	username := message.From.UserName
	if username == "" {
		username = message.From.FirstName
	} else {
		username = "@" + username
	}

	for i, adminMention := range adminMentions {
		if adminMention == "@" {
			// fallback if admin has no username
			admins, _ := bot.GetChatAdministrators(tgbotapi.ChatConfig{ChatID: chatID})
			for _, admin := range admins {
				if admin.User.ID == adminIDs[i] {
					adminMentions[i] = admin.User.FirstName
					break
				}
			}
		}
	}

	// reportMsg := fmt.Sprintf("⚠️ Deleted message from %s for %s. (Violation %d/3)\nCc: %s",
	// username, reason, count, strings.Join(adminMentions, " "))
	// sendMessage(bot, chatID, reportMsg)

	// Direct message to admins
	dmReport := fmt.Sprintf("🚨 *Moderator Alert in Chat %d*\nUser: %s (ID: %d)\nReason: %s\nViolations: %d",
		chatID, username, userID, reason, count)

	for _, adminID := range adminIDs {
		// Attempt to DM the admin. Note: this will fail if the admin hasn't started the bot directly.
		dm := tgbotapi.NewMessage(int64(adminID), dmReport)
		dm.ParseMode = "Markdown"
		bot.Send(dm) // ignore errors for DMs
	}

	// Apply global ban on 3rd violation
	if count >= 3 {
		err := GlobalBanUser(client, userID, reason, 0)
		if err != nil {
			log.Printf("Failed to globally ban user %d: %v", userID, err)
			sendMessage(bot, chatID, fmt.Sprintf("Failed to ban %s globally. Error: %v", username, err))
			return
		}

		// Kick user from the current chat
		kickConfig := tgbotapi.KickChatMemberConfig{
			ChatMemberConfig: tgbotapi.ChatMemberConfig{
				ChatID: chatID,
				UserID: userID,
			},
		}
		_, kickErr := bot.KickChatMember(kickConfig)
		if kickErr != nil {
			log.Printf("Failed to kick user %d from chat %d: %v", userID, chatID, kickErr)
		}

		// Try to kick from all other known chats
		settingsMutex.RLock()
		for cid := range settingsCache {
			if cid == chatID {
				continue
			}
			otherKickConfig := tgbotapi.KickChatMemberConfig{
				ChatMemberConfig: tgbotapi.ChatMemberConfig{
					ChatID: cid,
					UserID: userID,
				},
			}
			bot.KickChatMember(otherKickConfig) // Best-effort
		}
		settingsMutex.RUnlock()

		sendMessage(bot, chatID, fmt.Sprintf("🚫 %s has been **globally banned** for repeated violations.", username))

		// Update the DM report to reflect the global ban
		dmReport = fmt.Sprintf("🚨 *GLOBAL BAN*\nUser: %s (ID: %d)\nReason: %s\nViolations: %d\nBanned from all chats.",
			username, userID, reason, count)
	}
}

func extractURLs(text string) []string {
	if text == "" {
		return nil
	}
	matches := urlRegex.FindAllString(text, -1)
	return matches
}

func isAllowedDomain(link string, allowedDomains []string) bool {
	// Add scheme if missing so url.Parse works reliably
	if !strings.HasPrefix(link, "http://") && !strings.HasPrefix(link, "https://") {
		link = "https://" + link
	}

	parsed, err := url.Parse(link)
	if err != nil {
		return false
	}

	host := strings.ToLower(parsed.Host)
	// Strip www. prefix for cleaner matching
	host = strings.TrimPrefix(host, "www.")

	for _, domain := range allowedDomains {
		// If the allowed entry is a full URL, match it as a prefix.
		if strings.HasPrefix(strings.TrimSuffix(strings.ToLower(link), "/"), strings.TrimSuffix(strings.ToLower(domain), "/")) {
			return true
		}
		domain = strings.ToLower(strings.TrimPrefix(domain, "www."))
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return true
		}
	}

	return false
}

func handleInteractiveState(bot *tgbotapi.BotAPI, message *tgbotapi.Message, client *mongo.Client, settings *ModChatSettings, state AddRuleState) {
	chatID := message.Chat.ID
	userID := message.From.ID

	if message.Text == "/cancel" {
		ClearInteractiveState(chatID, userID)
		sendMessage(bot, chatID, "Rule creation cancelled.")
		return
	}

	if state.Step == 1 {
		// Expecting trigger word
		if message.Text == "" {
			sendMessage(bot, chatID, "Please send a text keyword to trigger the rule, or type /cancel to abort.")
			return
		}

		trigger := normalizeRuleTrigger(message.Text)
		SetInteractiveState(chatID, userID, AddRuleState{Step: 2, TriggerWord: trigger})

		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Great! Now send the text or media that should be sent when someone says `%s`.", trigger))
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true, Selective: true}
		bot.Send(msg)
		return
	}

	if state.Step == 2 || state.Step == 3 {

		var msgToProcess *tgbotapi.Message

		if state.Step == 2 {
			msgToProcess = message
		} else { // Step 3
			// We already have the response from the pending rule message
			// And the current message contains the trigger word
			if message.Text == "" {
				sendMessage(bot, chatID, "Please send a text keyword to trigger the rule, or type /cancel to abort.")
				return
			}
			state.TriggerWord = normalizeRuleTrigger(message.Text)

			pendingMsg, exists := GetAndClearPendingRuleMessage(chatID, userID)
			if !exists || pendingMsg == nil {
				sendMessage(bot, chatID, "Error: The pending message was lost. Please try again.")
				ClearInteractiveState(chatID, userID)
				return
			}
			msgToProcess = pendingMsg
		}

		// Expecting response
		rule := ModRuleDoc{TriggerWord: state.TriggerWord}

		if msgToProcess.Photo != nil && len(*msgToProcess.Photo) > 0 {
			photos := *msgToProcess.Photo
			rule.ResponseType = "photo"
			rule.ResponseFileID = photos[len(photos)-1].FileID
		} else if msgToProcess.Video != nil {
			rule.ResponseType = "video"
			rule.ResponseFileID = msgToProcess.Video.FileID
		} else if msgToProcess.Voice != nil {
			rule.ResponseType = "voice"
			rule.ResponseFileID = msgToProcess.Voice.FileID
		} else if msgToProcess.Document != nil {
			rule.ResponseType = "document"
			rule.ResponseFileID = msgToProcess.Document.FileID
		} else if msgToProcess.Animation != nil {
			rule.ResponseType = "animation"
			rule.ResponseFileID = msgToProcess.Animation.FileID
		} else if msgToProcess.Text != "" {
			rule.ResponseType = "text"
			rule.ResponseText = msgToProcess.Text
		} else {
			sendMessage(bot, chatID, "Unsupported media type. Please send text, photo, video, document, voice, or animation.")
			return
		}

		settings.Rules[state.TriggerWord] = rule
		SaveChatSettings(client, settings)
		ClearInteractiveState(chatID, userID)
		sendMessage(bot, chatID, fmt.Sprintf("✅ Rule added for keyword: `%s`", state.TriggerWord))
	}
}
