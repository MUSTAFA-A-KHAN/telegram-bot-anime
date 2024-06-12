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
func SendMessageWithButtons(bot *tgbotapi.BotAPI, chatID int64, text string, buttons []tgbotapi.InlineKeyboardButton) error {
	msg := tgbotapi.NewMessage(chatID, text)
	if len(buttons) > 0 {
		var keyboard [][]tgbotapi.InlineKeyboardButton
		keyboard = append(keyboard, buttons)
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	}
	_, err := bot.Send(msg)
	return err
}
