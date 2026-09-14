package en

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	baseURL   = "https://en.wiktionary.org/api/rest_v1/page/definition/"
	userAgent = "telegram-bot-anime/1.0 (https://github.com/MUSTAFA-A-KHAN/telegram-bot-anime)"
)

type Wiktionary struct {
	client *http.Client
}

func NewWiktionary() *Wiktionary {
	return &Wiktionary{
		client: &http.Client{},
	}
}

type response map[string][]Entry

type Entry struct {
	PartOfSpeech string       `json:"partOfSpeech"`
	Definitions  []Definition `json:"definitions"`
}

type Definition struct {
	Definition     string `json:"definition"`
	ParsedExamples []struct {
		Example string `json:"example"`
	} `json:"parsedExamples"`
}

func (w *Wiktionary) GetMeaning(word string) (string, error) {
	word = strings.TrimSpace(word)

	if word == "" {
		return "", fmt.Errorf("word cannot be empty")
	}
	endpoint := baseURL + url.PathEscape(word)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	// Wikimedia rejects requests without a descriptive User-Agent (403).
	req.Header.Set("User-Agent", userAgent)

	resp, err := w.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request wiktionary: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("word %q not found", word)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("wiktionary returned status %d", resp.StatusCode)
	}
	var data response
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", fmt.Errorf("decode wiktionary response: %w", err)
	}
	meanings, ok := data["en"]
	if !ok {
		return "", fmt.Errorf("no English definition found for %q", word)
	}
	var meaningss string
	for _, meaning := range meanings {
		parts := meaning.PartOfSpeech
		if parts != "" {
			meaningss += parts + ":\n"
		}
		for _, definition := range meaning.Definitions {
			if definition.Definition != "" {
				meaningss += definition.Definition + "\n"
			}
		}
	}
	if meaningss == "" {
		return "", fmt.Errorf("no definition found for %q", word)
	}
	return meaningss, nil
}

// func (w *Wiktionary) GetMeaning(word string) (string, error) {
// 	word = strings.TrimSpace(word)

// 	if word == "" {
// 		return "", fmt.Errorf("word cannot be empty")
// 	}

// 	endpoint := baseURL + url.PathEscape(word)

// 	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
// 	if err != nil {
// 		return "", fmt.Errorf("create request: %w", err)
// 	}

// 	req.Header.Set("User-Agent", "telegram-bot-anime/1.0")

// 	resp, err := w.client.Do(req)
// 	if err != nil {
// 		return "", fmt.Errorf("request wiktionary: %w", err)
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode == http.StatusNotFound {
// 		return "", fmt.Errorf("word %q not found", word)
// 	}

// 	if resp.StatusCode != http.StatusOK {
// 		return "", fmt.Errorf("wiktionary returned status %d", resp.StatusCode)
// 	}

// 	var data response

// 	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
// 		return "", fmt.Errorf("decode wiktionary response: %w", err)
// 	}

// 	entries, ok := data["en"]
// 	if !ok {
// 		return "", fmt.Errorf("no English definition found for %q", word)
// 	}

// 	for _, entry := range entries {
// 		for _, definition := range entry.Definitions {
// 			if definition.Definition != "" {
// 				return definition.Definition, nil
// 			}
// 		}
// 	}

// 	return "", fmt.Errorf("no definition found for %q", word)
// }
