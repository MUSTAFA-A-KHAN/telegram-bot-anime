package wordgridbot

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"strings"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/service"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/view"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"go.mongodb.org/mongo-driver/mongo"
)

func LoadWords() []string {
	content, err := os.ReadFile("controller/wordgridbot/lib/english_words_gt_10164946_family_safe_v3.txt")
	if err != nil {
		log.Printf("Error reading scramy_words.txt: %v", err)
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
	// Group words by length
	wordsByLength := make(map[int][]string)
	for _, word := range allWords {
		word = strings.ToUpper(word)
		len := len(word)
		if len >= 4 && len <= 8 {
			wordsByLength[len] = append(wordsByLength[len], word)
		}
	}

	// Pick words in increasing order of length
	// Limit words per length to ensure variety
	var result []string
	wordsPerLength := (n + 4) / 5 // Distribute words across 5 different lengths (4-8)

	for length := 4; length <= 8 && len(result) < n; length++ {
		words := wordsByLength[length]
		if len(words) == 0 {
			continue
		}

		// Shuffle words of this length
		rand.Shuffle(len(words), func(i, j int) {
			words[i], words[j] = words[j], words[i]
		})

		// Add limited words from this length
		wordsToAdd := wordsPerLength
		if len(result)+wordsToAdd > n {
			wordsToAdd = n - len(result)
		}
		if len(words) < wordsToAdd {
			wordsToAdd = len(words)
		}
		result = append(result, words[:wordsToAdd]...)
	}

	return result
}

func getCluesText(words []string, foundWords map[string]bool) string {
	var clues []string
	for _, w := range words {
		w = strings.ToUpper(w)
		if foundWords[w] {
			clues = append(clues, fmt.Sprintf("✅ %s", w))
		} else {
			dashes := strings.Repeat("-", len(w)-1)
			clues = append(clues, fmt.Sprintf("%c%s (%d)", w[0], dashes, len(w)))
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

func countRemainingWords(words []string, foundWords map[string]bool) int {
	remaining := 0
	for _, w := range words {
		if !foundWords[strings.ToUpper(w)] {
			remaining++
		}
	}
	return remaining
}

func sendWordFoundFeedback(bot *tgbotapi.BotAPI, chatID int64, userID int, userName string, foundWord string, pointsEarned int, remainingWords int, gridMessageID int) {
	remaining := "Remaining"
	if remainingWords == 1 {
		remaining = "Last One"
	} else if remainingWords == 0 {
		remaining = "🎉 LAST WORD! +15 BONUS POINTS!"
	}

	text := fmt.Sprintf("👤 *%s* found *%s*. +%d Points ✅\n\n📌 *%s*", userName, foundWord, pointsEarned, remaining)

	// Create link to grid message
	gridLink := fmt.Sprintf("https://t.me/c/%d/%d", -chatID-1000000000000, gridMessageID)

	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("🔙 Go To Grid", gridLink),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyToMessageID = gridMessageID
	msg.ReplyMarkup = inlineKeyboard
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)
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

	words := getRandomWords(10)
	grid, placedWords, positions := GenerateGrid(words, 10)

	state.Active = true
	state.Grid = grid
	state.Words = placedWords
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

	// Calculate remaining words BEFORE adding points
	remainingWords := countRemainingWords(state.Words, state.FoundWords)

	// Determine points: bonus if this is the last word
	pointsEarned := 10
	if remainingWords == 0 {
		pointsEarned = 25 // Bonus for finding the last word
	}

	state.UserScores[int64(message.From.ID)] += pointsEarned
	state.UserNames[int64(message.From.ID)] = message.From.FirstName

	go repository.InsertWordleBonusDoc(message.From.ID, message.From.FirstName, chatID, client, "WordGridPoints", pointsEarned)

	// Send feedback message
	go sendWordFoundFeedback(bot, chatID, message.From.ID, message.From.FirstName, guess, pointsEarned, remainingWords, state.MessageID)

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
	if !allFound {
		caption += GetLeaderboardText(state.UserScores, state.UserNames)
	}

	var newMsgText string
	var newMsgMarkup tgbotapi.InlineKeyboardMarkup

	if allFound {
		state.Active = false
		caption = fmt.Sprintf("🎉 *Word Grid Completed!*\n\nAll words have been found!\n\n🔍 *Words:*\n%s", getCluesText(state.Words, state.FoundWords))

		roundSummary := GetLeaderboardText(state.UserScores, state.UserNames)
		// Convert Markdown * to HTML <b> for roundSummary, since globalLeaderboard is HTML.
		roundSummaryHTML := strings.ReplaceAll(roundSummary, "*", "<b>")
		roundSummaryHTML = strings.ReplaceAll(roundSummaryHTML, "<b>Leaderboard:<b>", "<b>Leaderboard:</b>")
		globalLeaderboard := service.LeaderBoardList(client, "WordGridPoints", chatID)

		newMsgText = fmt.Sprintf("🎉 <b>Game Over Round Summary</b> 🎉\n%s\n\n%s", roundSummaryHTML, globalLeaderboard)
		newMsgMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Start New Grid 🔠", "wordgrid_start"),
			),
		)
	}

	go func(chat int64, msgID int, bts []byte, cap string, active bool, s *WordGridState, isGameOver bool, finalMsg string, finalMarkup tgbotapi.InlineKeyboardMarkup) {
		view.EditMessageMediaWithStyledButtons(bot.Token, chat, msgID, bts, "wordgrid.png", nil)

		// Send a new message to update the caption correctly as editMessageMedia doesn't always update caption easily if not provided in the media object JSON correctly
		editCaption := tgbotapi.NewEditMessageCaption(chat, msgID, cap)
		editCaption.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(editCaption)

		if isGameOver {
			newMsg := tgbotapi.NewMessage(chat, finalMsg)
			newMsg.ParseMode = tgbotapi.ModeHTML
			newMsg.ReplyMarkup = finalMarkup
			bot.Send(newMsg)
		}

		saveWordGridStateAsync(chat, s)
	}(chatID, state.MessageID, imgBytes, caption, state.Active, state, allFound, newMsgText, newMsgMarkup)
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
