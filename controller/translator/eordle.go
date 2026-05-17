package translator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
)

const eordlePrompt = `You are a Wordle expert. The secret word is exactly 5 letters long. Analyze the previous guesses and their feedback to find the best next 5-letter English word.

Feedback meanings:
🟩 = correct letter in the right position
🟨 = correct letter but wrong position
🟥 = letter not in the word

Input format: Each line is a guess with feedback emojis followed by the guessed word.
Example: 🟩 🟥 🟥 🟥 🟥 GREEN

Consider all constraints from previous guesses:
- Letters marked 🟩 must be in that exact position.
- Letters marked 🟨 must be in the word but not in that position.
- Letters marked 🟥 are not in the word at all.
- Repeated letters follow the above rules.

Recommend exactly one next 5-letter word that fits all constraints and maximizes information gain.
Reply with only the uppercase word, no explanation.`

func (t *TextTranslator) SolveWordle(puzzle string) string {
	input := strings.TrimSpace(puzzle)
	if input == "" {
		return "uwu."
	}
	if OpenAIKey == "" {
		return "OFF"
	}

	// derive constraints from provided puzzle
	pattern, present, excluded, notIn := parseConstraints(input)

	// build constraint summary for the prompt
	mustContain := ""
	if len(present) > 0 {
		parts := make([]string, 0, len(present))
		for _, r := range present {
			parts = append(parts, strings.ToUpper(string(r)))
		}
		mustContain = strings.Join(parts, ",")
	}
	mustExclude := ""
	if len(excluded) > 0 {
		parts := make([]string, 0, len(excluded))
		for _, r := range excluded {
			parts = append(parts, strings.ToUpper(string(r)))
		}
		mustExclude = strings.Join(parts, ",")
	}

	prompt := eordlePrompt + "\n\nConstraints:\n"
	prompt += fmt.Sprintf("Pattern: %s\n", pattern)
	if mustContain != "" {
		prompt += fmt.Sprintf("Must contain (any order): %s\n", mustContain)
	}
	if mustExclude != "" {
		prompt += fmt.Sprintf("Must NOT contain: %s\n", mustExclude)
	}
	// include per-position not-in info if any
	var notInParts []string
	for i := 0; i < 5; i++ {
		if len(notIn[i]) == 0 {
			continue
		}
		chars := make([]string, 0, len(notIn[i]))
		for ch := range notIn[i] {
			chars = append(chars, strings.ToUpper(string(ch)))
		}
		sort.Strings(chars)
		notInParts = append(notInParts, fmt.Sprintf("pos%d: %s", i+1, strings.Join(chars, ",")))
	}
	if len(notInParts) > 0 {
		prompt += fmt.Sprintf("Not in position constraints: %s\n", strings.Join(notInParts, "; "))
	}

	prompt += "\nGiven these constraints, recommend exactly one uppercase 5-letter English word. Reply only with the word."

	// Try up to 3 times to get a valid suggestion from LLM
	for attempt := 0; attempt < 4; attempt++ {
		result := t.callLLM(fmt.Sprintf("%s\n\n%s", prompt, input))
		if result == llmErrorMessage {
			return "Unable to solve the puzzle right now. Please try again later."
		}

		word := extractFirstWord(result)
		if word == "" {
			// if LLM didn't return a clean word, try again with stricter prompt
			prompt += "\nNote: reply with exactly one 5-letter uppercase word only."
			continue
		}

		if validateCandidate(word, pattern, present, excluded, notIn) {
			return fmt.Sprintf("Best next word: %s", word)
		}

		// If invalid, append a clarification and retry
		prompt += fmt.Sprintf("\nYour previous suggestion %s violated the constraints. Try again and ensure pattern, exclusion and not-in-position rules are followed.", word)
	}

	// If AI fails, return the analyzed constraints so user can choose manually
	analysis := t.AnalyzeEordle(input)
	// additionally include top local candidates
	cands, err := generateCandidates(pattern, present, excluded, notIn, 10)
	if err == nil && len(cands) > 0 {
		return fmt.Sprintf("no suggestion\n%s\nTop candidates: %s", analysis, strings.Join(cands, ", "))
	}

	return fmt.Sprintf("couldn't produce a valid suggestion.\n%s", analysis)
}

