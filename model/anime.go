package model

import (
	"encoding/json"
	"errors"
	"math/rand"
	"net/http"
	"time"
)

type Joke struct {
	ID        int    `json:"id"`
	Type      string `json:"type"`
	Setup     string `json:"setup"`
	Punchline string `json:"punchline"`
}
type WordList struct {
	CommonWords []string `json:"commonWords"`
}

// GetRandomWord fetches a random word from the provided API
func GetRandomWord() (string, error) {
	resp, err := http.Get("https://raw.githubusercontent.com/MUSTAFA-A-KHAN/json-data-hub/refs/heads/main/words.json")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var words WordList
	if err := json.NewDecoder(resp.Body).Decode(&words); err != nil {
		return "", err
	}

	if len(words.CommonWords) == 0 {
		return "", errors.New("no words found")
	}

	// Generate a random index
	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(words.CommonWords))

	// Return the random word
	return words.CommonWords[randomIndex], nil
}
