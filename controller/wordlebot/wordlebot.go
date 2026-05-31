package wordlebot

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/view"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"go.mongodb.org/mongo-driver/mongo"
)

// WordleState holds the state for a Wordle game in a specific chat.
type WordleState struct {
	sync.RWMutex
	Active         bool
	Word           string
	Guesses        []string
	Attempts       int
	MaxAttempts    int
	PendingNewGame bool
	CancelChan     chan bool
}

var (
	// wordleStates holds the Wordle game state per chat
	wordleStates = make(map[int64]*WordleState)
	wordleMutex  = &sync.RWMutex{}

	validWordleWords = make(map[string]bool)
	wordleWordList   = make([]string, 0)
	wordsLoaded      bool
	wordsMutex       sync.RWMutex
)

// GetOrCreateWordleState safely retrieves or creates a WordleState for a chatID.
func GetOrCreateWordleState(chatID int64) *WordleState {
	wordleMutex.Lock()
	defer wordleMutex.Unlock()
	if _, exists := wordleStates[chatID]; !exists {
		wordleStates[chatID] = &WordleState{
			Guesses:     make([]string, 0),
			MaxAttempts: 15,
		}
	}
	return wordleStates[chatID]
}

// LoadWordleWords loads the 5-letter words from words.txt
func LoadWordleWords() error {
	wordsMutex.Lock()
	defer wordsMutex.Unlock()

	if wordsLoaded {
		return nil
	}

	// Try to find words.txt starting from current dir or going up
	paths := []string{
		"controller/translator/words.txt",
		"../translator/words.txt",
		"../../controller/translator/words.txt",
	}

	var file *os.File
	var err error
	for _, p := range paths {
		file, err = os.Open(p)
		if err == nil {
			break
		}
	}

	if err != nil {
		return fmt.Errorf("could not find words.txt: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		word := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if len(word) == 5 {
			validWordleWords[word] = true
			wordleWordList = append(wordleWordList, word)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Load allowed_words.txt for validation
	allowedPaths := []string{
		"controller/translator/allowed_words.txt",
		"../translator/allowed_words.txt",
		"../../controller/translator/allowed_words.txt",
	}

	var allowedFile *os.File
	var allowedErr error
	for _, p := range allowedPaths {
		allowedFile, allowedErr = os.Open(p)
		if allowedErr == nil {
			break
		}
	}

	if allowedErr == nil {
		defer allowedFile.Close()
		allowedScanner := bufio.NewScanner(allowedFile)
		for allowedScanner.Scan() {
			word := strings.TrimSpace(strings.ToLower(allowedScanner.Text()))
			if len(word) == 5 {
				validWordleWords[word] = true
			}
		}
		if err := allowedScanner.Err(); err != nil {
			log.Printf("Error scanning allowed_words.txt: %v", err)
		}
	} else {
		log.Printf("Could not find allowed_words.txt, fallback to words.txt only: %v", allowedErr)
	}

	wordsLoaded = true
	log.Printf("Loaded %d Wordle target words and %d valid guesses", len(wordleWordList), len(validWordleWords))
	return nil
}

// getRandomWordleWord returns a random 5-letter word
func getRandomWordleWord() string {
	wordsMutex.RLock()
	defer wordsMutex.RUnlock()
	if len(wordleWordList) == 0 {
		return "apple" // fallback
	}
	return wordleWordList[rand.Intn(len(wordleWordList))]
}

// validateWordleGuess compares a guess against the target word and returns colored emojis
func validateWordleGuess(guess, target string) string {
	guess = strings.ToLower(guess)
	target = strings.ToLower(target)

	result := make([]string, 5)
	targetCounts := make(map[rune]int)

	// First pass: count characters in target and check for exact matches (Green)
	for i, ch := range target {
		targetCounts[ch]++
		result[i] = "🟥" // Default to Red
	}

	// Mark Green
	for i := 0; i < 5; i++ {
		if guess[i] == target[i] {
			result[i] = "🟩"
			targetCounts[rune(guess[i])]--
		}
	}

	// Second pass: check for correct letter in wrong place (Yellow)
	for i := 0; i < 5; i++ {
		if guess[i] != target[i] && targetCounts[rune(guess[i])] > 0 {
			result[i] = "🟨"
			targetCounts[rune(guess[i])]--
		}
	}

	return strings.Join(result, " ")
}

// buildWordleBoard generates the string representation of the current Wordle board
func buildWordleBoard(ws *WordleState) string {
	var sb strings.Builder
	for _, guess := range ws.Guesses {
		feedback := validateWordleGuess(guess, ws.Word)
		sb.WriteString(fmt.Sprintf("%s  %s\n", feedback, strings.ToUpper(guess)))
	}
	return sb.String()
}

// IsWordleActive checks if the Wordle game is active for a given chat ID
func IsWordleActive(chatID int64) bool {
	ws := GetOrCreateWordleState(chatID)
	ws.RLock()
	defer ws.RUnlock()
	return ws.Active
}

// HandleWordleCommand starts a new Wordle game
func HandleWordleCommand(bot *tgbotapi.BotAPI, chatID int64, username string) {
	ws := GetOrCreateWordleState(chatID)

	ws.Lock()
	if ws.Active {
		if ws.PendingNewGame {
			ws.Unlock()
			return // Already a pending request
		}
		ws.PendingNewGame = true
		ws.CancelChan = make(chan bool, 1)
		ws.Unlock()

		markup := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Cancel New Game ❌", "cancel_new_wordle"),
			),
		)
		alertMsg := fmt.Sprintf("⚠️ %s is trying to start a new Wordle game despite the current session going on!\n\nYou have 5 seconds to cancel the new game request.", username)
		sentMsg, err := bot.Send(tgbotapi.NewMessage(chatID, alertMsg))
		if err == nil {
			editMsg := tgbotapi.NewEditMessageReplyMarkup(chatID, sentMsg.MessageID, markup)
			bot.Send(editMsg)
		}

		go func() {
			select {
			case <-time.After(5 * time.Second):
				ws.Lock()
				if !ws.PendingNewGame {
					// It was cancelled
					ws.Unlock()
					return
				}
				ws.PendingNewGame = false
				ws.Active = true
				ws.Word = getRandomWordleWord()
				ws.Guesses = make([]string, 0)
				ws.Attempts = 0
				ws.Unlock()

				if err == nil {
					deleteMsg := tgbotapi.NewDeleteMessage(chatID, sentMsg.MessageID)
					bot.Send(deleteMsg)
				}

				msg := fmt.Sprintf("🐊 🖼 *Wordle started!* ✨\n\n• The word consists of 5 letters.\n• You have %d attempts.\n\n💡 Hints:\n🟩 Correct letter in the right spot\n🟨 Correct letter but in the wrong spot\n🟥 Letter is not in the word\n\nSend a 5-letter word to guess.", ws.MaxAttempts)
				view.SendMessage(bot, chatID, msg)
			case <-ws.CancelChan:
				// Cancelled by a user
				if err == nil {
					deleteMsg := tgbotapi.NewDeleteMessage(chatID, sentMsg.MessageID)
					bot.Send(deleteMsg)
				}
			}
		}()
		return
	}

	ws.Active = true
	ws.Word = getRandomWordleWord()
	ws.Guesses = make([]string, 0)
	ws.Attempts = 0
	ws.Unlock()

	msg := fmt.Sprintf("🐊 🖼 *Wordle started!* ✨\n\n• The word consists of 5 letters.\n• You have %d attempts.\n\n💡 Hints:\n🟩 Correct letter in the right spot\n🟨 Correct letter but in the wrong spot\n🟥 Letter is not in the word\n\nSend a 5-letter word to guess.", ws.MaxAttempts)
	view.SendMessage(bot, chatID, msg)
}

