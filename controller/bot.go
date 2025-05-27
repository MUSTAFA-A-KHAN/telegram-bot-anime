package controller

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/model"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/service"
	installOllama "github.com/MUSTAFA-A-KHAN/telegram-bot-anime/service/installOllama"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/view"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"go.mongodb.org/mongo-driver/mongo"
)

// ChatState holds the state for a specific chat, including the current word and user explaining it.
type ChatState struct {
	sync.RWMutex
	Word              string
	User              int
	LeadTimestamp     time.Time
	Leader            string
	LastHintTimestamp time.Time
	LastHintTypeSent  int // 0 or 1 to track which hint was last sent
}

var (
	// chatStates is a map that holds the state for each chat, identified by chat ID.
	chatStates = make(map[int64]*ChatState)
	// stateMutex ensures safe access to the chatStates map.
	stateMutex = &sync.RWMutex{}
)

// telegramReactions is a map that holds the reactions for each chat, identified by chat ID.
var telegramReactions = []string{
	"👍",  // Thumbs Up 0
	"👎",  // Thumbs Down 1
	"❤️", // Red Heart 2
	"😂",  // Face with Tears of Joy 3
	"😮",  // Surprised Face 4
	"😢",  // Crying Face 5
	"😡",  // Angry Face 6
	"🎉",  // Party Popper 7
	"🙌",  // Raising Hands 8
	"🤔",  // Thinking Face 9
	"🥰",  // Smiling Face with Hearts 10
	"🤯",  // Exploding Head 11
	"🤬",  // Face with Symbols on Mouth 12
	"👏",  // Clapping Hands 13
	"🤩",  // Star-Struck 14
	"😎",  // Smiling Face with Sunglasses 15
	"💯",  // 100 Points 16
	"🔥",  // Fire 17
	"🥳",  // Partying Face 18
	"⚡",  // Thunder 19
}

// getOrCreateChatState safely retrieves or creates a ChatState for a chatID.
func getOrCreateChatState(chatID int64) *ChatState {
	stateMutex.Lock()
	defer stateMutex.Unlock()
	if _, exists := chatStates[chatID]; !exists {
		chatStates[chatID] = &ChatState{}
	}
	return chatStates[chatID]
}

// deleteWarningMessage deletes a warning message from the chat state.
func deleteWarningMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message, sentMsg tgbotapi.Message, err error) {
	if err == nil {
		time.Sleep(1 * time.Second)
		deleteMsg := tgbotapi.NewDeleteMessage(message.Chat.ID, sentMsg.MessageID)
		_, err := bot.DeleteMessage(deleteMsg)
		if err != nil {
			log.Printf("Failed to delete message: %v", err)
		}
	} else {
		log.Printf("Failed to send message: %v", err)
	}
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

	// Create a single MongoDB client instance once
	client := repository.DbManager()
	if client == nil {
		return fmt.Errorf("failed to connect to MongoDB")
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		return err
	}

	// Pass the client instance to handleMessage and handleCallbackQuery via closure
	for update := range updates {
		if update.Message != nil {
			go handleMessage(bot, update.Message, client)
		} else if update.CallbackQuery != nil {
			go handleCallbackQuery(bot, update.CallbackQuery, client)
		}
	}

	return nil
}

var aiModeUsers = make(map[int64]bool)
var aiModeMutex = &sync.Mutex{}

