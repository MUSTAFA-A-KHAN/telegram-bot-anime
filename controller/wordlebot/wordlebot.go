package wordlebot

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/controller/wordlebot/image_generator"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/model"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/view"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"go.mongodb.org/mongo-driver/mongo"
)

// WordleStateDoc is the MongoDB-serializable version of WordleState
type WordleStateDoc struct {
	ChatID         int64    `bson:"_id"`
	Active         bool     `bson:"active"`
	Word           string   `bson:"word"`
	Guesses        []string `bson:"guesses"`
	Attempts       int      `bson:"attempts"`
	MaxAttempts    int      `bson:"max_attempts"`
	PendingNewGame bool     `bson:"pending_new_game"`
}

// saveWordleStateAsync asynchronously saves the Wordle state to MongoDB
func saveWordleStateAsync(chatID int64, state *WordleState) {
	state.RLock()
	doc := WordleStateDoc{
		ChatID:         chatID,
		Active:         state.Active,
		Word:           state.Word,
		Guesses:        state.Guesses,
		Attempts:       state.Attempts,
		MaxAttempts:    state.MaxAttempts,
		PendingNewGame: state.PendingNewGame,
	}
	state.RUnlock()

	go func() {
		client := repository.DbManager()
		if client != nil {
			repository.SaveGameState(client, "WordleStates", chatID, doc)
		}
	}()
}

// LoadSavedStates loads the persisted Wordle states from MongoDB into the memory map
func LoadSavedStates(client *mongo.Client) {
	var results []WordleStateDoc
	err := repository.LoadAllGameStates(client, "WordleStates", &results)
	if err != nil {
		log.Printf("Failed to load saved Wordle states: %v", err)
		return
	}

	wordleMutex.Lock()
	defer wordleMutex.Unlock()

	for _, doc := range results {
		ws := &WordleState{
			Active:         doc.Active,
			Word:           doc.Word,
			Guesses:        doc.Guesses,
			Attempts:       doc.Attempts,
			MaxAttempts:    doc.MaxAttempts,
			PendingNewGame: doc.PendingNewGame,
			CancelChan:     make(chan bool, 1),
		}
		wordleStates[doc.ChatID] = ws
	}
	log.Printf("Loaded %d active Wordle games from MongoDB", len(results))
}

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
func validateWordleGuess(guess, target string, colorConfig string) string {
	// ⚡ Bolt Optimization: Replacing map[rune]int with a fixed [256]int array
	// eliminates heap allocations for target letter counts.
	// Using a fixed string array for results avoids slice allocation.

	guess = strings.ToLower(guess)
	target = strings.ToLower(target)

	// Fast path for 5 letter ascii words
	if len(guess) == 5 && len(target) == 5 {
		var result [5]string
		var targetCounts [256]int

		missColor := "🟥"
		if colorConfig == "dark" {
			missColor = "⬛"
		} else if colorConfig == "light" {
			missColor = "⬜"
		}

		// First pass: count characters in target and default to miss
		for i := 0; i < 5; i++ {
			targetCounts[target[i]]++
			result[i] = missColor
		}

		// Mark Green
		for i := 0; i < 5; i++ {
			if guess[i] == target[i] {
				result[i] = "🟩"
				targetCounts[guess[i]]--
			}
		}

		// Second pass: check for correct letter in wrong place (Yellow)
		for i := 0; i < 5; i++ {
			if guess[i] != target[i] && targetCounts[guess[i]] > 0 {
				result[i] = "🟨"
				targetCounts[guess[i]]--
			}
		}

		return strings.Join(result[:], " ")
	}

	// Fallback for non 5-letter inputs
	result := make([]string, len(target))
	targetCounts := make(map[rune]int)

	missColor := "🟥"
	if colorConfig == "dark" {
		missColor = "⬛"
	} else if colorConfig == "light" {
		missColor = "⬜"
	}

	// First pass: count characters in target and check for exact matches (Green)
	for i, ch := range target {
		targetCounts[ch]++
		if i < len(result) {
			result[i] = missColor // Default to selected miss color
		}
	}

	// Mark Green
	limit := len(guess)
	if len(target) < limit {
		limit = len(target)
	}
	for i := 0; i < limit; i++ {
		if guess[i] == target[i] {
			result[i] = "🟩"
			targetCounts[rune(guess[i])]--
		}
	}

	// Second pass: check for correct letter in wrong place (Yellow)
	for i := 0; i < limit; i++ {
		if guess[i] != target[i] && targetCounts[rune(guess[i])] > 0 {
			result[i] = "🟨"
			targetCounts[rune(guess[i])]--
		}
	}

	return strings.Join(result, " ")
}