// parseConstraints returns pattern (e.g. FLU__), present letters, excluded letters and per-position not-in maps
func parseConstraints(puzzle string) (string, []rune, []rune, []map[rune]bool) {
	lines := strings.Split(strings.TrimSpace(puzzle), "\n")
	pattern := []rune{'_', '_', '_', '_', '_'}
	presentMap := make(map[rune]bool)
	excludedMap := make(map[rune]bool)
	// per-position not-in constraints from 🟨 feedback
	notIn := make([]map[rune]bool, 5)
	for i := 0; i < 5; i++ {
		notIn[i] = make(map[rune]bool)
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var emojis []rune
		for _, r := range line {
			if r == '🟩' || r == '🟨' || r == '🟥' {
				emojis = append(emojis, r)
			}
		}
		if len(emojis) < 5 {
			for _, tok := range strings.Fields(line) {
				for _, r := range tok {
					if r == '🟩' || r == '🟨' || r == '🟥' {
						emojis = append(emojis, r)
					}
				}
				if len(emojis) >= 5 {
					break
				}
			}
		}
		word := extractFirstWord(line)
		if word == "" || len(emojis) < 5 {
			continue
		}
		for i := 0; i < 5; i++ {
			fb := emojis[i]
			ch := rune(word[i])
			switch fb {
			case '🟩':
				pattern[i] = ch
				presentMap[ch] = true
				if excludedMap[ch] {
					delete(excludedMap, ch)
				}
			case '🟨':
				presentMap[ch] = true
				notIn[i][ch] = true
				if excludedMap[ch] {
					delete(excludedMap, ch)
				}
			case '🟥':
				if !presentMap[ch] {
					excludedMap[ch] = true
				}
			}
		}
	}

	// remove any excluded that are present
	for ch := range presentMap {
		if excludedMap[ch] {
			delete(excludedMap, ch)
		}
	}
	pres := make([]rune, 0, len(presentMap))
	for ch := range presentMap {
		pres = append(pres, ch)
	}
	excl := make([]rune, 0, len(excludedMap))
	for ch := range excludedMap {
		excl = append(excl, ch)
	}
	// build pattern string
	pat := make([]string, 5)
	for i, r := range pattern {
		if r == '_' {
			pat[i] = "_"
		} else {
			pat[i] = strings.ToUpper(string(r))
		}
	}
	return strings.Join(pat, ""), pres, excl, notIn
}

// validateCandidate ensures the suggested word fits pattern, contains all present letters and excludes excluded letters
func validateCandidate(word, pattern string, present, excluded []rune, notIn []map[rune]bool) bool {
	if len(word) != 5 {
		return false
	}
	w := strings.ToUpper(word)
	// pattern
	for i := 0; i < 5; i++ {
		if pattern[i] != '_' && pattern[i] != w[i] {
			return false
		}
	}
	// per-position not-in checks (yellow constraints)
	if len(notIn) == 5 {
		for i := 0; i < 5; i++ {
			if notIn[i][rune(w[i])] {
				return false
			}
		}
	}
	// present letters
	for _, ch := range present {
		if !strings.ContainsRune(w, ch) {
			return false
		}
	}
	// excluded letters
	for _, ch := range excluded {
		if strings.ContainsRune(w, ch) {
			return false
		}
	}
	return true
}

var (
	wordListURL     = "https://raw.githubusercontent.com/MUSTAFA-A-KHAN/json-data-hub/refs/heads/main/words.json"
	cachedWordList  []string
	cachedWordMutex sync.Mutex
)

