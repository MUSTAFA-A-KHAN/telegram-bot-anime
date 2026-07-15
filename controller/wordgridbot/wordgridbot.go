package wordgridbot

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"strings"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/view"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"go.mongodb.org/mongo-driver/mongo"
)

func LoadWords() []string {
	content, err := os.ReadFile("controller/translator/words.txt")
	if err != nil {
		log.Printf("Error reading words.txt: %v", err)
		return []string{"HELLO", "WORLD", "GAMES", "TELEGRAM"}
	}
	words := strings.Split(string(content), "\n")
	var valid []string
	for _, w := range words {
		w = strings.TrimSpace(w)
		if len(w) >= 4 && len(w) <= 8 {
			valid = append(valid, strings.ToUpper(w))
		}
	}
	return valid
}

var allWords = LoadWords()

func getRandomWords(n int) []string {
	words := make([]string, len(allWords))
	copy(words, allWords)
	rand.Shuffle(len(words), func(i, j int) {
		words[i], words[j] = words[j], words[i]
	})
	if len(words) > n {
		return words[:n]
	}
	return words
}

func getCluesText(words []string, foundWords map[string]bool) string {
	var clues []string
	for _, w := range words {
		w = strings.ToUpper(w)
		if foundWords[w] {
			clues = append(clues, fmt.Sprintf("✅ %s", w))
		} else {
			blanks := strings.Repeat("_ ", len(w)-1)
			clues = append(clues, fmt.Sprintf("✅ %c %s", w[0], strings.TrimSpace(blanks)))
		}
	}
	return strings.Join(clues, "\n")
}

