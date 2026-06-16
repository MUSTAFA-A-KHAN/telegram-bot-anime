package animebot

import (
	"encoding/json"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/view"
	"github.com/agnivade/levenshtein"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"go.mongodb.org/mongo-driver/mongo"
)

type AnimeData struct {
	Emotes  string   `json:"emotes"`
	Answers []string `json:"answers"`
}

var (
	animeList []AnimeData
	mu        sync.Mutex
	rng       *rand.Rand
)

type GameState struct {
	Active   bool
	Question AnimeData
}

var activeGames = make(map[int64]*GameState)
var activeGamesMu sync.Mutex

func init() {
	source := rand.NewSource(time.Now().UnixNano())
	rng = rand.New(source)
}

func LoadAnimeData() {
	data, err := os.ReadFile("controller/animebot/anime.json")
	if err != nil {
		// handle err
		return
	}
	json.Unmarshal(data, &animeList)
}

func HandleAnimeCommand(bot *tgbotapi.BotAPI, chatID int64, client *mongo.Client) {
	if len(animeList) == 0 {
		LoadAnimeData()
	}
	if len(animeList) == 0 {
		return
	}

	activeGamesMu.Lock()
	defer activeGamesMu.Unlock()

	question := animeList[rng.Intn(len(animeList))]
	activeGames[chatID] = &GameState{
		Active:   true,
		Question: question,
	}

	text := "<b>Anime Emote Guess!</b>\n\nGuess the anime or character from these emotes:\n" + question.Emotes + "\n\nType your guess!"
	view.SendMessagehtml(bot, chatID, text)
}

func IsAnimeActive(chatID int64) bool {
	activeGamesMu.Lock()
	defer activeGamesMu.Unlock()
	state, exists := activeGames[chatID]
	return exists && state.Active
}

func CancelAnime(chatID int64) bool {
	activeGamesMu.Lock()
	defer activeGamesMu.Unlock()

	if state, exists := activeGames[chatID]; exists && state.Active {
		delete(activeGames, chatID)
		return true
	}
	return false
}

func HandleGuess(bot *tgbotapi.BotAPI, message *tgbotapi.Message, client *mongo.Client, chatID int64, text string) {
	if message == nil {
		return
	}

	activeGamesMu.Lock()
	state, exists := activeGames[chatID]
	if !exists || !state.Active {
		activeGamesMu.Unlock()
		return
	}
	question := state.Question
	activeGamesMu.Unlock()

	correct := false
	bestAnswer := question.Answers[0]
	for _, ans := range question.Answers {
		if checkAnswerFuzzy(text, ans) {
			correct = true
			bestAnswer = ans
			break
		}
	}

	if correct {
		activeGamesMu.Lock()
		delete(activeGames, chatID)
		activeGamesMu.Unlock()

		points := 10
		go repository.InsertWordleBonusDoc(message.From.ID, message.From.FirstName, chatID, client, "AnimePoints", points)

		view.SendMessagehtml(bot, chatID, "🎉 Correct! It was <b>"+bestAnswer+"</b>!\nYou earned "+strconv.Itoa(points)+" points!")
	} else {
		// Not correct, maybe close but give a generic message
		// view.SendMessagehtml(bot, chatID, "❌ Nope, try again!")
	}
}

func checkAnswerFuzzy(guess, answer string) bool {
	g := strings.ToLower(strings.TrimSpace(guess))
	a := strings.ToLower(strings.TrimSpace(answer))

	// if exactly same
	if g == a {
		return true
	}

	dist := levenshtein.ComputeDistance(g, a)
	// allow 2 edits for longer words
	maxDist := len(a) / 4
	if maxDist < 1 {
		maxDist = 1
	}
	if maxDist > 3 {
		maxDist = 3
	}
	return dist <= maxDist
}
