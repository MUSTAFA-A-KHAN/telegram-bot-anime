package geographybot

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/view"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"go.mongodb.org/mongo-driver/mongo"
)

// GeographyStateDoc is the MongoDB-serializable version of GeographyState
type GeographyStateDoc struct {
	ChatID         int64    `bson:"_id"`
	Active         bool     `bson:"active"`
	QuestionType   string   `bson:"question_type"`
	TargetCountry  string   `bson:"target_country"`
	TargetAnswer   string   `bson:"target_answer"`
	Options        []string `bson:"options"`
	Attempts       int      `bson:"attempts"`
	MaxAttempts    int      `bson:"max_attempts"`
	PendingNewGame bool     `bson:"pending_new_game"`
}

// GeographyState holds the state for a Geography game in a specific chat.
type GeographyState struct {
	sync.RWMutex
	Active         bool
	QuestionType   string // "capital", "flag", "region"
	TargetCountry  string
	TargetAnswer   string
	Options        []string
	Attempts       int
	MaxAttempts    int
	PendingNewGame bool
	CancelChan     chan bool
}

var (
	geographyStates = make(map[int64]*GeographyState)
	geographyMutex  = &sync.RWMutex{}
)

// saveGeographyStateAsync asynchronously saves the Geography state to MongoDB
func saveGeographyStateAsync(chatID int64, state *GeographyState) {
	state.RLock()
	doc := GeographyStateDoc{
		ChatID:         chatID,
		Active:         state.Active,
		QuestionType:   state.QuestionType,
		TargetCountry:  state.TargetCountry,
		TargetAnswer:   state.TargetAnswer,
		Options:        state.Options,
		Attempts:       state.Attempts,
		MaxAttempts:    state.MaxAttempts,
		PendingNewGame: state.PendingNewGame,
	}
	state.RUnlock()

	go func() {
		client := repository.DbManager()
		if client != nil {
			repository.SaveGameState(client, "GeographyStates", chatID, doc)
		}
	}()
}

// LoadSavedStates loads the persisted Geography states from MongoDB into the memory map
func LoadSavedStates(client *mongo.Client) {
	var results []GeographyStateDoc
	err := repository.LoadAllGameStates(client, "GeographyStates", &results)
	if err != nil {
		log.Printf("Failed to load saved Geography states: %v", err)
		return
	}

	geographyMutex.Lock()
	defer geographyMutex.Unlock()

	for _, doc := range results {
		gs := &GeographyState{
			Active:         doc.Active,
			QuestionType:   doc.QuestionType,
			TargetCountry:  doc.TargetCountry,
			TargetAnswer:   doc.TargetAnswer,
			Options:        doc.Options,
			Attempts:       doc.Attempts,
			MaxAttempts:    doc.MaxAttempts,
			PendingNewGame: doc.PendingNewGame,
			CancelChan:     make(chan bool, 1),
		}
		geographyStates[doc.ChatID] = gs
	}
	log.Printf("Loaded %d active Geography games from MongoDB", len(results))
}

// Country represents a country's data
type Country struct {
	Name    string `json:"name"`
	Capital string `json:"capital"`
	Region  string `json:"region"`
	Flag    string `json:"flag"`
}

var (
	countryData []Country
	dataLoaded  bool
)

// LoadGeographyData loads the static country JSON dataset into memory
func LoadGeographyData() error {
	path := filepath.Join("controller", "geographybot", "countries.json")
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open countries.json: %w", err)
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(&countryData); err != nil {
		return fmt.Errorf("failed to decode countries.json: %w", err)
	}

	// Filter out any entries that might be incomplete
	var validCountries []Country
	for _, c := range countryData {
		if c.Name != "" && c.Capital != "" && c.Flag != "" {
			validCountries = append(validCountries, c)
		}
	}
	countryData = validCountries
	dataLoaded = true

	log.Printf("Loaded %d valid countries for Geography mode", len(countryData))
	return nil
}

// IsGeographyActive returns true if there is an active game in the chat
func IsGeographyActive(chatID int64) bool {
	geographyMutex.RLock()
	defer geographyMutex.RUnlock()
	state, exists := geographyStates[chatID]
	if !exists {
		return false
	}
	state.RLock()
	defer state.RUnlock()
	return state.Active
}

// HandleGeographyCommand handles the /geography command
func HandleGeographyCommand(bot *tgbotapi.BotAPI, chatID int64, userName string, client *mongo.Client) {
	geographyMutex.Lock()
	state, exists := geographyStates[chatID]
	if !exists {
		state = &GeographyState{
			CancelChan: make(chan bool, 1),
		}
		geographyStates[chatID] = state
	}
	geographyMutex.Unlock()

	state.Lock()
	if state.Active {
		state.Unlock()
		view.SendMessage(bot, chatID, "A Geography game is already active! Answer the current question or wait for it to finish.")
		return
	}
	state.PendingNewGame = false
	state.Unlock()

	startNewRound(bot, chatID, client)
}

