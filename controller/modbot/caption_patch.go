package modbot

import (
	"strings"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// ExtractCommandFromCaption extracts a command and its arguments from a message caption if it exists.
func ExtractCommandFromCaption(message *tgbotapi.Message) (isCommand bool, command string, args string) {
	if message.Caption == "" {
		return false, "", ""
	}

	// Because CaptionEntities isn't supported in this specific v1.0.1 fork,
	// we will manually parse the caption to see if it starts with a command.
	if strings.HasPrefix(message.Caption, "/") {
		parts := strings.SplitN(message.Caption, " ", 2)
		cmd := parts[0][1:] // Remove leading slash

		// Remove bot username if present (e.g., /addrule@BotName)
		if idx := strings.Index(cmd, "@"); idx != -1 {
			cmd = cmd[:idx]
		}

		arguments := ""
		if len(parts) > 1 {
			arguments = strings.TrimSpace(parts[1])
		}

		return true, cmd, arguments
	}

	return false, "", ""
}
