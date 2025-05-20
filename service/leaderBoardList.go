package service

import (
	"context"
	"fmt"
	"log"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
)

func LeaderBoardList(collection string) string {
	// Get MongoDB client
	client := repository.DbManager()
	idCounts, err := repository.CountIDOccurrences(client, collection)
	if err != nil {
		log.Fatal(err)
	}

	// Create header for leaderboard
	leaderboard := "⭐ Leaderboard ⭐\n\n"

	// Iterate over ID counts and format each entry
	for i, count := range idCounts {
		leaderboard += fmt.Sprintf("%d. %v – %v\n", i+1, count["Name"], count["count"])
	}

	// Add footer message
	leaderboard += "\n✨ Keep it up and aim for the top! \u2728"

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

	return fmt.Sprintf("Stats for user %s (ID: %d):\nCount: %d", name, userID, count)
}
