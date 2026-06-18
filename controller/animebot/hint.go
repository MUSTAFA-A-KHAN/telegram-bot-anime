package animebot

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/view"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type jikanResponse struct {
	Data []struct {
		Images struct {
			Webp struct {
				ImageURL string `json:"image_url"`
			} `json:"webp"`
		} `json:"images"`
	} `json:"data"`
}

// HandleAnimeHint fetches the cover image from Jikan API and sends it as a sticker hint
func HandleAnimeHint(bot *tgbotapi.BotAPI, chatID int64) {
	activeGamesMu.Lock()
	state, exists := activeGames[chatID]
	activeGamesMu.Unlock()

	if !exists || !state.Active {
		view.SendMessage(bot, chatID, "No active Anime game. Start one with /anime!")
		return
	}

	answer := state.Question.Answers[0]

	// Basic normalization to improve search
	searchQuery := answer
	if strings.Contains(strings.ToLower(answer), "fullmetal alchemist") {
		searchQuery = "Fullmetal Alchemist"
	}

	apiURL := fmt.Sprintf("https://api.jikan.moe/v4/anime?q=%s&sfw=true", url.QueryEscape(searchQuery))
	resp, err := http.Get(apiURL)
	if err != nil {
		view.SendMessage(bot, chatID, "Failed to get hint right now 😔")
		return
	}
	defer resp.Body.Close()

	var jResp jikanResponse
	if err := json.NewDecoder(resp.Body).Decode(&jResp); err != nil {
		view.SendMessage(bot, chatID, "Failed to get hint right now 😔")
		return
	}

	if len(jResp.Data) > 0 {
		imageURL := jResp.Data[0].Images.Webp.ImageURL
		err := view.SendSticker(bot, chatID, imageURL)
		if err != nil {
			// fallback if sticker send fails
			view.SendMessagehtml(bot, chatID, fmt.Sprintf("Here is a hint: <a href=\"%s\">Image</a>", imageURL))
		}
	} else {
		view.SendMessage(bot, chatID, "Couldn't find a picture hint for this one! 🤔")
	}
}