// startNewRound picks a random question and sends it to the chat
func startNewRound(bot *tgbotapi.BotAPI, chatID int64, client *mongo.Client) {
	if !dataLoaded || len(countryData) < 4 {
		view.SendMessage(bot, chatID, "Geography data is currently unavailable.")
		return
	}

	rand.Seed(time.Now().UnixNano())
	targetIndex := rand.Intn(len(countryData))
	target := countryData[targetIndex]

	questionTypes := []string{"capital", "flag", "region"}
	qType := questionTypes[rand.Intn(len(questionTypes))]

	var answer string
	var question string
	var options []string

	switch qType {
	case "capital":
		answer = target.Capital
		question = fmt.Sprintf("🌎 *Geography Mode*\n\nWhat is the capital of *%s %s*?", target.Flag, target.Name)
	case "flag":
		answer = target.Name
		question = fmt.Sprintf("🌎 *Geography Mode*\n\nWhich country does this flag belong to: *%s*?", target.Flag)
	case "region":
		answer = target.Region
		question = fmt.Sprintf("🌎 *Geography Mode*\n\nWhich region is *%s %s* located in?", target.Flag, target.Name)
	}

	// Generate 3 random wrong options
	wrongOptions := make([]string, 0, 3)
	for len(wrongOptions) < 3 {
		randIdx := rand.Intn(len(countryData))
		if randIdx == targetIndex {
			continue
		}
		var wOpt string
		switch qType {
		case "capital":
			wOpt = countryData[randIdx].Capital
		case "flag":
			wOpt = countryData[randIdx].Name
		case "region":
			wOpt = countryData[randIdx].Region
		}

		if wOpt == answer {
			continue
		}

		// ensure unique options
		isDuplicate := false
		for _, o := range wrongOptions {
			if o == wOpt {
				isDuplicate = true
				break
			}
		}
		if !isDuplicate {
			wrongOptions = append(wrongOptions, wOpt)
		}
	}

	options = append(options, answer)
	options = append(options, wrongOptions...)

	// Shuffle options
	rand.Shuffle(len(options), func(i, j int) {
		options[i], options[j] = options[j], options[i]
	})

	geographyMutex.RLock()
	state := geographyStates[chatID]
	geographyMutex.RUnlock()

	state.Lock()
	state.Active = true
	state.TargetCountry = target.Name
	state.TargetAnswer = answer
	state.QuestionType = qType
	state.Options = options
	state.Attempts = 0
	state.MaxAttempts = 1 // 1 attempt for MCQ
	state.Unlock()

	saveGeographyStateAsync(chatID, state)

	// Send Question with Inline Buttons
	var keyboard [][]tgbotapi.InlineKeyboardButton
	// 2 buttons per row
	for i := 0; i < len(options); i += 2 {
		var row []tgbotapi.InlineKeyboardButton
		btn1 := tgbotapi.NewInlineKeyboardButtonData(options[i], "geo_ans_"+options[i])
		row = append(row, btn1)
		if i+1 < len(options) {
			btn2 := tgbotapi.NewInlineKeyboardButtonData(options[i+1], "geo_ans_"+options[i+1])
			row = append(row, btn2)
		}
		keyboard = append(keyboard, row)
	}

	markup := tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	view.SendMessageWithButtons(bot, chatID, question, markup)
}

// HandleGeographyCallback handles the inline button callbacks for MCQ
func HandleGeographyCallback(bot *tgbotapi.BotAPI, chatID int64, userID int, userName string, data string, callbackQueryID string, messageID int, client *mongo.Client) {
	geographyMutex.RLock()
	state, exists := geographyStates[chatID]
	geographyMutex.RUnlock()

	if !exists {
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callbackQueryID, "Game not found."))
		return
	}

	state.Lock()
	if !state.Active {
		state.Unlock()
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callbackQueryID, "Game is not active."))
		return
	}

	userAnswer := strings.TrimPrefix(data, "geo_ans_")
	correctAnswer := state.TargetAnswer

	// Update UI to remove buttons
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, fmt.Sprintf("Question answered by %s.", userName))
	bot.Send(editMsg)

	if strings.EqualFold(userAnswer, correctAnswer) {
		// Correct
		state.Active = false
		state.PendingNewGame = true
		state.Unlock()

		points := 5 // standard geography points
		if client != nil {
			repository.InsertWordleBonusDoc(userID, userName, chatID, client, "GeographyPoints", points)
		}

		successMsg := fmt.Sprintf("✅ *Correct, %s!*\n\nThe answer was *%s*.\nYou earned %d Geography points! 🌍\n\n_Use /geography to play again._", userName, correctAnswer, points)
		view.SendMessage(bot, chatID, successMsg)

	} else {
		// Wrong
		state.Active = false
		state.PendingNewGame = false
		state.Unlock()

		failMsg := fmt.Sprintf("❌ *Incorrect, %s!*\n\nThe correct answer was *%s*.\n\n_Use /geography to play again._", userName, correctAnswer)
		view.SendMessage(bot, chatID, failMsg)
	}

	saveGeographyStateAsync(chatID, state)
}

// HandleGuess handles exact text match guesses
func HandleGuess(bot *tgbotapi.BotAPI, message *tgbotapi.Message, client *mongo.Client, chatID int64, text string) {
	geographyMutex.RLock()
	state, exists := geographyStates[chatID]
	geographyMutex.RUnlock()

	if !exists {
		return
	}

	state.Lock()
	if !state.Active {
		state.Unlock()
		return
	}

	correctAnswer := state.TargetAnswer
	if strings.EqualFold(strings.TrimSpace(text), correctAnswer) {
		state.Active = false
		state.PendingNewGame = true
		state.Unlock()

		points := 5
		if client != nil {
			repository.InsertWordleBonusDoc(message.From.ID, message.From.FirstName, chatID, client, "GeographyPoints", points)
		}

		successMsg := fmt.Sprintf("✅ *Correct, %s!*\n\nThe answer was *%s*.\nYou earned %d Geography points! 🌍\n\n_Use /geography to play again._", message.From.FirstName, correctAnswer, points)
		view.SendMessage(bot, chatID, successMsg)

		saveGeographyStateAsync(chatID, state)
	} else {
		state.Unlock()
	}
}
