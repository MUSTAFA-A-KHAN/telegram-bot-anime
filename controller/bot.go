package controller

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/model"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/service"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/view"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// ChatState holds the state for a specific chat, including the current word and user explaining it.
type ChatState struct {
	sync.RWMutex
	Word          string
	User          int
	LeadTimestamp time.Time
	Leader        string
}

var (
	// chatStates is a map that holds the state for each chat, identified by chat ID.
	chatStates = make(map[int64]*ChatState)
	// stateMutex ensures safe access to the chatStates map.
	stateMutex = &sync.RWMutex{}
)

// getOrCreateChatState safely retrieves or creates a ChatState for a chatID.
func getOrCreateChatState(chatID int64) *ChatState {
	stateMutex.Lock()
	defer stateMutex.Unlock()
	if _, exists := chatStates[chatID]; !exists {
		chatStates[chatID] = &ChatState{}
	}
	return chatStates[chatID]
}

// createSingleButtonKeyboard creates an inline keyboard markup with a single button.
func createSingleButtonKeyboard(text, data string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(text, data),
		),
	)
}

// isLeaderActive checks if the current leader is active within the given duration.
func (cs *ChatState) isLeaderActive(duration time.Duration) bool {
	cs.RLock()
	defer cs.RUnlock()
	return cs.User != 0 && time.Since(cs.LeadTimestamp) < duration
}

// reset resets the chat state.
func (cs *ChatState) reset() {
	cs.Lock()
	defer cs.Unlock()
	cs.Word = ""
	cs.User = 0
	cs.LeadTimestamp = time.Time{}
	cs.Leader = ""
}

// StartBot initializes and starts the bot
func StartBot(token string) error {
	go startHTTPServer() //start http server with go routine
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return err
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		return err
	}

	for update := range updates {
		if update.Message != nil {
			go handleMessage(bot, update.Message)
		} else if update.CallbackQuery != nil {
			go handleCallbackQuery(bot, update.CallbackQuery)
		}
	}

	return nil
}

// handleMessage processes incoming messages and handles commands and guesses.
func handleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	adminID := int64(1006461736)

	chatState := getOrCreateChatState(chatID)

	log.Printf("[%s] %s", message.From.UserName, message.Text)

	switch message.Command() {
	case "getButton":
		Announcement := strings.Split(message.Text, "  ")
		if len(Announcement) > 1 {
			parts := strings.Split(Announcement[2], " ")
			if len(parts) > 2 {
				url := parts[0]
				messageText := strings.Join(parts[1:], " ")
				button := tgbotapi.NewInlineKeyboardButtonURL(messageText, url)
				view.SendMessageWithKeyboardButton(bot, message.Chat.ID, Announcement[1], button)
			}
		}
	case "start":
		view.SendMessage(bot, message.Chat.ID, "Welcome! Use /word to start a game.")
	case "stats":
		result := service.LeaderBoardList("CrocEn")
		view.SendMessage(bot, message.Chat.ID, result)
	case "leaderstats":
		result := service.LeaderBoardList("CrocEnLeader")
		view.SendMessage(bot, message.Chat.ID, result)
	case "report":
		if len(message.Text) > 7 {
			reportMessage := message.Text[7:]
			adminMessage := fmt.Sprintf("Report from @%s (%d):\n%s", message.From.UserName, message.From.ID, reportMessage)
			view.SendMessage(bot, adminID, adminMessage)
			view.SendMessage(bot, chatID, "Your report has been submitted. Thank you!")
		} else {
			view.SendMessage(bot, chatID, "Please provide a message with your report. Usage: /report [your message]")
		}
	case "word":
		chatState.RLock()
		wordEmpty := chatState.Word == ""
		leadExpired := time.Since(chatState.LeadTimestamp) >= 120*time.Second
		chatState.RUnlock()

		if wordEmpty || leadExpired {
			word, err := model.GetRandomWord()
			if err != nil {
				view.SendMessage(bot, message.Chat.ID, "Failed to fetch a word.")
				return
			}

			buttons := createSingleButtonKeyboard(" 🗣️ Explain ", "explain")

			chatState.Lock()
			chatState.Word = word
			chatState.User = 0
			chatState.Leader = ""
			chatState.Unlock()

			view.SendSticker(bot, chatID, "CAACAgUAAxkBAAEwCnNnYW-OkgV7Odt9osVwoBSzLC6vsAACMhMAAj45CFdCstMoIYiPfjYE")
			view.SendMessageWithButtons(bot, message.Chat.ID, "The word is ready! Click 'Explain' to explain the word.", buttons)
		} else {
			view.SendMessage(bot, message.Chat.ID, "A game is on.")
		}

	case "reveal":
		chatState.RLock()
		word := chatState.Word
		leadTime := chatState.LeadTimestamp
		chatState.RUnlock()

		if time.Since(leadTime) >= 600*time.Second {
			buttons := createSingleButtonKeyboard(" 🗣️ Explain ", "explain")
			view.SendMessageWithButtons(bot, message.Chat.ID, fmt.Sprintf("The word was:%s", word), buttons)

			chatState.reset()
		} else {
			view.SendMessage(bot, message.Chat.ID, "Wait for ten minutes")
		}
	default:
		chatState.RLock()
		word := chatState.Word
		user := chatState.User
		chatState.RUnlock()

		if user != 0 && service.NormalizeAndCompare(message.Text, word) && message.From.ID != user {
			buttons := createSingleButtonKeyboard("🌟 Claim Leadership 🙋", "explain")
			view.SendMessageWithButtons(bot, message.Chat.ID, fmt.Sprintf("Congratulations! %s guessed the word %s.\n /word", message.From.FirstName, word), buttons)

			client := repository.DbManager()
			repository.InsertDoc(message.From.ID, message.From.FirstName, message.Chat.ID, client, "CrocEn")
			repository.InsertDoc(user, chatState.Leader, message.Chat.ID, client, "CrocEnLeader")

			chatState.reset()
		}
	}
}