// handleMessage processes incoming messages and handles commands and guesses.
func handleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message, client *mongo.Client) {
	chatID := message.Chat.ID
	adminID := int64(1006461736)

	chatState := getOrCreateChatState(chatID)

	log.Printf("[%s] %s", message.From.UserName, message.Text)

	// New DM scenario: if chat is private, bot gives hint and user guesses
	if message.Chat.IsPrivate() {
		fmt.Println("------------------------------------------" + message.Command() + "------------------------------------------")
		text := message.Text
		switch message.Command() {
		case "ai_on":
			aiModeMutex.Lock()
			aiModeUsers[chatID] = true
			aiModeMutex.Unlock()
			view.SendMessage(bot, chatID, "AI mode is now enabled! Enjoy the smart responses.")
			return
		case "ai_off":
			aiModeMutex.Lock()
			delete(aiModeUsers, chatID)
			aiModeMutex.Unlock()
			view.SendMessage(bot, chatID, "AI mode has been disabled.")
			return
		case "rules":
			rulesText := "*🎮 Game Rules 🎮*\n\n" +
				"*Players:*\n" +
				"1. Guess the word by typing your answer.\n" +
				"2. Use /hint to get clues about the word, but wait at least a minute between hints.\n" +
				"3. Use /reveal to reveal the word if you give up, but only after 10 minutes of gameplay.\n\n" +
				"*Leaders:*\n" +
				"1. Claim leadership by clicking 'Explain' or using the appropriate command.\n" +
				"2. Explain the word to other players without directly saying it.\n" +
				"3. You can get a new word or drop leadership using the provided buttons.\n\n" +
				"*General:*\n" +
				"1. Be respectful and fair to other players.\n" +
				"2. Have fun and enjoy the game!\n\n" +
				"Type /word to start a new game or /rules to see these rules again."
			msg := tgbotapi.NewMessage(chatID, rulesText)
			msg.ParseMode = "Markdown"
			_, err := bot.Send(msg)
			if err != nil {
				log.Printf("Failed to send rules message: %v", err)
			}
			return
		}

		aiModeMutex.Lock()
		aiOn := aiModeUsers[chatID]
		aiModeMutex.Unlock()

		if aiOn {
			// AI processing here
			wordChannel, errChannel := installOllama.RunOllama(text)

			// Send the initial message (could be an empty string or placeholder)
			initialMsg := tgbotapi.NewMessage(chatID, "Hello")
			initialMessage, err := bot.Send(initialMsg)
			if err != nil {
				log.Println("Failed to send initial message:", err)
				return
			}

			// Start a variable to accumulate the text as we receive each word
			var accumulatedText string

			// Process words as they arrive
			for word := range wordChannel {
				// Accumulate the word and append it to the message content
				accumulatedText += word + " "

				// Update the same message with the accumulated text
				editedMsg := tgbotapi.NewEditMessageText(chatID, initialMessage.MessageID, strings.TrimSpace(accumulatedText))
				_, err := bot.Send(editedMsg)
				if err != nil {
					log.Println("Failed to update message:", err)
				}
			}

			// If an error occurs during execution, send it to the user
			if err := <-errChannel; err != nil {
				// Send an error message if something goes wrong
				errorMsg := tgbotapi.NewMessage(chatID, err.Error())
				_, err := bot.Send(errorMsg)
				if err != nil {
					log.Println("Failed to send error message:", err)
				}
				return
			}
			//
		}

		if message.Command() == "stats" {
			result := service.LeaderBoardList("CrocEn")
			view.SendMessagehtml(bot, chatID, result)
		}
		if message.Command() == "mystats" {
			// args := strings.Fields(message.CommandArguments())
			// if len(args) < 1 {
			// 	view.SendMessage(bot, chatID, "Please provide a user ID. Usage: /userstats <userID>")
			// 	return
			// }
			// userIDStr := args[0]
			ID := strconv.Itoa(message.From.ID)
			userID, err := strconv.Atoi(ID)
			if err != nil {
				sentMsg, err := view.SendMessage(bot, chatID, "Invalid user ID. Please enter a valid numeric user ID.")
				deleteWarningMessage(bot, message, sentMsg, err)
				return
			}
			result := service.GetUserStatsByID(userID)
			view.SendMessage(bot, chatID, result)
		}

		if message.Command() == "installAI" {
			logs, err := installOllama.Install(true)
			logsText := strings.Join(logs, "\n")
			if err != nil {
				view.SendMessage(bot, chatID, logsText+"\nLogs:\n"+err.Error())
			}
			view.SendMessage(bot, chatID, logsText+"\nLogs:\n")
		}
		if message.Command() == "buildModel" {
			// Prepare the command
			output, err := installOllama.BuildOllamaModel()
			if err != nil {
				view.SendMessage(bot, chatID, "Build fail Error:"+err.Error())
			}
			view.SendMessage(bot, chatID, "\nLogs:\n"+output)
		}
		// if message.Command() == "executeAI" {
		// 	userPrompt := strings.TrimSpace(message.CommandArguments())
		// 	if userPrompt == "" {
		// 		view.SendMessage(bot, chatID,
		// 			"Please add a prompt, e.g.  /executeAI Explain Newton’s third law")
		// 		return
		// 	}

		// 	result, err := installOllama.RunOllama(userPrompt)
		// 	if err != nil {
		// 		view.SendMessage(bot, chatID, "Error:"+err.Error())
		// 		return
		// 	}
		// 	view.SendMessage(bot, chatID, result)
		// 	return
		// }

		if message.Command() == "leaderstats" {
			result := service.LeaderBoardList("CrocEnLeader")
			view.SendMessagehtml(bot, chatID, result)
		}
		chatState.RLock()
		wordEmpty := chatState.Word == ""
		lastHint := chatState.LastHintTimestamp
		lastHintType := chatState.LastHintTypeSent
		chatState.RUnlock()
		// Start a new game if no word or lead expired
		if wordEmpty || time.Since(chatState.LeadTimestamp) >= 640*time.Second {
			word, err := model.GetRandomWord()
			if err != nil {
				view.SendMessage(bot, chatID, "Oops! Unable to fetch a word right now. Please try again later.")
				return
			}
			chatState.Lock()
			chatState.Word = word
			chatState.User = message.From.ID
			chatState.Leader = message.From.FirstName
			chatState.LeadTimestamp = time.Now()
			chatState.LastHintTimestamp = time.Time{}
			chatState.LastHintTypeSent = 0
			chatState.Unlock()
			button := tgbotapi.NewInlineKeyboardButtonURL("add to group", "https://t.me/Croco_rebirth_bot?startgroup=true")
			view.SendMessageWithKeyboardButton(bot, chatID, "Unlock my full potential by adding me to a group chat! 🔓\nMeanwhile, here's a word for you to guess. Need a hint? Just type 👉 /hint.", button)
			return
		}

		// Handle /hint command in DM
		if message.Command() == "hint" {
			if !lastHint.IsZero() && time.Since(lastHint) < 8*time.Second {
				sentMsg, err := view.SendMessage(bot, chatID, "Please take a moment to think before asking for another hint.")
				deleteWarningMessage(bot, message, sentMsg, err)
				return
			}

			chatState.RLock()
			var hint string
			if lastHintType == 0 {
				hint = model.GenerateMeaningHint(chatState.Word)
			} else {
				hint = model.GenerateMeaningHint(chatState.Word)
				hint = hint + "\n" + model.GenerateHint(chatState.Word)
				hint = hint + "\n" + model.GenerateAuroraHint(chatState.Word)
			}
			chatState.RUnlock()

			view.SendMessage(bot, chatID, hint)

			chatState.Lock()
			chatState.LastHintTimestamp = time.Now()
			chatState.LastHintTypeSent = 1 - lastHintType
			chatState.Unlock()
			return
		}

		// Handle /reveal command in DM
		if message.Command() == "reveal" {
			if time.Since(chatState.LeadTimestamp) >= 6*time.Second {
				view.SendMessage(bot, chatID, fmt.Sprintf("The word was: %s", chatState.Word))
				chatState.reset()
			} else {
				sentMsg, err := view.SendMessage(bot, chatID, "Please try to read the hint before revealing the word.")
				deleteWarningMessage(bot, message, sentMsg, err)
			}
			return
		}

		// Check user's guess in DM
		chatState.RLock()
		word := chatState.Word
		chatState.RUnlock()

		if service.NormalizeAndCompare(message.Text, word) && message.From.ID == chatState.User {
			view.SendMessage(bot, chatID, fmt.Sprintf("%s ! You guessed the word '%s' correctly!", telegramReactions[7], word))
			view.ReactToMessage(bot.Token, chatID, message.MessageID, telegramReactions[17], true)
			view.ReactToMessage(bot.Token, chatID, message.MessageID, "⚡", true)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("Recovered from panic in InsertDoc goroutine: %v", r)
					}
				}()
				repository.InsertDoc(message.From.ID, message.From.FirstName, chatID, client, "CrocEn")
			}()

			chatState.reset()
			return
		}
		if message.Command() == "report" {
			msgstr, _ := MessageToJSONString(message)
			view.SendMessage(bot, adminID, msgstr)
			view.SendMessage(bot, chatID, "Thank you! Your report has been successfully submitted.")
		} else {
			view.SendMessage(bot, chatID, "Try again! Need a clue? Type 👉 /hint.\nOr reveal the word by typing 👉 /reveal.")
			return
		}
	}

	// Existing group chat handling
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
		view.SendMessage(bot, message.Chat.ID, "Welcome! Type /word to start a new game.")
	case "stats":
		result := service.LeaderBoardList("CrocEn")
		view.SendMessagehtml(bot, message.Chat.ID, result)
	case "mystats":
		result := service.GetUserStatsByID(message.From.ID)
		view.SendMessage(bot, chatID, result)
	case "leaderstats":
		result := service.LeaderBoardList("CrocEnLeader")
		view.SendMessagehtml(bot, message.Chat.ID, result)
	case "rules":
		rulesText := "*🎮 Game Rules 🎮*\n\n" +
			"*Players:*\n" +
			"1. Guess the word by typing your answer.\n" +
			"2. Use /hint to get clues about the word, but wait at least a minute between hints.\n" +
			"3. Use /reveal to reveal the word if you give up, but only after 10 minutes of gameplay.\n\n" +
			"*Leaders:*\n" +
			"1. Claim leadership by clicking 'Explain' or using the appropriate command.\n" +
			"2. Explain the word to other players without directly saying it.\n" +
			"3. You can get a new word or drop leadership using the provided buttons.\n\n" +
			"*General:*\n" +
			"1. Be respectful and fair to other players.\n" +
			"2. Have fun and enjoy the game!\n\n" +
			"Type /word to start a new game or /rules to see these rules again."
		msg := tgbotapi.NewMessage(chatID, rulesText)
		msg.ParseMode = "Markdown"
		_, err := bot.Send(msg)
		if err != nil {
			log.Printf("Failed to send rules message: %v", err)
		}
		return
	case "report":
		// if len(message.Text) > 7 {
		// reportMessage := message.Text[7:]
		// adminMessage := fmt.Sprintf("Report from @%s FromID-(%d) ChatID-(%d) From-(%s):\n Message-%s", message.From.UserName, message.From.ID, message.Chat.ID, message.From.FirstName, reportMessage)
		// view.SendMessage(bot, adminID, adminMessage)
		msgstr, _ := MessageToJSONString(message)
		view.SendMessage(bot, adminID, msgstr)
		view.SendMessage(bot, chatID, "Your report has been submitted. Thank you!")
		// } else {
		// 	view.SendMessage(bot, chatID, "Please provide a message with your report. Usage: /report [your message]")
		// }
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
			view.SendMessageWithButtons(bot, message.Chat.ID, "The word is ready! Click 'Explain' to start explaining it.", buttons)
		} else {
			sentMsg, err := view.SendMessage(bot, message.Chat.ID, "A game is currently in progress.")
			deleteWarningMessage(bot, message, sentMsg, err)
		}
	case "hint":
		chatState.RLock()
		wordEmpty := chatState.Word == ""
		lastHint := chatState.LastHintTimestamp
		lastHintType := chatState.LastHintTypeSent
		chatState.RUnlock()

		if wordEmpty {
			buttons := createSingleButtonKeyboard(" 🗣️ Explain ", "explain")
			view.SendMessageWithButtons(bot, message.Chat.ID, "No active game right now. Click below to start one!", buttons)
			return
		}

		if !lastHint.IsZero() && time.Since(lastHint) < 1*time.Minute {
			sentMsg, err := view.SendMessage(bot, message.Chat.ID, "Please wait a minute before requesting another hint.")
			deleteWarningMessage(bot, message, sentMsg, err)
			return
		}

		chatState.RLock()
		var hint string
		if lastHintType == 0 {
			hint = model.GenerateMeaningHint(chatState.Word)
		} else {
			hint = model.GenerateMeaningHint(chatState.Word)
			hint = hint + "\n" + model.GenerateHint(chatState.Word)
		}
		chatState.RUnlock()

		view.SendMessage(bot, message.Chat.ID, hint)

		chatState.Lock()
		chatState.LastHintTimestamp = time.Now()
		chatState.LastHintTypeSent = 1 - lastHintType
		chatState.Unlock()

	case "reveal":
		chatState.RLock()
		word := chatState.Word
		leadTime := chatState.LeadTimestamp
		chatState.RUnlock()

		if time.Since(leadTime) >= 600*time.Second {
			buttons := createSingleButtonKeyboard(" 🗣️ Explain ", "explain")
			view.SendMessageWithButtons(bot, message.Chat.ID, fmt.Sprintf("The word was: %s", word), buttons)

			chatState.reset()
		} else {
			sentMsg, err := view.SendMessage(bot, message.Chat.ID, "Please wait for 10 minutes before revealing the word.")
			deleteWarningMessage(bot, message, sentMsg, err)
		}
	default:
		chatState.RLock()
		word := chatState.Word
		user := chatState.User
		leader := chatState.Leader
		chatState.RUnlock()

		if user != 0 && service.NormalizeAndCompare(message.Text, word) && message.From.ID != user {
			chatState.reset()
			buttons := createSingleButtonKeyboard("🌟 Claim Leadership 🙋", "explain")
			view.SendMessageWithButtons(bot, message.Chat.ID, fmt.Sprintf("%s! %s guessed the word %s.\n /word", telegramReactions[7], message.From.FirstName, word), buttons)
			go view.ReactToMessage(bot.Token, chatID, message.MessageID, "🔥", true)
			go view.ReactToMessage(bot.Token, chatID, message.MessageID, "⚡", true)
			// client := repository.DbManager()
			go repository.InsertDoc(message.From.ID, message.From.FirstName, message.Chat.ID, client, "CrocEn")
			go repository.InsertDoc(user, leader, message.Chat.ID, client, "CrocEnLeader")

		}
	}
}

