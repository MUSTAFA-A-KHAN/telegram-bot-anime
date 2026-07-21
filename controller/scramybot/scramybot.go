package scramybot

import (
	"bufio"
	"fmt"
	"html"
	"log"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/view"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"go.mongodb.org/mongo-driver/mongo"
)

// ScramyStateDoc is the MongoDB-serializable version of ScramyState
type ScramyStateDoc struct {
	ChatID         int64               `bson:"_id"`
	Active         bool                `bson:"active"`
	Letters        string              `bson:"letters"`
	FoundWords     []string            `bson:"found_words"`
	UserWords      map[string][]string `bson:"user_words"`
	UserScores     map[string]int      `bson:"user_scores"`
	UserNames      map[string]string   `bson:"user_names"`
	MaxWords       int                 `bson:"max_words"`
	PendingNewGame bool                `bson:"pending_new_game"`
}

// saveScramyStateAsync asynchronously saves the Scramy state to MongoDB
func saveScramyStateAsync(chatID int64, state *ScramyState) {
	state.RLock()

	// Convert int keys to string keys for MongoDB BSON compatibility
	userWordsStr := make(map[string][]string)
	for k, v := range state.UserWords {
		userWordsStr[strconv.Itoa(k)] = v
	}

	userScoresStr := make(map[string]int)
	for k, v := range state.UserScores {
		userScoresStr[strconv.Itoa(k)] = v
	}

	userNamesStr := make(map[string]string)
	for k, v := range state.UserNames {
		userNamesStr[strconv.Itoa(k)] = v
	}

	doc := ScramyStateDoc{
		ChatID:         chatID,
		Active:         state.Active,
		Letters:        state.Letters,
		FoundWords:     state.FoundWords,
		UserWords:      userWordsStr,
		UserScores:     userScoresStr,
		UserNames:      userNamesStr,
		MaxWords:       state.MaxWords,
		PendingNewGame: state.PendingNewGame,
	}
	state.RUnlock()

	go func() {
		client := repository.DbManager()
		if client != nil {
			repository.SaveGameState(client, "ScramyStates", chatID, doc)
		}
	}()
}

