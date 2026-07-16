package wordgridbot

import (
	"fmt"
	"log"
	"sync"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
	"go.mongodb.org/mongo-driver/mongo"
)

type WordPosition struct {
	StartRow int `bson:"start_row"`
	StartCol int `bson:"start_col"`
	EndRow   int `bson:"end_row"`
	EndCol   int `bson:"end_col"`
}

// WordGridState holds the in-memory state of an active Word Grid game
type WordGridState struct {
	sync.RWMutex
	Active        bool
	Grid          [][]string
	Words         []string
	WordPositions map[string]WordPosition
	FoundWords    map[string]bool
	UserScores    map[int64]int
	UserNames     map[int64]string
	MessageID     int
	CancelChan    chan bool
}

// WordGridStateDoc is the MongoDB-serializable version of WordGridState
type WordGridStateDoc struct {
	ChatID        int64                   `bson:"_id"`
	Active        bool                    `bson:"active"`
	Grid          [][]string              `bson:"grid"`
	Words         []string                `bson:"words"`
	WordPositions map[string]WordPosition `bson:"word_positions"`
	FoundWords    map[string]bool         `bson:"found_words"`
	UserScores    map[string]int          `bson:"user_scores"`
	UserNames     map[string]string       `bson:"user_names"`
	MessageID     int                     `bson:"message_id"`
}

var (
	wordGridStates = make(map[int64]*WordGridState)
	wordGridMutex  sync.RWMutex
)

// saveWordGridStateAsync asynchronously saves the Word Grid state to MongoDB
func saveWordGridStateAsync(chatID int64, state *WordGridState) {
	state.RLock()

	userScoresStr := make(map[string]int)
	for k, v := range state.UserScores {
		userScoresStr[fmt.Sprintf("%d", k)] = v
	}

	userNamesStr := make(map[string]string)
	for k, v := range state.UserNames {
		userNamesStr[fmt.Sprintf("%d", k)] = v
	}

	doc := WordGridStateDoc{
		ChatID:        chatID,
		Active:        state.Active,
		Grid:          state.Grid,
		Words:         state.Words,
		WordPositions: state.WordPositions,
		FoundWords:    state.FoundWords,
		UserScores:    userScoresStr,
		UserNames:     userNamesStr,
		MessageID:     state.MessageID,
	}
	state.RUnlock()

	go func() {
		client := repository.DbManager()
		if client != nil {
			repository.SaveGameState(client, "WordGridStates", chatID, doc)
		}
	}()
}

// LoadSavedStates loads the persisted Word Grid states from MongoDB into the memory map
func LoadSavedStates(client *mongo.Client) {
	var results []WordGridStateDoc
	err := repository.LoadAllGameStates(client, "WordGridStates", &results)
	if err != nil {
		log.Printf("Failed to load saved Word Grid states: %v", err)
		return
	}

	wordGridMutex.Lock()
	defer wordGridMutex.Unlock()

	for _, doc := range results {
		ws := &WordGridState{
			Active:        doc.Active,
			Grid:          doc.Grid,
			Words:         doc.Words,
			WordPositions: doc.WordPositions,
			FoundWords:    doc.FoundWords,
			UserScores:    make(map[int64]int),
			UserNames:     make(map[int64]string),
			MessageID:     doc.MessageID,
			CancelChan:    make(chan bool, 1),
		}

		for kStr, v := range doc.UserScores {
			var k int64
			fmt.Sscanf(kStr, "%d", &k)
			ws.UserScores[k] = v
		}
		for kStr, v := range doc.UserNames {
			var k int64
			fmt.Sscanf(kStr, "%d", &k)
			ws.UserNames[k] = v
		}

		wordGridStates[doc.ChatID] = ws
	}
	log.Printf("Loaded %d Word Grid states", len(results))
}
