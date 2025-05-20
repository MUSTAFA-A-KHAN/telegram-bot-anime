package service

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
)

func LeaderBoardList(collection string) string {
	// Get MongoDB client
	client := repository.DbManager()
	idCounts, err := repository.CountIDOccurrences(client, collection)
	if err != nil {
		log.Fatal(err)
	}

	// Limit to top 10 players
	limit := 10
	if len(idCounts) < limit {
		limit = len(idCounts)
	}

	// Create header for leaderboard with emojis and formatting
	leaderboard := "🏆 *Top 10 Players Leaderboard* 🏆\n\n"
	leaderboard += fmt.Sprintf("%-4s %-20s %s\n", "Rank", "Player", "Score")
	leaderboard += strings.Repeat("─", 32) + "\n"

	// Iterate over top players and format each entry
	for i := 0; i < limit; i++ {
		count := idCounts[i]
		name := fmt.Sprintf("%v", count["Name"])
		score := fmt.Sprintf("%v", count["count"])
		leaderboard += fmt.Sprintf("%-4d %-20s %s\n", i+1, name, score)
	}

	// Add footer message with emoji
	leaderboard += "\n✨ Keep it up and aim for the top! ✨"

	// Close the MongoDB client connection
	err = client.Disconnect(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	return leaderboard
}

// GetUserStatsByID returns formatted stats string for a given user ID
func GetUserStatsByID(userID int) string {
	client := repository.DbManager()
	defer func() {
		err := client.Disconnect(context.TODO())
		if err != nil {
			log.Fatal(err)
		}
	}()

	result, err := repository.GetUserStatsByID(client, "CrocEn", userID)
	if err != nil {
		return fmt.Sprintf("No stats found for user ID %d", userID)
	}

	name, _ := result["Name"].(string)
	count, _ := result["count"].(int32) // MongoDB returns int32 for count

	return fmt.Sprintf("You  %s have successfully guessed for :\n %d Times", name, count)
}
