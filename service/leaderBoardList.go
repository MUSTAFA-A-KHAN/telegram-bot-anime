package service

import (
	"fmt"
	"html"
	"log"
	"strconv"
	"strings"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
	"go.mongodb.org/mongo-driver/mongo"
)

func LeaderBoardList(client *mongo.Client, collection string, chatID int64) string {
	idCounts, err := repository.CountIDOccurrences(client, collection, chatID)
	if err != nil {
		log.Fatal(err)
	}

	limit := 10
	if len(idCounts) < limit {
		limit = len(idCounts)
	}

	rankEmojis := []string{"🥇", "🥈", "🥉"}

	leaderboard := "🏆 <b>Top 10 Players Leaderboard</b> 🏆\n<blockquote expandable>\n"

	for i := 0; i < limit; i++ {
		count := idCounts[i]
		name := fmt.Sprintf("%v", count["Name"])

		// Fetch and append equipped emojis to the user's name
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
		rankDisplay := strconv.Itoa(i + 1)
		if i < 3 {
			rankDisplay = rankEmojis[i]
		} else {
			rankDisplay = "⭐ " + rankDisplay
		}
		leaderboard += fmt.Sprintf("<b>%s</b> %s — %s\n", rankDisplay, name, score)
	}

	leaderboard += "</blockquote>\n✨ <b>Keep it up and aim for the top!</b> ✨\n"

	return leaderboard
}

func GetUserStatsByID(client *mongo.Client, userID int) string {

	result, err := repository.GetUserStatsByID(client, "CrocEn", userID)
	stats := "something went wrong"
	isBlockquoteOpen := false
	if err != nil {
		stats = "No winning stats found"
	} else {
		name, _ := result["Name"].(string)

		equippedEmojis, err := repository.GetEquippedEmojis(client, userID)
		if err == nil && len(equippedEmojis) > 0 {
			name += " " + strings.Join(equippedEmojis, "")
		}

		count, _ := result["count"].(int32)

		stats = fmt.Sprintf("📊 <b>Word Guess Stats</b>\n<blockquote>\n👤 <b>Player:</b> %s\n\n🎯 <b>Correct Guesses:</b> %d", html.EscapeString(name), count)
		isBlockquoteOpen = true
	}

	result, err = repository.GetUserStatsByID(client, "CrocEnLeader", userID)
	if err != nil {
		if isBlockquoteOpen {
			stats += "\n</blockquote>"
		}
		return stats
	}

	// name, _ = result["Name"].(string)
	count, _ := result["count"].(int32)

	if !isBlockquoteOpen {
		stats += "\n<blockquote>"
	}
	stats += fmt.Sprintf("\n\n👑 <b>Times Leaded:</b> %d\n</blockquote>", count)
	return stats

}

func GetWordleUserStatsByID(client *mongo.Client, userID int) string {
	result, err := repository.GetUserStatsByID(client, "WordleEn", userID)
	stats := "something went wrong"
	if err != nil {
		stats = "No winning stats found"
	} else {
		name, _ := result["Name"].(string)

		equippedEmojis, err := repository.GetEquippedEmojis(client, userID)
		if err == nil && len(equippedEmojis) > 0 {
			name += " " + strings.Join(equippedEmojis, "")
		}

		count := 0
		if val, ok := result["count"]; ok {
			switch v := val.(type) {
			case int32:
				count = int(v)
			case int64:
				count = int(v)
			case int:
				count = v
			}
		}

		stats = fmt.Sprintf("📊 <b>Wordle Stats</b>\n<blockquote>\n👤 <b>Player:</b> %s\n\n🪙 <b>Points:</b> %d\n</blockquote>", html.EscapeString(name), count)
	}

	return stats
}

func GetScramyUserStatsByID(client *mongo.Client, userID int) string {
	result, err := repository.GetUserStatsByID(client, "ScramyEn", userID)
	stats := "something went wrong"
	if err != nil {
		stats = "No winning stats found"
	} else {
		name, _ := result["Name"].(string)
		count := 0
		if val, ok := result["count"]; ok {
			switch v := val.(type) {
			case int32:
				count = int(v)
			case int64:
				count = int(v)
			case int:
				count = v
			}
		}

		stats = fmt.Sprintf("📊 <b>Scramy Stats</b>\n<blockquote>\n👤 <b>Player:</b> %s\n\n💎 <b>Points:</b> %d\n</blockquote>", html.EscapeString(name), count)
	}
	return stats
}
