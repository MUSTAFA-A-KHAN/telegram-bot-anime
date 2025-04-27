package controller

import (
	"fmt"
	"log"
	"net/http"
	"os"
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

// StartBot initializes and starts the bot
func StartBot(token string) error {
	go startHTTPServer() //start http server with go routine
	// Create a new instance of the bot using the provided token.
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return err
	}

	// Enable debug mode to log detailed information about bot operations.
	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Configure update settings with a timeout of 60 seconds.
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	// Get the updates channel to receive incoming messages and callback queries.
	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		return err
	}

	// Process incoming updates (messages and callback queries) in a loop.
	for update := range updates {
		if update.Message != nil {
			// Handle incoming messages.
			go handleMessage(bot, update.Message)
		} else if update.CallbackQuery != nil {
			// Handle incoming callback queries.
			go handleCallbackQuery(bot, update.CallbackQuery)
		}
	}

	return nil
}

// handleMessage processes incoming messages and handles commands and guesses.
func handleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	adminID := int64(1006461736)

	// Ensure the chat state exists, and initialize it if necessary.
	stateMutex.Lock()
	if _, exists := chatStates[chatID]; !exists {
		chatStates[chatID] = &ChatState{}
	}
	chatState := chatStates[chatID]
	stateMutex.Unlock()

	log.Printf("[%s] %s", message.From.UserName, message.Text)

	switch message.Command() {
	case "getButton":
		Announcement := strings.Split(message.Text, "  ")
		if len(Announcement) > 1 {
			parts := strings.Split(Announcement[2], " ")
			if len(parts) > 2 {
				url := parts[0]

				// Extract the message (everything after the URL)
				messageText := strings.Join(parts[1:], " ")
				button := tgbotapi.NewInlineKeyboardButtonURL(messageText, url)
				view.SendMessageWithKeyboardButton(bot, message.Chat.ID, Announcement[1], button)
			}
		}
	case "start":
		// Send a welcome message with instructions to start the game.
		variable := os.Getenv("TEST")
		view.SendMessage(bot, message.Chat.ID, "Welcome! Use /word to start a game.test-"+variable)
	case "stats":
		// Send the user stats of game.
		result := service.LeaderBoardList("CrocEn")
		view.SendMessage(bot, message.Chat.ID, result)
	case "leaderstats":
		// Send the user stats of game.
		result := service.LeaderBoardList("CrocEnLeader")
		view.SendMessage(bot, message.Chat.ID, result)
	case "report":
		// Allow users to report an issue or feedback.
		if len(message.Text) > 7 { // "report " has 7 characters
			reportMessage := message.Text[7:] // Extract the message after the "report " command
			adminMessage := fmt.Sprintf("Report from @%s (%d):\n%s", message.From.UserName, message.From.ID, reportMessage)
			// Send the report to the admin
			view.SendMessage(bot, adminID, adminMessage)
			view.SendMessage(bot, chatID, "Your report has been submitted. Thank you!")
		} else {
			view.SendMessage(bot, chatID, "Please provide a message with your report. Usage: /report [your message]")
		}
	case "word":
		// Fetch a random word from the model.
		word, err := model.GetRandomWord()
		if err != nil {
			view.SendMessage(bot, message.Chat.ID, "Failed to fetch a word.")
			return
		}

		// Create a button to start explaining the word.
		// Create the inline keyboard with each button on a separate line.
		buttons := tgbotapi.NewInlineKeyboardMarkup(
			// First line with a single button
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(" 🗣️ Explain ", "explain"),
			),
		)
		// Update the chat state with the new word and reset the user explaining it.
		chatState.Lock()
		chatState.Word = word
		chatState.User = 0
		chatState.Leader = ""
		chatState.Unlock()
		view.SendSticker(bot, chatID, "CAACAgUAAxkBAAEwCnNnYW-OkgV7Odt9osVwoBSzLC6vsAACMhMAAj45CFdCstMoIYiPfjYE")

		// Send a message with the word and the explain button.
		view.SendMessageWithButtons(bot, message.Chat.ID, fmt.Sprintf("The word is ready! Click 'Explain' to explain the word."), buttons)

	default:
		// Handle guesses from users.
		chatState.RLock()
		word := chatState.Word
		user := chatState.User
		chatState.RUnlock()

		// Check if the guessed word matches the current word.
		if user != 0 && service.NormalizeAndCompare(message.Text, word) && message.From.ID != user {
			buttons := tgbotapi.NewInlineKeyboardMarkup(
				// First line with a single button
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("🌟 Claim Leadership 🙋", "explain"),
				),
			)
			view.SendMessageWithButtons(bot, message.Chat.ID, fmt.Sprintf("Congratulations! %s guessed the word %s.\n /word", message.From.FirstName, chatState.Word), buttons)
			client := repository.DbManager()
			repository.InsertDoc(message.From.ID, message.From.FirstName, message.Chat.ID, client, "CrocEn")
			repository.InsertDoc(chatState.User, chatState.Leader, message.Chat.ID, client, "CrocEnLeader")
			// Reset the chat state after a correct guess.
			chatState.Lock()
			chatState.Word = ""
			chatState.User = 0
			chatState.LeadTimestamp = time.Time{}
			chatState.Unlock()
		} else if user != 0 {
			// view.SendMessage(bot, message.Chat.ID, "That's not correct. Try again!")
		}
	}
}

