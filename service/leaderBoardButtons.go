package service

import (
	"fmt"
	"log"
	"strings"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/view"
	"go.mongodb.org/mongo-driver/mongo"
)

func LeaderBoardListButtons(client *mongo.Client, collection string, chatID int64) *view.CustomInlineKeyboardMarkup {
	idCounts, err := repository.CountIDOccurrences(client, collection, chatID)
	if err != nil {
		log.Printf("Error getting leaderboard: %v", err)
	}

	limit := 10
	if len(idCounts) < limit {
		limit = len(idCounts)
	}

	if limit == 0 {
		return nil
	}

	rankEmojis := []string{"🥇", "🥈", "🥉"}
	styles := []string{"primary", "success", "danger"}

	var buttons [][]view.CustomInlineKeyboardButton

	for i := 0; i < limit; i++ {
		count := idCounts[i]
		name := fmt.Sprintf("%v", count["Name"])

		var userID int
		if id, ok := count["_id"]; ok {
			switch v := id.(type) {
			case int32:
				userID = int(v)
			case int64:
				userID = int(v)
			case int:
				userID = v
			}
		}

		equippedEmojis, err := repository.GetEquippedEmojis(client, userID)
		if err == nil && len(equippedEmojis) > 0 {
			name += " " + strings.Join(equippedEmojis, "")
		}

		score := fmt.Sprintf("%v", count["count"])
		if collection == "WordleEn" {
			score += " 🪙"
		}

		rankDisplay := fmt.Sprintf("%d", i+1)
		style := "primary" // Telegram only supports "primary", "success", "danger"
		if i < 3 {
			rankDisplay = rankEmojis[i]
			style = styles[i]
		} else {
			rankDisplay = "⭐ " + rankDisplay
			style=""
		}

		text := fmt.Sprintf("%s | %s | %s", rankDisplay, name, score)

		btn := view.CustomInlineKeyboardButton{
			Text:         text,
			CallbackData: "ignore", // Button does nothing
			Style:        style,
		}

		buttons = append(buttons, []view.CustomInlineKeyboardButton{btn})
	}

	return &view.CustomInlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}
}