// CancelPendingGame cancels an ongoing new game request
func CancelPendingGame(bot *tgbotapi.BotAPI, chatID int64, username string) bool {
	ws := GetOrCreateWordleState(chatID)
	ws.Lock()
	defer ws.Unlock()

	if ws.PendingNewGame {
		ws.PendingNewGame = false
		select {
		case ws.CancelChan <- true:
		default:
		}
		view.SendMessage(bot, chatID, fmt.Sprintf("✅ %s cancelled the new Wordle game request. The current game will continue.", username))
		return true
	}
	return false
}

// HandleGuess processes a guess for a Wordle game
func HandleGuess(bot *tgbotapi.BotAPI, message *tgbotapi.Message, client *mongo.Client, chatID int64, text string) {
	ws := GetOrCreateWordleState(chatID)
	ws.Lock()
	defer ws.Unlock()

	if !ws.Active {
		return
	}

	guess := strings.ToLower(strings.TrimSpace(text))
	if len(guess) != 5 {
		return // Not a valid guess format, ignore
	}

	wordsMutex.RLock()
	isValid := validWordleWords[guess]
	wordsMutex.RUnlock()

	if !isValid {
		view.SendMessage(bot, chatID, fmt.Sprintf("❌ %s is not a valid word.", strings.ToUpper(guess)))
		return
	}

	alreadyGuessed := false
	for _, g := range ws.Guesses {
		if g == guess {
			alreadyGuessed = true
			break
		}
	}

	if alreadyGuessed {
		view.SendMessage(bot, chatID, "⚠️ This word was already guessed!")
		return
	}

	ws.Guesses = append(ws.Guesses, guess)
	ws.Attempts++

	board := buildWordleBoard(ws)

	if guess == ws.Word {
		ws.Active = false
		points := 25 - ws.Attempts + 1
		if points < 1 {
			points = 1 // Ensure minimum 1 point for winning
		}
		msg := fmt.Sprintf("%s\n\n🟩 🟩 🟩 🟩 🟩  %s   [+%d💎]\n🎉 [%s](tg://user?id=%d) guessed it in %d attempts!",
			board, strings.ToUpper(ws.Word), points, message.From.FirstName, message.From.ID, ws.Attempts)

		buttons := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Start new Wordle! 🟩🟨", "wordle_start"),
				tgbotapi.NewInlineKeyboardButtonData("Start Croc Game 🐊", "explain"),
			),
		)

		view.ReplyToMessageWithButtons(bot, message.MessageID, chatID, msg, buttons)
		go repository.InsertWordleDoc(message.From.ID, message.From.FirstName, chatID, client, "WordleEn", ws.Attempts)
	} else if ws.Attempts >= ws.MaxAttempts {
		ws.Active = false
		msg := fmt.Sprintf("%s\n\n❌ Out of attempts! The word was %s.", board, strings.ToUpper(ws.Word))

		buttons := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Start new Wordle! 🟩🟨", "wordle_start"),
				tgbotapi.NewInlineKeyboardButtonData("Start Croc Game 🐊", "explain"),
			),
		)

		view.ReplyToMessageWithButtons(bot, message.MessageID, chatID, msg, buttons)
	} else {
		view.ReplyToMessage(bot, message.MessageID, chatID, board)
	}
}
