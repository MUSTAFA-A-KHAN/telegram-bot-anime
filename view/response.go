package view

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// SendMessage sends a simple text message to the user
func SendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := bot.Send(msg)
	return err
}

// SendMessageWithButtons sends a message with inline keyboard buttons to the user
func SendMessageWithButtons(bot *tgbotapi.BotAPI, chatID int64, text string, buttons tgbotapi.InlineKeyboardMarkup) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	if len(buttons.InlineKeyboard) > 0 {
		msg.ReplyMarkup = buttons
	}
	_, err := bot.Send(msg)
	return err
}

// SendSticker sends a sticker to the user
func SendSticker(bot *tgbotapi.BotAPI, chatID int64, stickerFileID string) error {
	sticker := tgbotapi.NewStickerShare(chatID, stickerFileID)
	_, err := bot.Send(sticker)
	return err
}
