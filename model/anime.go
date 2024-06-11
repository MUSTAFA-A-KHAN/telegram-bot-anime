package model

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Joke struct {
	ID        int    `json:"id"`
	Type      string `json:"type"`
	Setup     string `json:"setup"`
	Punchline string `json:"punchline"`
}

// GetRandomWord fetches a random word from the provided API
func GetRandomWord() (string, error) {
	resp, err := http.Get("https://random-word-api.herokuapp.com/word")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var words []string
	if err := json.NewDecoder(resp.Body).Decode(&words); err != nil {
		return "", err
	}

	if len(words) == 0 {
		return "", errors.New("no words found")
	}

	return words[0], nil
}