func loadWordList() ([]string, error) {
	cachedWordMutex.Lock()
	defer cachedWordMutex.Unlock()
	if len(cachedWordList) > 0 {
		return cachedWordList, nil
	}
	resp, err := http.Get(wordListURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var data struct {
		CommonWords []string `json:"commonWords"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	for _, w := range data.CommonWords {
		if len(w) == 5 {
			cachedWordList = append(cachedWordList, strings.ToUpper(strings.TrimSpace(w)))
		}
	}
	return cachedWordList, nil
}

// generateCandidates returns up to limit words from the wordlist that satisfy constraints
func generateCandidates(pattern string, present, excluded []rune, notIn []map[rune]bool, limit int) ([]string, error) {
	words, err := loadWordList()
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, limit)
	for _, w := range words {
		if validateCandidate(w, pattern, present, excluded, notIn) {
			out = append(out, w)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (t *TextTranslator) callLLM(prompt string) string {
	if OpenAIKey == "" {
		return llmErrorMessage
	}

	url := "https://api.llm7.io/v1/chat/completions"
	payload := map[string]interface{}{
		"model":       "openai/gpt-4o",
		"temperature": 0.2,
		"messages":    []map[string]string{{"role": "user", "content": prompt}},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return llmErrorMessage
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return llmErrorMessage
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", OpenAIKey))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return llmErrorMessage
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return llmErrorMessage
	}
	if resp.StatusCode != http.StatusOK {
		return llmErrorMessage
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return llmErrorMessage
	}

	if choices, ok := response["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					return content
				}
			}
		}
	}

	return llmErrorMessage
}

func extractFirstWord(text string) string {
	wordRe := regexp.MustCompile(`\b[a-zA-Z]{5}\b`)
	if match := wordRe.FindString(text); match != "" {
		return strings.ToUpper(match)
	}
	return ""
}

// AnalyzeEordle parses the puzzle input and returns the known pattern (with _ for unknowns),
// letters known to be present (from 🟨/🟩), and letters excluded (from 🟥).
func (t *TextTranslator) AnalyzeEordle(puzzle string) string {
	lines := strings.Split(strings.TrimSpace(puzzle), "\n")
	// pattern holds confirmed greens (underscore for unknowns)
	pattern := []rune{'_', '_', '_', '_', '_'}
	present := make(map[rune]bool)
	excluded := make(map[rune]bool)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// collect emoji feedback runes in order
		var emojis []rune
		for _, r := range line {
			if r == '🟩' || r == '🟨' || r == '🟥' {
				emojis = append(emojis, r)
			}
		}

		// fallback: collect emoji tokens from fields if not inline
		if len(emojis) < 5 {
			for _, tok := range strings.Fields(line) {
				for _, r := range tok {
					if r == '🟩' || r == '🟨' || r == '🟥' {
						emojis = append(emojis, r)
					}
				}
				if len(emojis) >= 5 {
					break
				}
			}
		}

		word := extractFirstWord(line)
		if word == "" || len(emojis) < 5 {
			continue
		}

		for i, fb := range emojis[:5] {
			ch := rune(word[i])
			switch fb {
			case '🟩':
				pattern[i] = ch
				present[ch] = true
				delete(excluded, ch)
			case '🟨':
				present[ch] = true
				delete(excluded, ch)
			case '🟥':
				// tentatively excluded; we'll remove if seen as present/green later
				if !present[ch] {
					excluded[ch] = true
				}
			}
		}
	}

	// ensure excluded doesn't contain any present letters
	for ch := range present {
		delete(excluded, ch)
	}

	// build pattern string
	pat := make([]string, 5)
	for i, r := range pattern {
		if r == '_' {
			pat[i] = "_"
		} else {
			pat[i] = string(r)
		}
	}

	// lists
	pres := make([]string, 0, len(present))
	for ch := range present {
		pres = append(pres, strings.ToUpper(string(ch)))
	}
	sort.Strings(pres)
	excl := make([]string, 0, len(excluded))
	for ch := range excluded {
		excl = append(excl, strings.ToUpper(string(ch)))
	}
	sort.Strings(excl)

	return fmt.Sprintf("Pattern: %s\nPresent: %s\nExcluded: %s",
		strings.Join(pat, ""),
		strings.Join(pres, ","),
		strings.Join(excl, ","))
}
