package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
)

func main() {
	client := repository.DbManager()
	if client == nil {
		log.Fatal("Could not connect to database")
	}

	collections := []string{"CrocEnLeader", "WordleEn", "ScramyEn", "GeographyPoints"}
	allData := make(map[string]interface{})

	for _, coll := range collections {
		data, err := repository.CountIDOccurrences(client, coll, 0)
		if err != nil {
			log.Printf("Error fetching data for %s: %v", coll, err)
			continue
		}
		allData[coll] = data
	}

	jsonData, err := json.MarshalIndent(allData, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling JSON: %v", err)
	}

	err = os.WriteFile("leaderboard.json", jsonData, 0644)
	if err != nil {
		log.Fatalf("Error writing file: %v", err)
	}

	fmt.Println("Successfully exported leaderboard.json")
}
