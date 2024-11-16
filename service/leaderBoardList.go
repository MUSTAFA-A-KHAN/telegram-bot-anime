package service

import (
	"context"
	"fmt"
	"log"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
)

func LeaderBoardList() string {
	// Get MongoDB client
	client := repository.DbManager()
	idCounts, err := repository.CountIDOccurrences(client)
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
