package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/STRockefeller/dictionaries"
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

// GenerateHint generates a hint for the given word by revealing the first letter and the length of the word.
func GenerateHint(word string) string {
	if len(word) == 0 {
		return "No word available for hint."
	}
	firstLetter := strings.ToUpper(string(word[0]))
	length := strconv.Itoa(len(word))
	return "Hint: The word starts with '" + firstLetter + "' and is " + length + " letters long."
}

var meaningCache = struct {
	sync.RWMutex
	m map[string]string
}{m: make(map[string]string)}

// fetchMeaningFromAPI fetches the meaning of a word from a dictionary API (OwlBot API used here as example).
func fetchMeaningFromAPI(word string) (string, error) {
	dict := dictionaries.NewEnglishDictionary()
	result, err := dict.Search(word)
	if err != nil {
		return "", err
	}
	fmt.Println(result)

	// Display the first available definition
	if len(result) > 0 && len(result[0].Meanings) > 0 && len(result[0].Meanings[0].Definitions) > 0 {
		return result[0].Meanings[0].Definitions[0].Definition, nil
	}
	return "", errors.New("no definition found")
}

// GenerateMeaningHint generates a hint explaining the meaning or context of the given word.
// It uses a dictionary API to fetch real meanings and caches results for sustainability.
func GenerateMeaningHint(word string) string {
	if len(word) == 0 {
		return "No word available for meaning hint."
	}

	meaningCache.RLock()
	cachedMeaning, found := meaningCache.m[word]
	meaningCache.RUnlock()
	if found {
		return "Meaning hint: " + cachedMeaning
	}

	meaning, err := fetchMeaningFromAPI(word)
	meaning = strings.ReplaceAll(meaning, word, "_")
	if err != nil {
		// fallback to placeholder
		return "Meaning hint: This is a placeholder for the meaning or context of the word: " + word + ":(."
	}

	meaningCache.Lock()
	meaningCache.m[word] = meaning
	meaningCache.Unlock()
	return "ⓘ Hint: " + meaning
}
