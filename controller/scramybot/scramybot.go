package scramybot

import (
	"bufio"
	"fmt"
	"html"
	"log"
	"math/rand"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/view"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"go.mongodb.org/mongo-driver/mongo"
)

// ScramyState holds the state for a Scramy game in a specific chat.
type ScramyState struct {
	sync.RWMutex
	Active         bool
	Letters        string
	FoundWords     []string
	UserWords      map[int][]string
	UserScores     map[int]int
	UserNames      map[int]string
	MaxWords       int
	PendingNewGame bool
	CancelChan     chan bool
}

var (
	scramyStates = make(map[int64]*ScramyState)
	scramyMutex  = &sync.RWMutex{}

	validWordsList []string
	validWordsMap  map[string]bool
	wordsLoaded    bool
	wordsMutex     sync.RWMutex
)

// GetOrCreateScramyState safely retrieves or creates a ScramyState for a chatID.
func GetOrCreateScramyState(chatID int64) *ScramyState {
	scramyMutex.Lock()
	defer scramyMutex.Unlock()
	if _, exists := scramyStates[chatID]; !exists {
		scramyStates[chatID] = &ScramyState{
			FoundWords: make([]string, 0),
			UserWords:  make(map[int][]string),
			UserScores: make(map[int]int),
			UserNames:  make(map[int]string),
			MaxWords:   10,
		}
	}
	return scramyStates[chatID]
}

// LoadScramyWords loads the 5-letter words from words.txt and allowed_words.txt
func LoadScramyWords() error {
	wordsMutex.Lock()
	defer wordsMutex.Unlock()

	if wordsLoaded {
		return nil
	}

	validWordsList = make([]string, 0)
	validWordsMap = make(map[string]bool)

	paths := []string{
		"controller/translator/scramy_words.txt",
		"../translator/scramy_words.txt",
		"../../controller/translator/scramy_words.txt",
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
		if len(word) >= 4 {
			validWordsMap[word] = true
			validWordsList = append(validWordsList, word)
		}
	}

	allowedPaths := []string{
		"controller/translator/scramy_allowed_words.txt",
		"../translator/scramy_allowed_words.txt",
		"../../controller/translator/scramy_allowed_words.txt",
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
			if len(word) >= 4 {
				validWordsMap[word] = true
			}
		}
	}

	wordsLoaded = true
	log.Printf("Loaded %d valid 4+ letter words for SCRAMY", len(validWordsList))
	return nil
}

// generateScramyLetters generates 15 random letters that can form at least 10 valid 5-letter words
func generateScramyLetters() string {
	wordsMutex.RLock()
	defer wordsMutex.RUnlock()

	vowels := []rune{'a', 'e', 'i', 'o', 'u'}
	consonants := []rune{'b', 'c', 'd', 'f', 'g', 'h', 'l', 'm', 'n', 'p', 'r', 's', 't', 'w', 'y'}

	for {
		var letters []rune
		letters = make([]rune, 15)
		for j := 0; j < 5; j++ { // 5 vowels
			letters[j] = vowels[rand.Intn(len(vowels))]
		}
		for j := 5; j < 15; j++ { // 10 consonants
			letters[j] = consonants[rand.Intn(len(consonants))]
		}

		// Because 15 letters almost covers the entire list of 20 unique vowels+consonants provided above,
		// duplicates are going to be very common, so we just allow duplicates in the 15-letter pool
		// rather than forcing all 15 to be strictly unique, to make it easier.

		rand.Shuffle(15, func(i, j int) {
			letters[i], letters[j] = letters[j], letters[i]
		})

		letterSet := make(map[rune]bool)
		for _, l := range letters {
			letterSet[l] = true
		}

		count := 0
		for _, w := range validWordsList {
			valid := true
			for _, ch := range w {
				if !letterSet[ch] {
					valid = false
					break
				}
			}
			if valid {
				count++
			}
		}

		if count >= 10 {
			// Convert to upper case separated by comma space
			str := ""
			for i, l := range letters {
				str += strings.ToUpper(string(l))
				if i < 14 {
					str += ", "
				}
			}
			return str
		}
	}
}

