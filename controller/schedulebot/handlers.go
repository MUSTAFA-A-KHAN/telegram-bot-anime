package schedulebot

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/robfig/cron/v3"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type AdminCacheEntry struct {
	Admins    map[int]bool
	ExpiresAt time.Time
}

var (
	adminCache = make(map[int64]AdminCacheEntry)
	adminMutex sync.RWMutex

	// State for conversational additions: map[chatID]map[userID]cronString
	addingState = make(map[int64]map[int]string)
	stateMutex  sync.RWMutex

	// Track which job ID maps to which MongoDB ID
	cronJobs   = make(map[string]cron.EntryID)
	cronJobsMu sync.RWMutex
)

// isAdmin checks if the user is an administrator in the group.
func isAdmin(bot *tgbotapi.BotAPI, chatID int64, userID int) bool {
	if chatID > 0 { // Private chat
		return true
	}

	adminMutex.RLock()
	entry, exists := adminCache[chatID]
	adminMutex.RUnlock()

	if exists && time.Now().Before(entry.ExpiresAt) {
		return entry.Admins[userID]
	}

	admins, err := bot.GetChatAdministrators(tgbotapi.ChatConfig{ChatID: chatID})
	if err != nil {
		log.Printf("Failed to get chat administrators for chat %d: %v", chatID, err)
		return false
	}

	newAdmins := make(map[int]bool)
	for _, admin := range admins {
		newAdmins[admin.User.ID] = true
	}

	adminMutex.Lock()
	adminCache[chatID] = AdminCacheEntry{
		Admins:    newAdmins,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	adminMutex.Unlock()

	return newAdmins[userID]
}

func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

func loadSchedulesFromDB(client *mongo.Client) {
	schedules, err := getAllSchedules(client)
	if err != nil {
		log.Printf("Failed to load schedules: %v", err)
		return
	}

	for _, s := range schedules {
		scheduleJob(s)
	}
}

func scheduleJob(s ScheduledMessage) {
	idStr := s.ID.Hex()
	entryID, err := cronParser.AddFunc(s.CronExpression, func() {
		executeSchedule(s)
	})
	if err != nil {
		log.Printf("Failed to schedule job %s: %v", idStr, err)
		return
	}

	cronJobsMu.Lock()
	cronJobs[idStr] = entryID
	cronJobsMu.Unlock()
}

func executeSchedule(s ScheduledMessage) {
	if botInstance == nil {
		return
	}

	if s.MediaType == "" || s.MediaType == "text" {
		msg := tgbotapi.NewMessage(s.ChatID, s.Text)
		botInstance.Send(msg)
		return
	}

	// Handle media
	var sendMsg tgbotapi.Chattable
	switch s.MediaType {
	case "photo":
		msg := tgbotapi.NewPhotoShare(s.ChatID, s.FileID)
		msg.Caption = s.Caption
		sendMsg = msg
	case "video":
		msg := tgbotapi.NewVideoShare(s.ChatID, s.FileID)
		msg.Caption = s.Caption
		sendMsg = msg
	case "animation":
		msg := tgbotapi.NewAnimationShare(s.ChatID, s.FileID)
		msg.Caption = s.Caption
		sendMsg = msg
	case "audio":
		msg := tgbotapi.NewAudioShare(s.ChatID, s.FileID)
		msg.Caption = s.Caption
		sendMsg = msg
	case "document":
		msg := tgbotapi.NewDocumentShare(s.ChatID, s.FileID)
		msg.Caption = s.Caption
		sendMsg = msg
	case "voice":
		msg := tgbotapi.NewVoiceShare(s.ChatID, s.FileID)
		msg.Caption = s.Caption
		sendMsg = msg
	}

	if sendMsg != nil {
		botInstance.Send(sendMsg)
	}
}

func handleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message, client *mongo.Client) {
	if message == nil {
		return
	}

	chatID := message.Chat.ID
	userID := message.From.ID

	// Check if user is in adding state
	stateMutex.RLock()
	userState, chatHasState := addingState[chatID]
	var pendingCron string
	var hasPending bool
	if chatHasState {
		pendingCron, hasPending = userState[userID]
	}
	stateMutex.RUnlock()

	if hasPending && !message.IsCommand() {
		// They are sending the message to be scheduled
		var mediaType, fileID, text, caption string

		if message.Photo != nil && len(*message.Photo) > 0 {
			mediaType = "photo"
			photos := *message.Photo
			fileID = photos[len(photos)-1].FileID
			caption = message.Caption
		} else if message.Video != nil {
			mediaType = "video"
			fileID = message.Video.FileID
			caption = message.Caption
		} else if message.Animation != nil {
			mediaType = "animation"
			fileID = message.Animation.FileID
			caption = message.Caption
		} else if message.Audio != nil {
			mediaType = "audio"
			fileID = message.Audio.FileID
			caption = message.Caption
		} else if message.Document != nil {
			mediaType = "document"
			fileID = message.Document.FileID
			caption = message.Caption
		} else if message.Voice != nil {
			mediaType = "voice"
			fileID = message.Voice.FileID
			caption = message.Caption
		} else {
			mediaType = "text"
			text = message.Text
		}

		s := ScheduledMessage{
			ID:             primitive.NewObjectID(),
			ChatID:         chatID,
			AddedBy:        userID,
			CronExpression: pendingCron,
			MediaType:      mediaType,
			FileID:         fileID,
			Text:           text,
			Caption:        caption,
		}

		_, err := saveSchedule(client, s)
		if err != nil {
			sendMessage(bot, chatID, "❌ Failed to save scheduled message.")
		} else {
			scheduleJob(s)
			sendMessage(bot, chatID, "✅ Message scheduled successfully.")
		}

		// Clear state
		stateMutex.Lock()
		delete(addingState[chatID], userID)
		stateMutex.Unlock()
		return
	}

	if !message.IsCommand() {
		return
	}

	command := message.Command()

	// Only admins can manage schedules
	if command == "schedule" || command == "listschedules" || command == "cancelschedule" {
		if !isAdmin(bot, chatID, userID) {
			sendMessage(bot, chatID, "❌ You must be an admin to use this command.")
			return
		}
	}

	switch command {
	case "schedule":
		args := strings.TrimSpace(message.CommandArguments())
		if args == "" {
			sendMessage(bot, chatID, "Usage: `/schedule <cron_expression>`\nExample: `/schedule 0 10 * * *` (Every day at 10 AM)\n\nYou can also use format: `* * * * *` (minute, hour, day of month, month, day of week).")
			return
		}

		// Validate cron expression
		_, err := cron.ParseStandard(args)
		if err != nil {
			sendMessage(bot, chatID, "❌ Invalid cron expression. Please use standard cron format (e.g. `0 10 * * *`).")
			return
		}

		// Save state
		stateMutex.Lock()
		if addingState[chatID] == nil {
			addingState[chatID] = make(map[int]string)
		}
		addingState[chatID][userID] = args
		stateMutex.Unlock()

		msg := tgbotapi.NewMessage(chatID, "Cron expression accepted! Now send the message (text, photo, document, etc.) you want to schedule.")
		msg.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true, Selective: true}
		bot.Send(msg)

	case "listschedules":
		schedules, err := getSchedules(client, chatID)
		if err != nil {
			sendMessage(bot, chatID, "❌ Failed to fetch schedules.")
			return
		}

		if len(schedules) == 0 {
			sendMessage(bot, chatID, "No active scheduled messages in this chat.")
			return
		}

		var sb strings.Builder
		sb.WriteString("📅 *Scheduled Messages:*\n\n")
		for i, s := range schedules {
			sb.WriteString(fmt.Sprintf("%d. ID: `%s`\n   Cron: `%s`\n   Type: %s\n\n", i+1, s.ID.Hex(), s.CronExpression, s.MediaType))
		}
		sb.WriteString("Use `/cancelschedule <id>` to remove one.")
		sendMessage(bot, chatID, sb.String())

	case "cancelschedule":
		args := strings.TrimSpace(message.CommandArguments())
		if args == "" {
			sendMessage(bot, chatID, "Usage: `/cancelschedule <id>`")
			return
		}

		objID, err := primitive.ObjectIDFromHex(args)
		if err != nil {
			sendMessage(bot, chatID, "❌ Invalid schedule ID format.")
			return
		}

		res, err := deleteSchedule(client, chatID, objID)
		if err != nil {
			sendMessage(bot, chatID, "❌ Failed to delete schedule.")
			return
		}

		if res.DeletedCount > 0 {
			cronJobsMu.Lock()
			if entryID, exists := cronJobs[args]; exists {
				cronParser.Remove(entryID)
				delete(cronJobs, args)
			}
			cronJobsMu.Unlock()
			sendMessage(bot, chatID, "✅ Schedule deleted successfully.")
		} else {
			sendMessage(bot, chatID, "❌ Schedule not found.")
		}
	}
}