// getSuperscript returns the given number as a string of superscript characters
func getSuperscript(num int) string {
	// ⚡ Bolt Optimization: Replace map lookup and fmt.Sprintf with fixed array and strconv.Itoa
	// This reduces allocations and speeds up execution from ~1000ns to ~170ns.
	superscripts := [...]string{
		"⁰", "¹", "²", "³", "⁴",
		"⁵", "⁶", "⁷", "⁸", "⁹",
	}
	strNum := strconv.Itoa(num)
	var sb strings.Builder
	// Each superscript character is typically 3 bytes in UTF-8
	sb.Grow(len(strNum) * 3)
	for i := 0; i < len(strNum); i++ {
		c := strNum[i]
		if c >= '0' && c <= '9' {
			sb.WriteString(superscripts[c-'0'])
		} else {
			sb.WriteByte(c)
		}
	}
	return sb.String()
}

// buildWordleBoard generates the string representation of the current Wordle board
func buildWordleBoard(ws *WordleState, colorConfig string) string {
	var sb strings.Builder
	// Rough pre-allocation: guesses * (~30 bytes emojis + 7 bytes space/word + 2 bytes newline)
	sb.Grow(len(ws.Guesses) * 45)
	for i, guess := range ws.Guesses {
		feedback := validateWordleGuess(guess, ws.Word, colorConfig)

		// ⚡ Bolt Optimization: Inline ASCII uppercase conversion avoids strings.ToUpper allocation
		guessUpper := make([]byte, len(guess))
		for j := 0; j < len(guess); j++ {
			if guess[j] >= 'a' && guess[j] <= 'z' {
				guessUpper[j] = guess[j] - 32
			} else {
				guessUpper[j] = guess[j]
			}
		}

		if i > 10 {
			attemptNum := getSuperscript(i + 1)
			sb.WriteString(feedback)
			sb.WriteString("  ")
			sb.Write(guessUpper)
			sb.WriteString(" ")
			sb.WriteString(attemptNum)
			sb.WriteString("\n")
		} else {
			sb.WriteString(feedback)
			sb.WriteString("  ")
			sb.Write(guessUpper)
			sb.WriteString("\n")
		}
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
func HandleWordleCommand(bot *tgbotapi.BotAPI, chatID int64, username string, client *mongo.Client) {
	ws := GetOrCreateWordleState(chatID)

	ws.Lock()
	if ws.Active {
		if ws.PendingNewGame {
			ws.Unlock()
			saveWordleStateAsync(chatID, ws)
			return // Already a pending request
		}
		ws.PendingNewGame = true
		ws.CancelChan = make(chan bool, 1)
		ws.Unlock()
		saveWordleStateAsync(chatID, ws)

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
					saveWordleStateAsync(chatID, ws)
					return
				}
				ws.PendingNewGame = false
				ws.Active = true
				ws.Word = getRandomWordleWord()
				ws.Guesses = make([]string, 0)
				ws.Attempts = 0
				ws.Unlock()
				saveWordleStateAsync(chatID, ws)

				if err == nil {
					deleteMsg := tgbotapi.NewDeleteMessage(chatID, sentMsg.MessageID)
					bot.Send(deleteMsg)
				}

				buttons := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("Change Layout ⚙️", "setting_wordle_view_new"),
						tgbotapi.NewInlineKeyboardButtonData("Wordle Color 🎨", "setting_wordle_color_new"),
					),
				)

				settings := GetChatSettings(chatID, client)
				missEmoji := "🟥"
				if settings.WordleColor == "dark" {
					missEmoji = "⬛"
				} else if settings.WordleColor == "light" {
					missEmoji = "⬜"
				}

				msg := fmt.Sprintf("🐊 🖼 *Wordle started!* ✨\n\n• The word consists of 5 letters.\n• You have %d attempts.\n\n💡 Hints:\n🟩 Correct letter in the right spot\n🟨 Correct letter but in the wrong spot\n%s Letter is not in the word\n\nSend a 5-letter word to guess.", ws.MaxAttempts, missEmoji)
				view.SendMessageWithButtons(bot, chatID, msg, buttons)
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
	saveWordleStateAsync(chatID, ws)

	buttons := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Change Layout ⚙️", "setting_wordle_view_new"),
			tgbotapi.NewInlineKeyboardButtonData("Wordle Color 🎨", "setting_wordle_color_new"),
		),
	)

	settings := GetChatSettings(chatID, client)
	missEmoji := "🟥"
	if settings.WordleColor == "dark" {
		missEmoji = "⬛"
	} else if settings.WordleColor == "light" {
		missEmoji = "⬜"
	}

	msg := fmt.Sprintf("🐊 🖼 *Wordle started!* ✨\n\n• The word consists of 5 letters.\n• You have %d attempts.\n\n💡 Hints:\n🟩 Correct letter in the right spot\n🟨 Correct letter but in the wrong spot\n%s Letter is not in the word\n\nSend a 5-letter word to guess.", ws.MaxAttempts, missEmoji)
	view.SendMessageWithButtons(bot, chatID, msg, buttons)
}

