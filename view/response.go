package view

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/config"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/model"
	tgbotapiv5Ovy "github.com/OvyFlash/telegram-bot-api"
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

func ReplyToMessageWithButtonsHTML(bot *tgbotapi.BotAPI, mesgID int, chatID int64, text string, buttons tgbotapi.InlineKeyboardMarkup) (tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyToMessageID = mesgID
	msg.ParseMode = tgbotapi.ModeHTML
	if len(buttons.InlineKeyboard) > 0 {
		msg.ReplyMarkup = buttons
	}
	res, err := bot.Send(msg)
	return res, err
}

const CustomWordMessageEffectID = "5066576334143095943"

func SendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) (tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	res, err := bot.Send(msg)
	return res, err
}
func SendMessageMarkdown(bot *tgbotapi.BotAPI, chatID int64, text string) (tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	res, err := bot.Send(msg)
	return res, err
}

// Only works  with telegram premium accounts and the effect ID must be valid and available to the bot. If the effect ID is invalid or not available, the message will be sent without the effect.
func SendMessageWithEffectID(bot *tgbotapi.BotAPI, chatID int64, text string, effectID string) (tgbotapi.Message, error) {
	params := url.Values{}
	params.Add("chat_id", strconv.FormatInt(chatID, 10))
	params.Add("text", text)
	if effectID != "" {
		params.Add("message_effect_id", effectID)
	}

	apiResp, err := bot.MakeRequest("sendMessage", params)
	if err != nil {
		return tgbotapi.Message{}, err
	}

	var msg tgbotapi.Message
	if err := json.Unmarshal(apiResp.Result, &msg); err != nil {
		return tgbotapi.Message{}, err
	}

	return msg, nil
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
func SendMessageWithButtonsV2(chatID int64, text string, buttons tgbotapi.InlineKeyboardMarkup, usrid int, msgID int) error {
	msg := tgbotapiv5Ovy.NewMessage(chatID, text)
	msg.ParseMode = tgbotapiv5Ovy.ModeMarkdown
	msg.ReceiverUserID = int64(usrid)
	msg.ReplyParameters.EphemeralMessageID = msgID
	if len(buttons.InlineKeyboard) > 0 {
		msg.ReplyMarkup = buttons
	}
	bot, _ := tgbotapiv5Ovy.NewBotAPI(config.App.CatTelegramToken)
	_, err := bot.Send(msg)
	// update, err := bot.HandleUpdate(nil)
	// update.Message
	log.Print("Error:", err)
	return err
}

// SendRichMessage sends a rich message using the OvyFlash library with blocks (paragraphs, tables, images, etc.)
func SendRichMessage(chatID int64, richMessage tgbotapiv5Ovy.InputRichMessage) error {
	bot, _ := tgbotapiv5Ovy.NewBotAPI(config.App.CatTelegramToken)
	_, err := bot.SendRichMessage(tgbotapiv5Ovy.NewSendRichMessage(chatID, richMessage))
	if err != nil {
		log.Print("Error sending rich message:", err)
	}
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

func ReplyToMessageWithButtons(bot *tgbotapi.BotAPI, mesgID int, chatID int64, text string, buttons tgbotapi.InlineKeyboardMarkup) (tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyToMessageID = mesgID
	msg.ParseMode = tgbotapi.ModeMarkdown
	if len(buttons.InlineKeyboard) > 0 {
		msg.ReplyMarkup = buttons
	}
	res, err := bot.Send(msg)
	return res, err
}

func ReplyToMessageWithPhotoAndButtons(bot *tgbotapi.BotAPI, mesgID int, chatID int64, photoData []byte, caption string, buttons tgbotapi.InlineKeyboardMarkup) (tgbotapi.Message, error) {
	file := tgbotapi.FileBytes{
		Name:  "wordle.png",
		Bytes: photoData,
	}
	msg := tgbotapi.NewPhotoUpload(chatID, file)
	msg.ReplyToMessageID = mesgID
	msg.Caption = caption
	msg.ParseMode = tgbotapi.ModeMarkdown
	if len(buttons.InlineKeyboard) > 0 {
		msg.ReplyMarkup = buttons
	}
	res, err := bot.Send(msg)
	return res, err
}

func EditMessageMediaWithButtons(bot *tgbotapi.BotAPI, chatID int64, messageID int, mediaURL string, caption string, buttons tgbotapi.InlineKeyboardMarkup) error {
	params := url.Values{}
	params.Add("chat_id", strconv.FormatInt(chatID, 10))
	params.Add("message_id", strconv.Itoa(messageID))

	mediaObj := map[string]string{
		"type":       "photo",
		"media":      mediaURL,
		"caption":    caption,
		"parse_mode": "Markdown",
	}

	mediaBytes, err := json.Marshal(mediaObj)
	if err != nil {
		return err
	}
	params.Add("media", string(mediaBytes))

	replyMarkupBytes, err := json.Marshal(buttons)
	if err != nil {
		return err
	}
	params.Add("reply_markup", string(replyMarkupBytes))

	_, err = bot.MakeRequest("editMessageMedia", params)
	return err
}
func DeleteMessageAfterDelay(bot *tgbotapi.BotAPI, chatID int64, messageID int, delay time.Duration) {
	time.Sleep(1 * time.Second)
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	_, err := bot.DeleteMessage(deleteMsg)
	if err != nil {
		log.Printf("Failed to delete message: %v", err)
	}

}

var ovyBot *tgbotapiv5Ovy.BotAPI

// SendScramyRichMessage sends a rich message specifically designed for Scramy H1 mode using github.com/OvyFlash/telegram-bot-api
func SendScramyRichMessage(botToken string, chatID int64, textTop string, letters string, textBottom string, buttons tgbotapi.InlineKeyboardMarkup) error {
	if ovyBot == nil {
		bot, err := tgbotapiv5Ovy.NewBotAPI(botToken)
		if err != nil {
			return fmt.Errorf("failed to init OvyFlash bot: %w", err)
		}
		ovyBot = bot
	}

	// Construct the OvyFlash rich message blocks
	msg := tgbotapiv5Ovy.NewSendRichMessage(chatID, tgbotapiv5Ovy.NewInputRichMessageBlocks(
		tgbotapiv5Ovy.InputRichBlockParagraph{
			Type: "paragraph",
			Text: textTop,
		},
		tgbotapiv5Ovy.InputRichBlockSectionHeading{
			Type: "section_heading",
			Text: letters,
			Size: 1, // H1
		},
		tgbotapiv5Ovy.InputRichBlockParagraph{
			Type: "paragraph",
			Text: textBottom,
		},
	))

	// Convert tgbotapi.InlineKeyboardMarkup to tgbotapiv5Ovy.InlineKeyboardMarkup
	var ovyKeyboard [][]tgbotapiv5Ovy.InlineKeyboardButton
	for _, row := range buttons.InlineKeyboard {
		var ovyRow []tgbotapiv5Ovy.InlineKeyboardButton
		for _, btn := range row {
			var ovyBtn tgbotapiv5Ovy.InlineKeyboardButton
			ovyBtn.Text = btn.Text
			if btn.CallbackData != nil {
				ovyBtn.CallbackData = btn.CallbackData
			}
			if btn.URL != nil {
				ovyBtn.URL = btn.URL
			}
			// Copy other fields if needed, but CallbackData and URL are standard
			ovyRow = append(ovyRow, ovyBtn)
		}
		ovyKeyboard = append(ovyKeyboard, ovyRow)
	}

	if len(ovyKeyboard) > 0 {
		msg.ReplyMarkup = tgbotapiv5Ovy.InlineKeyboardMarkup{
			InlineKeyboard: ovyKeyboard,
		}
	}

	_, err := ovyBot.SendRichMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to send rich message request: %w", err)
	}

	return nil
}

// SendMessagehtmlWithButtons sends an HTML message with inline keyboard buttons to the user
func SendMessagehtmlWithButtons(bot *tgbotapi.BotAPI, chatID int64, text string, buttons tgbotapi.InlineKeyboardMarkup) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	if len(buttons.InlineKeyboard) > 0 {
		msg.ReplyMarkup = buttons
	}
	_, err := bot.Send(msg)
	return err
}
