package service

import (
	"fmt"
	"log"
	"strings"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
	"go.mongodb.org/mongo-driver/mongo"
)

func LeaderBoardList(client *mongo.Client, collection string) string {
	idCounts, err := repository.CountIDOccurrences(client, collection)
	if err != nil {
		log.Fatal(err)
	}

	limit := 10
	if len(idCounts) < limit {
		limit = len(idCounts)
	}

	rankEmojis := []string{"🥇", "🥈", "🥉"}

	leaderboard := "🏆 <b>Top 10 Players Leaderboard</b> 🏆\n\n"
	leaderboard += "<pre>"
	leaderboard += fmt.Sprintf("%-6s | %-20s | %s\n", "Rank", "Player", "Score")
	leaderboard += strings.Repeat("─", 38) + "\n"

	for i := 0; i < limit; i++ {
		count := idCounts[i]
		name := fmt.Sprintf("%v", count["Name"])
		score := fmt.Sprintf("%v", count["count"])
		rankDisplay := fmt.Sprintf("%d", i+1)
		if i < 3 {
			rankDisplay = rankEmojis[i]
		} else {
			rankDisplay = "⭐ " + rankDisplay
		}
		leaderboard += fmt.Sprintf("%-6s | %-20s | %s\n", rankDisplay, name, score)
	}

	leaderboard += "</pre>"
	leaderboard += "\n✨ <b>Keep it up and aim for the top!</b> ✨\n"

	return leaderboard
}

func GetUserStatsByID(client *mongo.Client, userID int) string {

	result, err := repository.GetUserStatsByID(client, "CrocEn", userID)
	stats := "something went wrong"
	if err != nil {
		stats = "No winning stats found"
	} else {
		name, _ := result["Name"].(string)
		count, _ := result["count"].(int32)

		stats = fmt.Sprintf("You %s have successfully guessed for:\n%d Times", name, count)
	}

	result, err = repository.GetUserStatsByID(client, "CrocEnLeader", userID)
	if err != nil {
		return stats + "\n:)"
	}

	// name, _ = result["Name"].(string)
	count, _ := result["count"].(int32)

	stats += fmt.Sprintf("\nAnd have leaded for:\n%d Times", count)
	return stats

}
