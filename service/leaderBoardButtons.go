package service

import (
	"fmt"
	"log"
	"strings"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/view"
	"go.mongodb.org/mongo-driver/mongo"
)

func LeaderBoardListButtons(client *mongo.Client, collection string, chatID int64, callbackData string) *view.CustomInlineKeyboardMarkup {
	idCounts, err := repository.CountIDOccurrences(client, collection, chatID)
	if err != nil {
		log.Printf("Error getting leaderboard: %v", err)
	}

	limit := 10
	if len(idCounts) < limit {
		limit = len(idCounts)
	}

	var buttons [][]view.CustomInlineKeyboardButton

	if limit == 0 {
		btn := view.CustomInlineKeyboardButton{
			Text:         "No stats found yet!",
			CallbackData: "ignore",
		}
		buttons = append(buttons, []view.CustomInlineKeyboardButton{btn})
	}

	rankEmojis := []string{"🥇", "🥈", "🥉"}
	styles := []string{"primary", "success", "danger"}

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
		} else if collection == "ScramyEn" {
			score += " 💎"
		}

		rankDisplay := fmt.Sprintf("%d", i+1)
		style := "primary" // Telegram only supports "primary", "success", "danger"
		if i < 3 {
			rankDisplay = rankEmojis[i]
			style = styles[i]
		} else {
			rankDisplay = "⭐ " + rankDisplay
			style = ""
		}

		text := fmt.Sprintf("%s | %s | %s", rankDisplay, name, score)

		btn := view.CustomInlineKeyboardButton{
			Text:         text,
			CallbackData: "ignore", // Button does nothing
			Style:        style,
		}

		buttons = append(buttons, []view.CustomInlineKeyboardButton{btn})
	}

	// Add navigation buttons based on the type of stats
	isGlobal := strings.HasPrefix(callbackData, "statsglobal")

	wordGuessLabel := "Word Guess"
	wordleLabel := "Wordle"
	scramyLabel := "Scramy"

	if strings.HasSuffix(callbackData, "wordguess") {
		wordGuessLabel = "✅ " + wordGuessLabel
	} else if strings.HasSuffix(callbackData, "wordle") {
		wordleLabel = "✅ " + wordleLabel
	} else if strings.HasSuffix(callbackData, "scramy") {
		scramyLabel = "✅ " + scramyLabel
	}

	var navPrefix string
	if isGlobal {
		navPrefix = "statsglobal_"
		wordGuessLabel += " Global"
		wordleLabel += " Global"
		scramyLabel += " Global"
	} else {
		navPrefix = "statsgroup_"
		wordGuessLabel += " Group"
		wordleLabel += " Group"
		scramyLabel += " Group"
	}

	navRow1 := []view.CustomInlineKeyboardButton{
		{Text: wordGuessLabel, CallbackData: navPrefix + "wordguess"},
		{Text: wordleLabel, CallbackData: navPrefix + "wordle"},
	}
	navRow2 := []view.CustomInlineKeyboardButton{
		{Text: scramyLabel, CallbackData: navPrefix + "scramy"},
	}

	buttons = append(buttons, navRow1)
	buttons = append(buttons, navRow2)

	return &view.CustomInlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}
}
