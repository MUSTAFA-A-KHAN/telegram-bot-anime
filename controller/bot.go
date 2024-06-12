package controller

import (
	"fmt"
	"log"
	"sync"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/model"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/service"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/view"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// ChatState holds the state for a specific chat
type ChatState struct {
	sync.RWMutex
	Word string
	User string
}

var (
	chatStates = make(map[int64]*ChatState)
	stateMutex = &sync.RWMutex{}
)

// StartBot initializes and starts the bot
func StartBot(token string) error {
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
			handleMessage(bot, update.Message)
		} else if update.CallbackQuery != nil {
			handleCallbackQuery(bot, update.CallbackQuery)
		}
	}

	return nil
}

func handleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID

	stateMutex.Lock()
	if _, exists := chatStates[chatID]; !exists {
		chatStates[chatID] = &ChatState{}
	}
	chatState := chatStates[chatID]
	stateMutex.Unlock()

	log.Printf("[%s] %s", message.From.UserName, message.Text)

	switch message.Command() {
	case "start":
		view.SendMessage(bot, message.Chat.ID, "Welcome! Use /word to start a game.")
	case "word":
		word, err := model.GetRandomWord()
		if err != nil {
			view.SendMessage(bot, message.Chat.ID, "Failed to fetch a word.")
			return
		}

		buttons := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("🗣️ Explain", "explain"),
		}

		chatState.Lock()
		chatState.Word = word
		chatState.User = ""
		chatState.Unlock()

		view.SendMessageWithButtons(bot, message.Chat.ID, fmt.Sprintf("The word is ready! Click 'Explain' to explain the word."), buttons)

	default:
		chatState.RLock()
		word := chatState.Word
		user := chatState.User
		chatState.RUnlock()

		if user != "" && service.NormalizeAndCompare(message.Text, word) {
			view.SendMessage(bot, message.Chat.ID, fmt.Sprintf("Congratulations! %s guessed the word correctly.\n /word", message.From.UserName))
			chatState.Lock()
			chatState.Word = ""
			chatState.User = ""
			chatState.Unlock()
		} else if user != "" {
			view.SendMessage(bot, message.Chat.ID, "That's not correct. Try again!")
		}
	}
}

func handleCallbackQuery(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID

	stateMutex.Lock()
	if _, exists := chatStates[chatID]; !exists {
		chatStates[chatID] = &ChatState{}
	}
	chatState := chatStates[chatID]
	stateMutex.Unlock()

	switch callback.Data {
	case "explain":
		chatState.Lock()
		if chatState.User != callback.From.UserName && chatState.User != "" {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, fmt.Sprintf("%s is already explaining the word. %s", chatState.User, callback.From.UserName)))
			chatState.Unlock()
			return
		}
		chatState.User = callback.From.UserName
		chatState.Unlock()
		bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, chatState.Word))
		view.SendMessage(bot, callback.Message.Chat.ID, fmt.Sprintf("%s is explaining the word:", callback.From.UserName))
	default:
		chatState.RLock()
		word := chatState.Word
		chatState.RUnlock()
		fmt.Printf("%s == %s ", callback.Message.Text, word)
		if service.NormalizeAndCompare(callback.Message.Text, word) {
			view.SendMessage(bot, callback.Message.Chat.ID, fmt.Sprintf("Congratulations! %s guessed the word correctly.", callback.From.UserName))
			chatState.Lock()
			chatState.Word = ""
			chatState.User = ""
			chatState.Unlock()
		} else {
			view.SendMessage(bot, callback.Message.Chat.ID, "That's not correct. Try again!")
		}
	}
	bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
}
