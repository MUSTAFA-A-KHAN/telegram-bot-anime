package en

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	baseURL   = "https://en.wiktionary.org/api/rest_v1/page/definition/"
	userAgent = "telegram-bot-anime/1.0 (https://github.com/MUSTAFA-A-KHAN/telegram-bot-anime)"

	maxDefinitionsPerPOS = 3
	// Telegram photo captions are limited to 1024 characters and the meaning
	// shares the caption with the game result text.
	maxMeaningLength = 600
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
	formatted := formatMeanings(meanings)
	if formatted == "" {
		return "", fmt.Errorf("no definition found for %q", word)
	}
	return formatted, nil
}

// formatMeanings renders entries as plain text grouped by part of speech,
// keeping at most maxDefinitionsPerPOS unique definitions each and staying
// within maxMeaningLength runes so it fits in a Telegram photo caption.
func formatMeanings(entries []Entry) string {
	var b strings.Builder
	seen := make(map[string]bool)

	for _, entry := range entries {
		var defs []string
		for _, d := range entry.Definitions {
			text := cleanDefinition(d.Definition)
			if text == "" || seen[text] {
				continue
			}
			seen[text] = true
			defs = append(defs, text)
			if len(defs) == maxDefinitionsPerPOS {
				break
			}
		}
		if len(defs) == 0 {
			continue
		}

		var section strings.Builder
		if entry.PartOfSpeech != "" {
			section.WriteString(entry.PartOfSpeech + ":\n")
		}
		for i, d := range defs {
			fmt.Fprintf(&section, "%d. %s\n", i+1, d)
		}

		if b.Len() > 0 && utf8.RuneCountInString(b.String()+section.String()) > maxMeaningLength {
			break
		}
		b.WriteString(section.String())
	}

	return truncateRunes(strings.TrimRight(b.String(), "\n"), maxMeaningLength)
}

var (
	styleRe = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	tagRe   = regexp.MustCompile(`<[^>]*>`)
)

// cleanDefinition converts a Wiktionary HTML definition into plain text.
func cleanDefinition(s string) string {
	// Nested <ol> lists hold sub-senses that the API also returns as
	// separate definitions, so drop them to avoid duplicates.
	if i := strings.Index(s, "<ol"); i >= 0 {
		s = s[:i]
	}
	s = styleRe.ReplaceAllString(s, "")
	s = tagRe.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	// Backticks would break the Markdown code block the bot wraps this in.
	s = strings.ReplaceAll(s, "`", "'")
	return strings.Join(strings.Fields(s), " ")
}

func truncateRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return strings.TrimSpace(string(r[:max-1])) + "…"
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