// getLetterString returns letters with spaces
func getLetterString(letters string) string {
	return strings.ReplaceAll(letters, ",", "")
}

func getPoints(wordsFoundCount int, wordLength int) int {
	bonus := 0
	if wordsFoundCount >= 1 && wordsFoundCount <= 4 {
		bonus = 0
	} else if wordsFoundCount >= 5 && wordsFoundCount <= 8 {
		bonus = 2
	} else if wordsFoundCount == 9 {
		bonus = 3
	} else if wordsFoundCount == 10 {
		bonus = 5
	}
	return wordLength + bonus
}

// IsScramyActive checks if the Scramy game is active for a given chat ID
func IsScramyActive(chatID int64) bool {
	ss := GetOrCreateScramyState(chatID)
	ss.RLock()
	defer ss.RUnlock()
	return ss.Active
}

// HandleScramyCommand starts a new Scramy game
func HandleScramyCommand(bot *tgbotapi.BotAPI, chatID int64, username string) {
	ss := GetOrCreateScramyState(chatID)

	ss.Lock()
	if ss.Active {
		if ss.PendingNewGame {
			ss.Unlock()
			return
		}
		ss.PendingNewGame = true
		ss.CancelChan = make(chan bool, 1)
		ss.Unlock()

		markup := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Cancel New Game ❌", "cancel_new_scramy"),
			),
		)
		alertMsg := fmt.Sprintf("⚠️ %s is trying to start a new Scramy game despite the current session going on!\n\nYou have 5 seconds to cancel the new game request.", username)
		sentMsg, err := bot.Send(tgbotapi.NewMessage(chatID, alertMsg))
		if err == nil {
			editMsg := tgbotapi.NewEditMessageReplyMarkup(chatID, sentMsg.MessageID, markup)
			bot.Send(editMsg)
		}

		go func() {
			select {
			case <-time.After(5 * time.Second):
				ss.Lock()
				if !ss.PendingNewGame {
					ss.Unlock()
					return
				}
				ss.PendingNewGame = false
				ss.Active = true
				ss.Letters = generateScramyLetters()
				ss.FoundWords = make([]string, 0)
				ss.UserWords = make(map[int][]string)
				ss.UserScores = make(map[int]int)
				ss.UserNames = make(map[int]string)
				ss.Unlock()

				if err == nil {
					deleteMsg := tgbotapi.NewDeleteMessage(chatID, sentMsg.MessageID)
					bot.Send(deleteMsg)
				}

				msg := fmt.Sprintf("📝 *WORD SCRAMBLE*\n\n🦴 Make words using these letters\n\n%s\n\n🔎 Words with 4 or more letters are accepted. Longer words give more points!\n\nTotal: 0/10", ss.Letters)
				view.SendMessage(bot, chatID, msg)
			case <-ss.CancelChan:
				if err == nil {
					deleteMsg := tgbotapi.NewDeleteMessage(chatID, sentMsg.MessageID)
					bot.Send(deleteMsg)
				}
			}
		}()
		return
	}

	ss.Active = true
	ss.Letters = generateScramyLetters()
	ss.FoundWords = make([]string, 0)
	ss.UserWords = make(map[int][]string)
	ss.UserScores = make(map[int]int)
	ss.UserNames = make(map[int]string)
	ss.Unlock()

	msg := fmt.Sprintf("📝 *WORD SCRAMBLE*\n\n🦴 Make words using these letters\n\n%s\n\n🔎 Words with 4 or more letters are accepted. Longer words give more points!\n\nTotal: 0/10", ss.Letters)
	view.SendMessage(bot, chatID, msg)
}

// CancelPendingGame cancels an ongoing new game request
func CancelPendingGame(bot *tgbotapi.BotAPI, chatID int64, username string) bool {
	ss := GetOrCreateScramyState(chatID)
	ss.Lock()
	defer ss.Unlock()

	if ss.PendingNewGame {
		ss.PendingNewGame = false
		select {
		case ss.CancelChan <- true:
		default:
		}
		view.SendMessage(bot, chatID, fmt.Sprintf("✅ %s cancelled the new Scramy game request. The current game will continue.", username))
		return true
	}
	return false
}

