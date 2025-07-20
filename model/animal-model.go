package model

import (
	"math/rand"
	"time"
)

// GetRandomAnimal returns a random animal name from a static list
func GetRandomAnimal() (string, error) {
	animals := []string{
		"Elephant",
		"Lion",
		"Tiger",
		"Giraffe",
		"Zebra",
		"Kangaroo",
		"Panda",
		"Koala",
		"Penguin",
		"Dolphin",
		"Whale",
		"Shark",
		"Eagle",
		"Owl",
		"Fox",
		"Wolf",
		"Bear",
		"Rabbit",
		"Deer",
		"Horse",
	}

	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(animals))
	return animals[randomIndex], nil
}
