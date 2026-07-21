package view

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func EditMessageMediaWithStyledButtons(botToken string, chatID int64, messageID int, media []byte, filename string, markup *tgbotapi.InlineKeyboardMarkup) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/editMessageMedia", botToken)

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Add chat_id and message_id
	w.WriteField("chat_id", strconv.FormatInt(chatID, 10))
	w.WriteField("message_id", strconv.Itoa(messageID))

	// Add media details as JSON string
	mediaDetails := map[string]interface{}{
		"type":  "photo",
		"media": "attach://" + filename,
	}
	mediaJSON, err := json.Marshal(mediaDetails)
	if err != nil {
		return err
	}
	w.WriteField("media", string(mediaJSON))

	if markup != nil {
		markupJSON, err := json.Marshal(markup)
		if err != nil {
			return err
		}
		w.WriteField("reply_markup", string(markupJSON))
	}

	// Add actual file
	fw, err := w.CreateFormFile(filename, filename)
	if err != nil {
		return err
	}
	if _, err = io.Copy(fw, bytes.NewReader(media)); err != nil {
		return err
	}

	w.Close()

	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(res.Body)
		if bytes.Contains(bodyBytes, []byte("message is not modified")) {
			return nil
		}
		log.Printf("Telegram API responded with status: %s, body: %s", res.Status, string(bodyBytes))
		return fmt.Errorf("Telegram API responded with status: %s", res.Status)
	}

	return nil
}