func GetLeaderboardText(userScores map[int64]int, userNames map[int64]string) string {
	if len(userScores) == 0 {
		return ""
	}

	type scoreEntry struct {
		Name  string
		Score int
	}

	var scores []scoreEntry
	for id, score := range userScores {
		scores = append(scores, scoreEntry{Name: userNames[id], Score: score})
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	var sb strings.Builder
	sb.WriteString("\n\n🏆 *Leaderboard:*\n")
	for i, entry := range scores {
		medal := "🏅"
		if i == 0 {
			medal = "🥇"
		} else if i == 1 {
			medal = "🥈"
		} else if i == 2 {
			medal = "🥉"
		}
		sb.WriteString(fmt.Sprintf("%s %s - %d pts\n", medal, entry.Name, entry.Score))
	}
	return sb.String()
}

func StartWordGridGame(bot *tgbotapi.BotAPI, chatID int64, client *mongo.Client) {
	wordGridMutex.Lock()
	state, exists := wordGridStates[chatID]
	if !exists {
		state = &WordGridState{
			CancelChan: make(chan bool, 1),
		}
		wordGridStates[chatID] = state
	}
	wordGridMutex.Unlock()

	state.Lock()
	if state.Active {
		state.Unlock()
		msg := tgbotapi.NewMessage(chatID, "A Word Grid game is already active in this chat! Find the remaining words or /cancelwordgrid to start a new one.")
		bot.Send(msg)
		return
	}

	words := getRandomWords(14)
	grid, positions := GenerateGrid(words, 12)

	state.Active = true
	state.Grid = grid
	state.Words = words
	state.WordPositions = positions
	state.FoundWords = make(map[string]bool)
	state.UserScores = make(map[int64]int)
	state.UserNames = make(map[int64]string)
	state.Unlock()

	imgBytes, _ := GenerateGridImage(grid, positions, state.FoundWords)

	caption := fmt.Sprintf("🔠 *Word Grid (Hard Mode)*\n\n🔍 *Find These Words:*\n\n%s\n\n♨️ _Find Words, Gain Score Points & Improve Your Leaderboard Rank._", getCluesText(words, state.FoundWords))

	msg := tgbotapi.NewPhotoUpload(chatID, tgbotapi.FileBytes{Name: "wordgrid.png", Bytes: imgBytes})
	msg.Caption = caption
	msg.ParseMode = tgbotapi.ModeMarkdown

	sentMsg, err := bot.Send(msg)
	if err == nil {
		state.Lock()
		state.MessageID = sentMsg.MessageID
		state.Unlock()
		saveWordGridStateAsync(chatID, state)
	}
}

func IsWordGridActive(chatID int64) bool {
	wordGridMutex.RLock()
	defer wordGridMutex.RUnlock()
	if state, exists := wordGridStates[chatID]; exists {
		state.RLock()
		defer state.RUnlock()
		return state.Active
	}
	return false
}

func HandleGuess(bot *tgbotapi.BotAPI, message *tgbotapi.Message, client *mongo.Client, chatID int64, text string) {
	wordGridMutex.RLock()
	state, exists := wordGridStates[chatID]
	wordGridMutex.RUnlock()

	if !exists {
		return
	}

	state.Lock()
	defer state.Unlock()

	if !state.Active {
		return
	}

	guess := strings.ToUpper(strings.TrimSpace(text))

	isWordInList := false
	for _, w := range state.Words {
		if strings.ToUpper(w) == guess {
			isWordInList = true
			break
		}
	}

	if !isWordInList {
		return
	}

	if state.FoundWords[guess] {
		// Word already found
		return
	}

	// Word found!
	state.FoundWords[guess] = true
	state.UserScores[int64(message.From.ID)] += 10
	state.UserNames[int64(message.From.ID)] = message.From.FirstName

	go repository.InsertWordleBonusDoc(message.From.ID, message.From.FirstName, chatID, client, "WordGridPoints", 10)

	// Check if game is over
	allFound := true
	for _, w := range state.Words {
		if !state.FoundWords[strings.ToUpper(w)] {
			allFound = false
			break
		}
	}

	imgBytes, _ := GenerateGridImage(state.Grid, state.WordPositions, state.FoundWords)

	caption := fmt.Sprintf("🔠 *Word Grid (Hard Mode)*\n\n🔍 *Find These Words:*\n\n%s\n\n♨️ _Find Words, Gain Score Points & Improve Your Leaderboard Rank._", getCluesText(state.Words, state.FoundWords))
	caption += GetLeaderboardText(state.UserScores, state.UserNames)

	if allFound {
		state.Active = false
		caption = fmt.Sprintf("🎉 *Word Grid Completed!*\n\nAll words have been found!\n\n%s", GetLeaderboardText(state.UserScores, state.UserNames))
	}

	go func(chat int64, msgID int, bts []byte, cap string, active bool, s *WordGridState) {
		view.EditMessageMediaWithStyledButtons(bot.Token, chat, msgID, bts, "wordgrid.png", nil)

		// Send a new message to update the caption correctly as editMessageMedia doesn't always update caption easily if not provided in the media object JSON correctly
		editCaption := tgbotapi.NewEditMessageCaption(chat, msgID, cap)
		editCaption.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(editCaption)

		saveWordGridStateAsync(chat, s)
	}(chatID, state.MessageID, imgBytes, caption, state.Active, state)
}

func HandleCancelWordGrid(bot *tgbotapi.BotAPI, chatID int64) {
	wordGridMutex.RLock()
	state, exists := wordGridStates[chatID]
	wordGridMutex.RUnlock()

	if !exists {
		msg := tgbotapi.NewMessage(chatID, "No active Word Grid game to cancel.")
		bot.Send(msg)
		return
	}

	state.Lock()
	if !state.Active {
		state.Unlock()
		msg := tgbotapi.NewMessage(chatID, "No active Word Grid game to cancel.")
		bot.Send(msg)
		return
	}

	state.Active = false
	state.Unlock()

	msg := tgbotapi.NewMessage(chatID, "🛑 Word Grid game has been cancelled.")
	bot.Send(msg)

	saveWordGridStateAsync(chatID, state)
}
