package view

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/model"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	tgbotapiv5 "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// SendMessage sends a simple text message to the user
func SendMessagehtml(bot *tgbotapi.BotAPI, chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	_, err := bot.Send(msg)
	return err
}
func ReplyToMessage(bot *tgbotapi.BotAPI, mesgID int, chatID int64, text string) (tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyToMessageID = mesgID
	res, err := bot.Send(msg)
	return res, err
}
func SendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) (tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	res, err := bot.Send(msg)
	return res, err
}

// SendMessageWithButtons sends a message with inline keyboard buttons to the user
func SendMessageWithMarkDown(bot *tgbotapi.BotAPI, chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapiv5.ModeMarkdownV2
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

// SendMessageWithButtons sends a message with inline keyboard buttons to the user
func SendMessageWithKeyboardButton(bot *tgbotapi.BotAPI, chatID int64, text string, buttons tgbotapi.InlineKeyboardButton) error {
	// button := tgbotapi.NewInlineKeyboardButtonURL("Click Here", "url")
	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup([]tgbotapi.InlineKeyboardButton{buttons})

	// Create the message with the inline keyboard
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = inlineKeyboard

	// Send the message
	_, err := bot.Send(msg)
	return err
}

// SendSticker sends a sticker to the user
func SendSticker(bot *tgbotapi.BotAPI, chatID int64, stickerFileID string) error {
	sticker := tgbotapi.NewStickerShare(chatID, stickerFileID)
	_, err := bot.Send(sticker)
	return err
}
func ReactToMessage(botToken string, chatID int64, messageID int, emoji string, isBig bool) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/setMessageReaction", botToken)

	reqBody := model.ReactionRequest{
		ChatID:    chatID,
		MessageID: messageID,
		Reaction: []model.Reaction{
			{
				Type:  "emoji",
				Emoji: emoji,
			},
		},
		IsBig: isBig,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Telegram API responded with status: %s", resp.Status)
	}

	return nil
}

// SendFile sends a file/document to the user
func SendFile(bot *tgbotapi.BotAPI, chatID int64, filename string, data []byte) error {
	file := tgbotapi.FileBytes{
		Name:  filename,
		Bytes: data,
	}
	msg := tgbotapi.NewDocumentUpload(chatID, file)
	_, err := bot.Send(msg)
	return err
}