// LoadSavedStates loads the persisted Scramy states from MongoDB into the memory map
func LoadSavedStates(client *mongo.Client) {
	var results []ScramyStateDoc
	err := repository.LoadAllGameStates(client, "ScramyStates", &results)
	if err != nil {
		log.Printf("Failed to load saved Scramy states: %v", err)
		return
	}

	scramyMutex.Lock()
	defer scramyMutex.Unlock()

	for _, doc := range results {
		ss := &ScramyState{
			Active:         doc.Active,
			Letters:        doc.Letters,
			MaxWords:       doc.MaxWords,
			PendingNewGame: doc.PendingNewGame,
			FoundWords:     doc.FoundWords,
			UserWords:      make(map[int][]string),
			UserScores:     make(map[int]int),
			UserNames:      make(map[int]string),
			CancelChan:     make(chan bool, 1),
		}

		for kStr, v := range doc.UserWords {
			var k int
			fmt.Sscanf(kStr, "%d", &k)
			ss.UserWords[k] = v
		}

		for kStr, v := range doc.UserScores {
			var k int
			fmt.Sscanf(kStr, "%d", &k)
			ss.UserScores[k] = v
		}

		for kStr, v := range doc.UserNames {
			var k int
			fmt.Sscanf(kStr, "%d", &k)
			ss.UserNames[k] = v
		}

		scramyStates[doc.ChatID] = ss
	}
	log.Printf("Loaded %d active Scramy games from MongoDB", len(results))
}

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

		// Bolt Optimization: Replace map[rune]bool with a boolean array for faster O(1) lookups
		// and use a byte-indexed loop to avoid rune decoding overhead since all words are ASCII.
		var letterSet [256]bool
		for _, l := range letters {
			if l < 256 {
				letterSet[l] = true
			}
		}

		count := 0
		for _, w := range validWordsList {
			valid := true
			for i := 0; i < len(w); i++ {
				if !letterSet[w[i]] {
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
func getLetterString(letters string, isSquared bool) string {
	var formatted strings.Builder
	for _, ch := range letters {
		if !isSquared {
			if ch != ',' && ch != ' ' {
				formatted.WriteRune(ch)
				formatted.WriteRune(' ')
			}
			continue
		}
		switch ch {
		case 'A', 'a':
			formatted.WriteString("🄰 ")
		case 'B', 'b':
			formatted.WriteString("🄱 ")
		case 'C', 'c':
			formatted.WriteString("🄲 ")
		case 'D', 'd':
			formatted.WriteString("🄳 ")
		case 'E', 'e':
			formatted.WriteString("🄴 ")
		case 'F', 'f':
			formatted.WriteString("🄵 ")
		case 'G', 'g':
			formatted.WriteString("🄶 ")
		case 'H', 'h':
			formatted.WriteString("🄷 ")
		case 'I', 'i':
			formatted.WriteString("🄸 ")
		case 'J', 'j':
			formatted.WriteString("🄹 ")
		case 'K', 'k':
			formatted.WriteString("🄺 ")
		case 'L', 'l':
			formatted.WriteString("🄻 ")
		case 'M', 'm':
			formatted.WriteString("🄼 ")
		case 'N', 'n':
			formatted.WriteString("🄽 ")
		case 'O', 'o':
			formatted.WriteString("🄾 ")
		case 'P', 'p':
			formatted.WriteString("🄿 ")
		case 'Q', 'q':
			formatted.WriteString("🅀 ")
		case 'R', 'r':
			formatted.WriteString("🅁 ")
		case 'S', 's':
			formatted.WriteString("🅂 ")
		case 'T', 't':
			formatted.WriteString("🅃 ")
		case 'U', 'u':
			formatted.WriteString("🅄 ")
		case 'V', 'v':
			formatted.WriteString("🅅 ")
		case 'W', 'w':
			formatted.WriteString("🅆 ")
		case 'X', 'x':
			formatted.WriteString("🅇 ")
		case 'Y', 'y':
			formatted.WriteString("🅈 ")
		case 'Z', 'z':
			formatted.WriteString("🅉 ")
		case ',', ' ':
			// Skip commas and spaces from the original string
		default:
			formatted.WriteRune(ch)
		}
	}
	return strings.TrimSpace(formatted.String())
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
			saveScramyStateAsync(chatID, ss)
			return
		}
		ss.PendingNewGame = true
		ss.CancelChan = make(chan bool, 1)
		ss.Unlock()
		saveScramyStateAsync(chatID, ss)

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
					saveScramyStateAsync(chatID, ss)
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
				saveScramyStateAsync(chatID, ss)

				if err == nil {
					deleteMsg := tgbotapi.NewDeleteMessage(chatID, sentMsg.MessageID)
					bot.Send(deleteMsg)
				}

				settings := GetChatSettings(chatID, nil)
				isSquared := settings.ScramyLetterView == "squared"
				isH1 := settings.ScramyLetterView == "h1"

				buttons := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("Change Layout ⚙️", "setting_scramy_letters_new"),
					),
				)
				if isH1 {
					topText := "📝 WORD SCRAMBLE\n\n🦴 Make words using these letters\n\n"
					bottomText := "\n\n🔎 Words with 4 or more letters are accepted. Longer words give more points!\n\nTotal: 0/10"
					letters := getLetterString(ss.Letters, false)
					view.SendScramyRichMessage(bot.Token, chatID, topText, letters, bottomText, buttons)
				} else {
					msg := fmt.Sprintf("📝 *WORD SCRAMBLE*\n\n🦴 Make words using these letters\n\n%s\n\n🔎 Words with 4 or more letters are accepted. Longer words give more points!\n\nTotal: 0/10", getLetterString(ss.Letters, isSquared))
					view.SendMessageWithButtons(bot, chatID, msg, buttons)
				}
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
	saveScramyStateAsync(chatID, ss)

	settings := GetChatSettings(chatID, nil)
	isSquared := settings.ScramyLetterView == "squared"
	isH1 := settings.ScramyLetterView == "h1"

	buttons := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Change Layout ⚙️", "setting_scramy_letters_new"),
		),
	)
	if isH1 {
		topText := "📝 WORD SCRAMBLE\n\n🦴 Make words using these letters\n\n"
		bottomText := "\n\n🔎 Words with 4 or more letters are accepted. Longer words give more points!\n\nTotal: 0/10"
		letters := getLetterString(ss.Letters, false)
		view.SendScramyRichMessage(bot.Token, chatID, topText, letters, bottomText, buttons)
	} else {
		msg := fmt.Sprintf("📝 *WORD SCRAMBLE*\n\n🦴 Make words using these letters\n\n%s\n\n🔎 Words with 4 or more letters are accepted. Longer words give more points!\n\nTotal: 0/10", getLetterString(ss.Letters, isSquared))
		view.SendMessageWithButtons(bot, chatID, msg, buttons)
	}
}

