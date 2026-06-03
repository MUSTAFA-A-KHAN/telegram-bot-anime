package view

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type CustomInlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
	Style        string `json:"style,omitempty"`
}

type CustomInlineKeyboardMarkup struct {
	InlineKeyboard [][]CustomInlineKeyboardButton `json:"inline_keyboard"`
}

type SendMessageRequest struct {
	ChatID      int64                       `json:"chat_id"`
	Text        string                      `json:"text"`
	ParseMode   string                      `json:"parse_mode,omitempty"`
	ReplyMarkup *CustomInlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

type EditMessageRequest struct {
	ChatID      int64                       `json:"chat_id"`
	MessageID   int                         `json:"message_id"`
	Text        string                      `json:"text"`
	ParseMode   string                      `json:"parse_mode,omitempty"`
	ReplyMarkup *CustomInlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

func SendMessageWithStyledButtons(botToken string, chatID int64, text string, markup *CustomInlineKeyboardMarkup) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	reqBody := SendMessageRequest{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   "HTML",
		ReplyMarkup: markup,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("failed to marshal request: %v", err)
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("failed to send request: %v", err)
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("Telegram API responded with status: %s, body: %s", resp.Status, string(bodyBytes))
		return fmt.Errorf("Telegram API responded with status: %s", resp.Status)
	}

	return nil
}

func EditMessageTextWithStyledButtons(botToken string, chatID int64, messageID int, text string, markup *CustomInlineKeyboardMarkup) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/editMessageText", botToken)

	reqBody := EditMessageRequest{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        text,
		ParseMode:   "HTML",
		ReplyMarkup: markup,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("failed to marshal request: %v", err)
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("failed to send request: %v", err)
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("Telegram API responded with status: %s, body: %s", resp.Status, string(bodyBytes))
		return fmt.Errorf("Telegram API responded with status: %s", resp.Status)
	}

	return nil
}
