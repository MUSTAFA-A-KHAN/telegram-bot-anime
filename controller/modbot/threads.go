package modbot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// messageThreadID returns the forum topic (message thread) that the given
// message belongs to. It returns 0 for private chats, regular groups and
// messages that do not belong to a specific topic (e.g. the General topic).
//
// In forum supergroups (topic-enabled groups) every message carries a
// message_thread_id. The bot must echo it back on every outgoing message,
// otherwise Telegram posts the response into the General topic (or fails
// with "message thread not found").
func messageThreadID(m *tgbotapi.Message) int {
	if m == nil || m.Chat == nil {
		return 0
	}
	return m.MessageThreadID
}