func isValidWordFromLetters(word string, letters string) bool {
	letterSet := make(map[rune]bool)
	letters = strings.ReplaceAll(letters, ",", "")
	letters = strings.ReplaceAll(letters, " ", "")
	letters = strings.ToLower(letters)

	for _, l := range letters {
		letterSet[l] = true
	}

	for _, ch := range word {
		if !letterSet[ch] {
			return false
		}
	}
	return true
}

func capitalizeWord(word string) string {
	if len(word) == 0 {
		return word
	}
	return strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
}

// HandleGuess processes a guess for a Scramy game
func HandleGuess(bot *tgbotapi.BotAPI, message *tgbotapi.Message, client *mongo.Client, chatID int64, text string) {
	ss := GetOrCreateScramyState(chatID)
	ss.Lock()
	defer ss.Unlock()

	if !ss.Active {
		return
	}

	guess := strings.ToLower(strings.TrimSpace(text))

	if len(guess) < 4 {
		return // Not a valid guess format, ignore
	}

	for i := 0; i < len(guess); i++ {
		if guess[i] < 'a' || guess[i] > 'z' {
			return // Ignore non-alphabetical inputs completely
		}
	}

	if !isValidWordFromLetters(guess, ss.Letters) {
		return
	}

	wordsMutex.RLock()
	isValid := validWordsMap[guess]
	wordsMutex.RUnlock()

	if !isValid {
		return
	}

	for _, g := range ss.FoundWords {
		if g == guess {
			return // Already guessed
		}
	}

	ss.FoundWords = append(ss.FoundWords, guess)
	ss.UserWords[message.From.ID] = append(ss.UserWords[message.From.ID], guess)

	points := getPoints(len(ss.FoundWords), len(guess))
	ss.UserScores[message.From.ID] += points
	ss.UserNames[message.From.ID] = message.From.FirstName

	letterStr := getLetterString(ss.Letters)

	if len(ss.FoundWords) >= ss.MaxWords {
		ss.Active = false

		msg := fmt.Sprintf("<b>%s</b> found \"<b>%s</b>\"\n+%d 💎\n\n🪟 %s\n\nTotal words found: %d/10\n\n◐ <b>Game over</b> ◑\n\n🏆 <b>Scores</b>\n\n",
			html.EscapeString(message.From.FirstName), html.EscapeString(capitalizeWord(guess)), points, letterStr, len(ss.FoundWords))

		type userScoreEntry struct {
			ID    int
			Name  string
			Score int
		}

		var scores []userScoreEntry
		for userID, score := range ss.UserScores {
			name := ss.UserNames[userID]
			if name == "" {
				name = fmt.Sprintf("User %d", userID) // Fallback if somehow not found
			}
			scores = append(scores, userScoreEntry{ID: userID, Name: name, Score: score})
			go repository.InsertWordleBonusDoc(userID, name, chatID, client, "ScramyEn", score) // reusing logic since it just inserts Score/Points
		}

		sort.SliceStable(scores, func(i, j int) bool {
			return scores[i].Score > scores[j].Score
		})

		if len(scores) > 0 {
			msg += fmt.Sprintf("<pre><code class=\"language-Winner\">%s - %d 💎</code></pre>\n", html.EscapeString(scores[0].Name), scores[0].Score)
			if len(scores) > 1 {
				msg += "👥 <b>Participants:</b>\n"
				for i := 1; i < len(scores); i++ {
					msg += fmt.Sprintf("<blockquote>%s - %d 💎</blockquote>\n", html.EscapeString(scores[i].Name), scores[i].Score)
				}
			}
		}

		buttons := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Start new Scramy! 📝", "scramy_start"),
				tgbotapi.NewInlineKeyboardButtonData("Start Wordle! 🟩🟨", "wordle_start"),
			),
		)

		view.ReplyToMessageWithButtonsHTML(bot, message.MessageID, chatID, msg, buttons)
	} else {
		msg := fmt.Sprintf("%s found \"%s\"\n+%d 💎\n\n🪟 %s\n\nTotal words found: %d/10",
			message.From.FirstName, capitalizeWord(guess), points, letterStr, len(ss.FoundWords))

		view.ReplyToMessage(bot, message.MessageID, chatID, msg)
	}
}