// handleCallbackQuery processes incoming callback queries and handles the "explain" action.
func handleCallbackQuery(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID
	chatState := getOrCreateChatState(chatID)

	switch callback.Data {
	case "explain":
		chatState.Lock()
		if chatState.User != callback.From.ID && chatState.User != 0 && time.Since(chatState.LeadTimestamp) < 120*time.Second {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, fmt.Sprintf("%s is already explaining the word. %s", chatState.User, callback.From.UserName)))
			chatState.Unlock()
			return
		}
		if chatState.User == callback.From.ID {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, chatState.Word))
			chatState.Unlock()
			return
		}
		if chatState.User == 0 || (time.Since(chatState.LeadTimestamp) >= 120*time.Second && chatState.User != callback.From.ID) {
			word, err := model.GetRandomWord()
			if err != nil {
				chatState.Unlock()
				return
			}
			buttons := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("See word 👀", "explain"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Next ⏭️", "next"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Changed my mind ❌", "droplead"),
				),
			)
			chatState.Word = word
			view.SendMessageWithButtons(bot, callback.Message.Chat.ID, fmt.Sprintf(" [%s](tg://user?id=%d)is explaining the word!", callback.From.FirstName, callback.From.ID), buttons)
		}
		chatState.User = callback.From.ID
		chatState.Leader = callback.From.FirstName
		chatState.LeadTimestamp = time.Now()
		chatState.Unlock()
		bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, chatState.Word))

	case "next":
		chatState.Lock()
		if chatState.User != callback.From.ID && chatState.User != 0 {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, fmt.Sprintf("%s is already explaining the word. %s", chatState.User, callback.From.UserName)))
			chatState.Unlock()
			return
		}
		if chatState.User == 0 {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, fmt.Sprintf("%s Please click on see word/claim Leadership", callback.From.FirstName)))
			chatState.Unlock()
			return
		}
		chatState.User = callback.From.ID
		chatState.Leader = callback.From.FirstName
		chatState.Unlock()
		chatState.Word, _ = model.GetRandomWord()
		bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, chatState.Word))

	case "droplead":
		chatState.Lock()
		if chatState.User != callback.From.ID {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, "You are not the leader, so you cannot drop the lead!"))
			chatState.Unlock()
			return
		}
		buttons := createSingleButtonKeyboard("🌟 Claim Leadership 🙋", "explain")
		view.SendMessageWithButtons(bot, callback.Message.Chat.ID, fmt.Sprintf("%s refused to lead -> %s \n", callback.From.FirstName, chatState.Word), buttons)
		chatState.reset()
		chatState.Unlock()

	default:
		chatState.RLock()
		word := chatState.Word
		chatState.RUnlock()
		if service.NormalizeAndCompare(callback.Message.Text, word) {
			buttons := createSingleButtonKeyboard("🌟 Claim Leadership 🙋", "explain")
			view.SendMessageWithButtons(bot, callback.Message.Chat.ID, fmt.Sprintf("Congratulations! %s guessed the word correctly.", callback.From.FirstName), buttons)
			chatState.Lock()
			chatState.reset()
			chatState.Unlock()
		}
	}
	bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
}

// startHTTPServer starts a simple HTTP server for health checks
func startHTTPServer() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Bot is running!")
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