// handleCallbackQuery processes incoming callback queries and handles the "explain" action.
func handleCallbackQuery(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID

	// Ensure the chat state exists, and initialize it if necessary.
	stateMutex.Lock()
	if _, exists := chatStates[chatID]; !exists {
		chatStates[chatID] = &ChatState{}
	}
	chatState := chatStates[chatID]
	stateMutex.Unlock()

	switch callback.Data {
	case "explain":
		// Handle the "explain" action.
		chatState.Lock()
		if chatState.User != callback.From.ID && chatState.User != 0 && time.Since(chatState.LeadTimestamp) < 120*time.Second {
			// If another user is already explaining the word, alert the current user.
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, fmt.Sprintf("%s is already explaining the word. %s", chatState.User, callback.From.UserName)))

			chatState.Unlock()
			return
		}
		if chatState.User == callback.From.ID {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, chatState.Word))

		}
		if chatState.User == 0 || time.Since(chatState.LeadTimestamp) >= 120*time.Second && chatState.User != callback.From.ID {
			word, err := model.GetRandomWord()
			if err != nil {
				return
			}
			// Create the inline keyboard with each button on a separate line.
			buttons := tgbotapi.NewInlineKeyboardMarkup(
				// First line with a single button
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("See word 👀", "explain"),
				),
				// Second line with a single button
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Next ⏭️", "next"),
				),
				// Third line with a single button
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Changed my mind ❌", "droplead"),
				),
			)
			chatState.Word = word
			// leader := fmt.Sprintf("[%s](tg://user?id=%d)", callback.From.UserName, callback.From.ID)
			// leader=tgbotapi.Inline
			view.SendMessageWithButtons(bot, callback.Message.Chat.ID, fmt.Sprintf(" [%s](tg://user?id=%d)is explaining the word!", callback.From.FirstName, callback.From.ID), buttons)
		}
		// Set the current user as the one explaining the word.
		chatState.User = callback.From.ID
		chatState.Leader = callback.From.FirstName
		chatState.LeadTimestamp = time.Now()
		chatState.Unlock()
		// Notify the user about the word to explain.
		bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, chatState.Word))

	case "next":
		// Handle the "next" action.
		chatState.Lock()
		if chatState.User != callback.From.ID && chatState.User != 0 {
			// If another user is already explaining the word, alert the current user.
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, fmt.Sprintf("%s is already explaining the word. %s", chatState.User, callback.From.UserName)))
			chatState.Unlock()
			return
		}
		if chatState.User == 0 {
			// If no user is explaining the word, alert the current user.
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, fmt.Sprintf("%s Please click on see word/claim Leadership", callback.From.FirstName)))
			chatState.Unlock()
			return
		}
		// Set the current user as the one explaining the word.
		chatState.User = callback.From.ID
		chatState.Leader = callback.From.FirstName
		chatState.Unlock()
		// Notify the user about the word to explain.
		chatState.Word, _ = model.GetRandomWord()
		bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, chatState.Word))
		// view.SendMessage(bot, callback.Message.Chat.ID, fmt.Sprintf("%s is explaining the word:", callback.From.UserName))
	case "droplead":
		// Handle the "droplead" action.
		chatState.Lock()
		if chatState.User != callback.From.ID {
			// If the current user is not the leader, prevent them from dropping the lead.
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, "You are not the leader, so you cannot drop the lead!"))
			chatState.Unlock()
			return
		}
		buttons := tgbotapi.NewInlineKeyboardMarkup(
			// First line with a single button
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🌟 Claim Leadership 🙋", "explain"),
			),
		)
		// Reset the chat state after dropping the lead.
		view.SendMessageWithButtons(bot, callback.Message.Chat.ID, fmt.Sprintf("%s refused to lead -> %s \n", callback.From.FirstName, chatState.Word), buttons)
		chatState.Word = ""
		chatState.User = 0
		chatState.LeadTimestamp = time.Time{}
		chatState.Unlock()
	default:
		// Handle guesses from callback queries (if any).
		chatState.RLock()
		word := chatState.Word
		chatState.RUnlock()
		fmt.Printf("%s == %s ", callback.Message.Text, word)
		// Check if the guessed word matches the current word.
		if service.NormalizeAndCompare(callback.Message.Text, word) {
			fmt.Print("calling Sendmessage")
			buttons := tgbotapi.NewInlineKeyboardMarkup(
				// First line with a single button
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("🌟 Claim Leadership 🙋", "explain"),
				),
			)
			view.SendMessageWithButtons(bot, callback.Message.Chat.ID, fmt.Sprintf("Congratulations! %s guessed the word correctly.", callback.From.FirstName), buttons)
			fmt.Println("calling DBManager")
			// Reset the chat state after a correct guess.
			chatState.Lock()
			chatState.Word = ""
			chatState.User = 0
			chatState.Unlock()
		} else {
			// view.SendMessage(bot, callback.Message.Chat.ID, "That's not correct. Try again!")
		}
	}
	// Acknowledge the callback query to remove the "loading" state in the client.
	bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
}

// startHTTPServer starts a simple HTTP server for health checks
func startHTTPServer() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Bot is running!")
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