// CancelPendingGame cancels an ongoing new game request
func CancelPendingGame(bot *tgbotapi.BotAPI, chatID int64, username string) bool {
	ws := GetOrCreateWordleState(chatID)
	ws.Lock()
	defer func() {
		ws.Unlock()
		saveWordleStateAsync(chatID, ws)
	}()

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
	defer func() {
		ws.Unlock()
		saveWordleStateAsync(chatID, ws)
	}()

	if !ws.Active {
		return
	}

	guess := strings.ToLower(strings.TrimSpace(text))

	// Check if it's exactly 5 bytes (ASCII check)
	if len(guess) != 5 {
		return // Not a valid guess format, ignore
	}

	// Ensure all characters are valid English alphabet letters
	for i := 0; i < len(guess); i++ {
		if guess[i] < 'a' || guess[i] > 'z' {
			return // Ignore non-alphabetical inputs completely
		}
	}

	wordsMutex.RLock()
	isValid := validWordleWords[guess]
	wordsMutex.RUnlock()

	if !isValid {
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ %s is not a valid word. Try again!", strings.ToUpper(guess)))
		msg.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}
		bot.Send(msg)
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
		msg := tgbotapi.NewMessage(chatID, "⚠️ This word was already guessed! Try again!")
		msg.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}
		bot.Send(msg)
		return
	}

	ws.Guesses = append(ws.Guesses, guess)
	ws.Attempts++

	settings := GetChatSettings(chatID, client)
	isImage := settings.WordleViewType == "image"
	var board string
	var imgData []byte

	if isImage {
		var err error
		imgData, err = image_generator.GenerateWordleImage(ws.Guesses, ws.Word, settings.WordleColor)
		if err != nil {
			log.Printf("Failed to generate wordle image: %v", err)
			isImage = false
			board = buildWordleBoard(ws, settings.WordleColor)
		}
	} else {
		board = buildWordleBoard(ws, settings.WordleColor)
	}

	if guess == ws.Word {
		ws.Active = false
		points := 25 - ws.Attempts + 1
		if points < 1 {
			points = 1 // Ensure minimum 1 point for winning
		}

		buttons := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Start new Wordle! 🟩🟨", "wordle_start"),
				tgbotapi.NewInlineKeyboardButtonData("Start Croc Game 🐊", "explain"),
			),
		)

		if isImage {
			meaning := model.GetWordMeaning(ws.Word)
			if meaning != "" {
				meaning = "\n\n```Meaning\n" + meaning + "\n```"
			}

			msg := fmt.Sprintf("🟩 🟩 🟩 🟩 🟩  %s   [+%d💎]\n🎉 [%s](tg://user?id=%d) guessed it in %d attempts!%s",
				strings.ToUpper(ws.Word), points, message.From.FirstName, message.From.ID, ws.Attempts, meaning)
			view.ReplyToMessageWithPhotoAndButtons(bot, message.MessageID, chatID, imgData, msg, buttons)
		} else {
			meaning := model.GetWordMeaning(ws.Word)
			if meaning != "" {
				meaning = "\n\n```Meaning\n" + meaning + "\n```"
			}

			msg := fmt.Sprintf("%s\n\n🟩 🟩 🟩 🟩 🟩  %s   [+%d💎]\n🎉 [%s](tg://user?id=%d) guessed it in %d attempts!%s",
				board, strings.ToUpper(ws.Word), points, message.From.FirstName, message.From.ID, ws.Attempts, meaning)
			view.ReplyToMessageWithButtons(bot, message.MessageID, chatID, msg, buttons)
		}

		go repository.InsertWordleDoc(message.From.ID, message.From.FirstName, chatID, client, "WordleEn", ws.Attempts)
	} else if ws.Attempts >= ws.MaxAttempts {
		ws.Active = false

		buttons := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Start new Wordle! 🟩🟨", "wordle_start"),
				tgbotapi.NewInlineKeyboardButtonData("Start Croc Game 🐊", "explain"),
			),
		)

		if isImage {
			meaning := model.GetWordMeaning(ws.Word)
			if meaning != "" {
				meaning = "\n\n```Meaning\n" + meaning + "\n```"
			}

			msg := fmt.Sprintf("❌ Out of attempts! The word was %s.%s", strings.ToUpper(ws.Word), meaning)
			view.ReplyToMessageWithPhotoAndButtons(bot, message.MessageID, chatID, imgData, msg, buttons)
		} else {
			meaning := model.GetWordMeaning(ws.Word)
			if meaning != "" {
				meaning = "\n\n```Meaning\n" + meaning + "\n```"
			}

			msg := fmt.Sprintf("%s\n\n❌ Out of attempts! The word was %s.%s", board, strings.ToUpper(ws.Word), meaning)
			view.ReplyToMessageWithButtons(bot, message.MessageID, chatID, msg, buttons)
		}
	} else {
		if isImage {
			view.ReplyToMessageWithPhotoAndButtons(bot, message.MessageID, chatID, imgData, "", tgbotapi.InlineKeyboardMarkup{})
		} else {
			view.ReplyToMessage(bot, message.MessageID, chatID, board)
		}
	}
}
