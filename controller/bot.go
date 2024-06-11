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

var (
	explainingWord = struct {
		sync.RWMutex
		word   string
		user   string
		chatID int64
	}{}
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
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			switch update.Message.Command() {
			case "start":
				view.SendMessage(bot, update.Message.Chat.ID, "Welcome! Use /word to start a game.")
			case "word":
				word, err := model.GetRandomWord()
				if err != nil {
					view.SendMessage(bot, update.Message.Chat.ID, "Failed to fetch a word.")
					continue
				}

				buttons := []tgbotapi.InlineKeyboardButton{
					tgbotapi.NewInlineKeyboardButtonData("🗣️ Explain", "explain"),
				}

				explainingWord.Lock()
				explainingWord.word = word
				explainingWord.user = ""
				explainingWord.chatID = update.Message.Chat.ID
				explainingWord.Unlock()

				view.SendMessageWithButtons(bot, update.Message.Chat.ID, fmt.Sprintf("The word is ready! Click 'Explain' to explain the word."), buttons)

			default:
				// Handle guesses here
				explainingWord.RLock()
				word := explainingWord.word
				user := explainingWord.user
				explainingWord.RUnlock()

				if user != "" && service.NormalizeAndCompare(update.Message.Text, word) {
					view.SendMessage(bot, update.Message.Chat.ID, fmt.Sprintf("Congratulations! %s guessed the word correctly. /word", update.Message.From.UserName))
					explainingWord.Lock()
					explainingWord.word = ""
					explainingWord.user = ""
					explainingWord.chatID = 0
					explainingWord.Unlock()
				} else if user != "" {
					view.SendMessage(bot, update.Message.Chat.ID, "That's not correct. Try again!")
				}
			}

		}

		if update.CallbackQuery != nil {
			callback := update.CallbackQuery
			switch callback.Data {
			case "explain":
				explainingWord.Lock()
				if explainingWord.user != callback.From.UserName && explainingWord.user != "" {
					bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, fmt.Sprintf("%s is already explaining the word. %s", explainingWord.user, callback.From.UserName)))
					// view.SendMessage(bot, callback.Message.Chat.ID, fmt.Sprintf("%s is already explaining the word.", explainingWord.user))
					explainingWord.Unlock()
					continue
				}
				explainingWord.user = callback.From.UserName
				explainingWord.Unlock()
				bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, explainingWord.word)) /**uncomment if you need it for multiusers */
				view.SendMessage(bot, callback.Message.Chat.ID, fmt.Sprintf("%s is explaining the word:", callback.From.UserName))
			default:
				explainingWord.RLock()
				word := explainingWord.word
				//user := explainingWord.user
				explainingWord.RUnlock()
				fmt.Println("%s ==%s ", update.Message.Text, word)
				// if strings.EqualFold(update.Message.Text, word) {
				if service.NormalizeAndCompare(update.Message.Text, word) {
					view.SendMessage(bot, callback.Message.Chat.ID, fmt.Sprintf("Congratulations! %s guessed the word correctly.", update.Message.From.UserName))
					explainingWord.Lock()
					explainingWord.word = ""
					explainingWord.user = ""
					explainingWord.chatID = 0
					explainingWord.Unlock()
				} else {
					view.SendMessage(bot, callback.Message.Chat.ID, "That's not correct. Try again!")
				}
			}
			bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
		}
	}

	return nil
}