// CancelPendingGame cancels an ongoing new game request
func CancelPendingGame(bot *tgbotapi.BotAPI, chatID int64, username string) bool {
	ss := GetOrCreateScramyState(chatID)
	ss.Lock()
	defer func() {
		ss.Unlock()
		saveScramyStateAsync(chatID, ss)
	}()

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
	// Bolt Optimization: Use a 256 boolean array instead of map for O(1) ASCII lookup,
	// and single loop mapping over raw characters avoiding string allocation overheads.
	var letterSet [256]bool
	for i := 0; i < len(letters); i++ {
		c := letters[i]
		if c >= 'A' && c <= 'Z' {
			letterSet[c+32] = true
		} else if c >= 'a' && c <= 'z' {
			letterSet[c] = true
		}
	}

	for i := 0; i < len(word); i++ {
		if !letterSet[word[i]] {
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
	defer func() {
		ss.Unlock()
		saveScramyStateAsync(chatID, ss)
	}()

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

	settings := GetChatSettings(chatID, client)
	isSquared := settings.ScramyLetterView == "squared"
	isH1 := settings.ScramyLetterView == "h1"
	letterStr := getLetterString(ss.Letters, isSquared)

	if len(ss.FoundWords) >= ss.MaxWords {
		ss.Active = false

		var msg string
		if isH1 {
			msg = fmt.Sprintf("<b>%s</b> found \"<b>%s</b>\"\n+%d 💎\n\nTotal words found: %d/10\n\n◐ <b>Game over</b> ◑\n\n",
				html.EscapeString(message.From.FirstName), html.EscapeString(capitalizeWord(guess)), points, len(ss.FoundWords))
		} else {
			msg = fmt.Sprintf("<b>%s</b> found \"<b>%s</b>\"\n+%d 💎\n\n🪟 %s\n\nTotal words found: %d/10\n\n◐ <b>Game over</b> ◑\n\n",
				html.EscapeString(message.From.FirstName), html.EscapeString(capitalizeWord(guess)), points, letterStr, len(ss.FoundWords))
		}

		longestWord := ""
		longestWordUserID := 0
		for userID, words := range ss.UserWords {
			for _, w := range words {
				if len(w) > len(longestWord) {
					longestWord = w
					longestWordUserID = userID
				}
			}
		}

		if longestWord != "" {
			longestWordBonus := 10
			ss.UserScores[longestWordUserID] += longestWordBonus
			longestWordUserName := ss.UserNames[longestWordUserID]
			if longestWordUserName == "" {
				longestWordUserName = fmt.Sprintf("User %d", longestWordUserID)
			}
			msg += fmt.Sprintf("🏆 <b>Largest Word Found</b>\n%s found \"<b>%s</b>\" and received +%d 💎\n\n", html.EscapeString(longestWordUserName), html.EscapeString(capitalizeWord(longestWord)), longestWordBonus)
		}

		msg += "🏆 <b>Scores</b>\n\n"

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
				msg += "<blockquote expandable>\n"
				for i := 1; i < len(scores); i++ {
					msg += fmt.Sprintf("%s - %d 💎\n", html.EscapeString(scores[i].Name), scores[i].Score)
				}
				msg += "</blockquote>\n"
			}
		}

		buttons := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Start new Scramy! 📝", "scramy_start"),
				tgbotapi.NewInlineKeyboardButtonData("Start Wordle! 🟩🟨", "wordle_start"),
			),
		)

		if isH1 {
			// Rich text cannot easily be a reply with the current OvyFlash types without constructing a custom payload,
			// and our helper SendScramyRichMessage doesn't currently support replying. We will send it as a normal message for H1.
			topText := msg
			bottomText := ""
			lettersStr := getLetterString(ss.Letters, false)
			view.SendScramyRichMessage(bot.Token, chatID, topText, lettersStr, bottomText, buttons)
		} else {
			view.ReplyToMessageWithButtonsHTML(bot, message.MessageID, chatID, msg, buttons)
		}
	} else {
		if isH1 {
			topText := fmt.Sprintf("%s found \"%s\"\n+%d 💎\n\n", message.From.FirstName, capitalizeWord(guess), points)
			bottomText := fmt.Sprintf("\n\nTotal words found: %d/10", len(ss.FoundWords))
			lettersStr := getLetterString(ss.Letters, false)
			view.SendScramyRichMessage(bot.Token, chatID, topText, lettersStr, bottomText, tgbotapi.InlineKeyboardMarkup{})
		} else {
			msg := fmt.Sprintf("%s found \"%s\"\n+%d 💎\n\n🪟 %s\n\nTotal words found: %d/10",
				message.From.FirstName, capitalizeWord(guess), points, letterStr, len(ss.FoundWords))

			view.ReplyToMessage(bot, message.MessageID, chatID, msg)
		}
	}
}