// handleCallbackQuery processes incoming callback queries and handles the "explain" action.
func handleCallbackQuery(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, client *mongo.Client) {
	chatID := callback.Message.Chat.ID
	chatState := getOrCreateChatState(chatID)

	switch callback.Data {
	case "explain":
		chatState.Lock()
		if chatState.User != callback.From.ID && chatState.User != 0 && time.Since(chatState.LeadTimestamp) < 600*time.Second {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, fmt.Sprintf("%s is already explaining the word. Please wait for your turn, %s.", chatState.Leader, callback.From.UserName)))
			chatState.Unlock()
			return
		}
		if chatState.User == callback.From.ID {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, chatState.Word))
			chatState.Unlock()
			return
		}
		if chatState.User == 0 || (time.Since(chatState.LeadTimestamp) >= 600*time.Second && chatState.User != callback.From.ID) {
			chatState.User = callback.From.ID
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
			view.SendMessageWithButtons(bot, callback.Message.Chat.ID, fmt.Sprintf(" [%s](tg://user?id=%d) is explaining the word!", callback.From.FirstName, callback.From.ID), buttons)
		}
		chatState.Leader = callback.From.FirstName
		chatState.LeadTimestamp = time.Now()
		chatState.Unlock()
		bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, chatState.Word))

	case "next":
		chatState.Lock()
		if chatState.User != callback.From.ID && chatState.User != 0 {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, fmt.Sprintf("%s is already explaining the word. %s", chatState.Leader, callback.From.UserName)))
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
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, "You are not the current leader, so you cannot drop the lead!"))
			chatState.Unlock()
			return
		}
		chatState.Unlock()
		buttons := createSingleButtonKeyboard("🌟 Claim Leadership 🙋", "explain")
		view.SendMessageWithButtons(bot, callback.Message.Chat.ID, fmt.Sprintf("%s refused to lead -> %s \n", callback.From.FirstName, chatState.Word), buttons)
		chatState.reset()

	default:
		chatState.RLock()
		word := chatState.Word
		chatState.RUnlock()
		if service.NormalizeAndCompare(callback.Message.Text, word) {
			buttons := createSingleButtonKeyboard("🌟 Claim Leadership 🙋", "explain")
			view.SendMessageWithButtons(bot, callback.Message.Chat.ID, fmt.Sprintf("%s! %s guessed the word correctly.", telegramReactions[0], callback.From.FirstName), buttons)
			chatState.Lock()
			chatState.reset()
			chatState.Unlock()
		}
	}
	bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
}
func MessageToJSONString(message *tgbotapi.Message) (string, error) {
	jsonBytes, err := json.MarshalIndent(message, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// startHTTPServer starts a simple HTTP server for health checks
func startHTTPServer() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Bot is running!")
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
