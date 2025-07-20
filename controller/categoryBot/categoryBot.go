package categorybot

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
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

func MessageToJSONString(message *tgbotapi.Message) (string, error) {
	jsonBytes, err := json.MarshalIndent(message, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// escapeMarkdownV2 escapes special characters for Telegram MarkdownV2 formatting
func escapeMarkdownV2(text string) string {
	specialChars := []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
	escaped := text
	for _, char := range specialChars {
		escaped = strings.ReplaceAll(escaped, char, "\\"+char)
	}
	return escaped
}

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

	// aiLastResponse stores the last AI response per chat for follow-up hints
	aiLastResponse  = make(map[int64]string)
	aiResponseMutex = &sync.RWMutex{}
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
	"💡",  // Bulb 20
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

func createCategoryBotKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		// First line with a single button
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("See word 👀", "explain"),
		),
		// Second line with multiple buttons
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("nxt⏭️", "next"),
			tgbotapi.NewInlineKeyboardButtonData("flower❀", "flower"),
			tgbotapi.NewInlineKeyboardButtonData("car🏎️𖦹 ׂ 𓈒", "car"),
			tgbotapi.NewInlineKeyboardButtonData("animal🐾", "animal"),
			tgbotapi.NewInlineKeyboardButtonData("AI Hint 💡", "ai_hint"),
		),
		// Third line with a single button
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Changed my mind ❌", "droplead"),
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
		// text := message.Text
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
				"1. Claim leadership by via button or using the /word command.\n" +
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
		case "addBot":
			button := tgbotapi.NewInlineKeyboardButtonURL("add to group", "https://t.me/Croco_rebirth_bot?startgroup=true")
			view.SendMessageWithKeyboardButton(bot, chatID, "Unlock my full potential by adding me to a group chat!", button)
		case "start":
			buttons := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Get Hint"+telegramReactions[20], "hint")))
			view.SendMessageWithButtons(bot, message.Chat.ID, "Heyyy! Got a word for ya 😏 Tap the button below if you need a lil hint 👇", buttons)
		case "exportdata":
			if message.From.ID != int(adminID) {
				log.Printf("not an admin")
				return
			}

			// Helper function to export and send data
			exportAndSend := func(name string, filename string) error {
				data, err := service.ExportAllData(client, name)
				if err != nil {
					return fmt.Errorf("failed to export %s data: %w", name, err)
				}
				err = view.SendFile(bot, adminID, filename, data)
				if err != nil {
					return fmt.Errorf("failed to send %s data file: %w", name, err)
				}
				return nil
			}

			// Export and send both datasets
			if err := exportAndSend("CrocEnLeader", "croc_en_leader_data.json"); err != nil {
				view.SendMessage(bot, chatID, err.Error())
				return
			}
			if err := exportAndSend("CrocEn", "croc_en_data.json"); err != nil {
				view.SendMessage(bot, chatID, err.Error())
				return
			}

			view.SendMessage(bot, chatID, "Both datasets exported and sent to admin successfully.")
			return
		case "stats":
			result := service.LeaderBoardList(client, "CrocEn")
			view.SendMessagehtml(bot, chatID, result)
		case "mystats":
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
			result := service.GetUserStatsByID(client, userID)
			view.ReplyToMessage(bot, message.MessageID, chatID, result)
		case "leaderstats":
			result := service.LeaderBoardList(client, "CrocEnLeader")
			view.SendMessagehtml(bot, message.Chat.ID, result)
		case "installAI":
			logs, err := installOllama.Install(true)
			logsText := strings.Join(logs, "\n")
			if err != nil {
				view.SendMessage(bot, chatID, logsText+"\nLogs:\n"+err.Error())
			}
			view.SendMessage(bot, chatID, logsText+"\nLogs:\n")
		case "buildModel":
			// Prepare the command
			output, err := installOllama.BuildOllamaModel()
			if err != nil {
				view.SendMessage(bot, chatID, "Build fail Error:"+err.Error())
			}
			view.SendMessage(bot, chatID, "\nLogs:\n"+output)
		case "report":
			msgstr, _ := MessageToJSONString(message)
			view.SendMessage(bot, adminID, msgstr)
			view.SendMessage(bot, chatID, "Thank you! Your report has been successfully submitted.")
		case "getlogs":
			logFilePath := "output.log"
			data, err := os.ReadFile(logFilePath)
			if err != nil {
				view.SendMessage(bot, chatID, fmt.Sprintf("Failed to read log file: %v", err))
				return
			}
			err = view.SendFile(bot, adminID, "output.log", data)
			if err != nil {
				view.SendMessage(bot, chatID, fmt.Sprintf("Failed to send log file: %v", err))
				return
			}
			view.SendMessage(bot, chatID, "Log file sent to admin successfully.")
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
		case "hint":
			break
		case "reveal":
			break
		case "":
			break
		default:
			view.SendMessage(bot, chatID, "OOPS! not supported in DM.")
		}

		chatState.RLock()
		wordEmpty := chatState.Word == ""
		lastHint := chatState.LastHintTimestamp
		lastHintType := chatState.LastHintTypeSent
		chatState.RUnlock()
		// Start a new game if no word or lead expired
		if wordEmpty || !chatState.isLeaderActive(640*time.Second) {
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
			return
		}

		// Handle /hint command in DM
		if message.Command() == "hint" {
			if !lastHint.IsZero() && time.Since(lastHint) < 8*time.Second {
				sentMsg, err := view.SendMessage(bot, chatID, "Please take a moment to think before asking for another hint.")
				deleteWarningMessage(bot, message, sentMsg, err)
				return
			}

			aiModeMutex.Lock()
			aiOn := aiModeUsers[chatID]
			aiModeMutex.Unlock()

			if aiOn {
				// Use AI to generate hint based on current word
				word := chatState.Word
				if word == "" {
					view.SendMessage(bot, chatID, "No active word to provide a hint for.")
					return
				}

				wordChannel, errChannel := installOllama.RunOllama("Explain the word: " + word)

				initialMsg := tgbotapi.NewMessage(chatID, "Thinking...")
				initialMessage, err := bot.Send(initialMsg)
				if err != nil {
					log.Println("Failed to send initial message:", err)
					return
				}

				var accumulatedText string
				for word := range wordChannel {
					accumulatedText += word + " "

					aiResponseMutex.Lock()
					aiLastResponse[chatID] = accumulatedText
					aiResponseMutex.Unlock()

					editedMsg := tgbotapi.NewEditMessageText(chatID, initialMessage.MessageID, strings.TrimSpace(accumulatedText))
					_, err := bot.Send(editedMsg)
					if err != nil {
						log.Println("Failed to update message:", err)
					}
				}

				if err := <-errChannel; err != nil {
					errorMsg := tgbotapi.NewMessage(chatID, err.Error())
					_, err := bot.Send(errorMsg)
					if err != nil {
						log.Println("Failed to send error message:", err)
					}
					return
				}
				return
			}

			// Fallback to existing hint logic if AI mode is off
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
			view.ReactToMessage(bot.Token, chatID, message.MessageID, telegramReactions[rand.Intn(8)+13], true)
			view.ReactToMessage(bot.Token, chatID, message.MessageID, telegramReactions[rand.Intn(8)+13], true)
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
		return
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
	case "ai_on":
		view.SendMessage(bot, chatID, "AI mode is now enabled! Enjoy the smart responses.")
		aiModeMutex.Lock()
		aiModeUsers[chatID] = true
		aiModeMutex.Unlock()
		view.SendMessage(bot, chatID, "AI mode is now enabled! Enjoy the smart responses.")
		return
	case "start":
		view.SendMessage(bot, message.Chat.ID, "Welcome! Type /word to start a new game.")
	case "stats":
		result := service.LeaderBoardList(client, "CrocEn")
		view.SendMessagehtml(bot, message.Chat.ID, result)
	case "mystats":
		result := service.GetUserStatsByID(client, message.From.ID)
		view.ReplyToMessage(bot, message.MessageID, chatID, result)
	case "leaderstats":
		result := service.LeaderBoardList(client, "CrocEnLeader")
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
		aiModeMutex.Lock()
		aiOn := aiModeUsers[chatID]
		aiModeMutex.Unlock()

		if aiOn {
			// Use AI to generate hint based on current word
			word := chatState.Word
			if word == "" {
				view.SendMessage(bot, chatID, "No active word to provide a hint for.")
				return
			}

			wordChannel, errChannel := installOllama.RunOllama("Give me a riddle about a " + word)

			initialMsg := tgbotapi.NewMessage(chatID, "Thinking...")
			initialMessage, err := bot.Send(initialMsg)
			if err != nil {
				log.Println("Failed to send initial message:", err)
				return
			}

			var accumulatedText string
			for word := range wordChannel {
				accumulatedText += word + " "

				aiResponseMutex.Lock()
				aiLastResponse[chatID] = accumulatedText
				aiResponseMutex.Unlock()

				editedMsg := tgbotapi.NewEditMessageText(chatID, initialMessage.MessageID, strings.TrimSpace(accumulatedText))
				_, err := bot.Send(editedMsg)
				if err != nil {
					log.Println("Failed to update message:", err)
				}
			}

			if err := <-errChannel; err != nil {
				errorMsg := tgbotapi.NewMessage(chatID, err.Error())
				_, err := bot.Send(errorMsg)
				if err != nil {
					log.Println("Failed to send error message:", err)
				}
				return
			}
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

		// Send chat action "typing" before sending hint
		// chatAction := tgbotapi.NewChatAction(message.Chat.ID, tgbotapi.ChatTyping)
		// bot.Send(chatAction)

		// Send chat action "typing" before sending hint
		chatAction := tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping)
		bot.Send(chatAction)

		// Escape MarkdownV2 special characters in hint text
		escapedHint := escapeMarkdownV2(hint)

		// Wrap escaped hint text in spoiler formatting for Telegram MarkdownV2
		spoilerHint := "||" + escapedHint + "||"
		msg := tgbotapi.NewMessage(chatID, spoilerHint)
		msg.ParseMode = "MarkdownV2"
		_, err := bot.Send(msg)
		if err != nil {
			log.Printf("Failed to send hint message with spoiler formatting: %v", err)
		}

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
			go view.ReactToMessage(bot.Token, chatID, message.MessageID, telegramReactions[rand.Intn(8)+13], true)
			go view.ReactToMessage(bot.Token, chatID, message.MessageID, telegramReactions[rand.Intn(8)+13], true)
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
			buttons := createCategoryBotKeyboard()
			chatState.Word = word
			view.SendMessageWithButtons(bot, callback.Message.Chat.ID, fmt.Sprintf(" [%s](tg://user?id=%d) is explaining the word!", callback.From.FirstName, callback.From.ID), buttons)

			// Remove the inline keyboard (buttons) from the "claim leadership" message when someone starts leading
			editMarkup := tgbotapi.NewEditMessageReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}})
			_, err = bot.Send(editMarkup)
			if err != nil {
				log.Printf("Failed to remove inline keyboard: %v", err)
			}
		}
		chatState.Leader = callback.From.FirstName
		chatState.LeadTimestamp = time.Now()
		chatState.Unlock()
		bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, chatState.Word))
	case "ai_hint":
		aiResponseMutex.RLock()
		aiResponseMutex.RUnlock()
		word := chatState.Word
		if word == "" {
			view.SendMessage(bot, chatID, "No active word to provide a hint for.")
			return
		}

		wordChannel, errChannel := installOllama.RunOllama("Explain \"" + word + "\"")

		initialMsg := tgbotapi.NewMessage(chatID, "Thinking...")
		initialMessage, err := bot.Send(initialMsg)
		if err != nil {
			log.Println("Failed to send initial message:", err)
			return
		}

		var accumulatedText string
		for word := range wordChannel {
			accumulatedText += word + " "

			aiResponseMutex.Lock()
			aiLastResponse[chatID] = accumulatedText
			aiResponseMutex.Unlock()

			editedMsg := tgbotapi.NewEditMessageText(chatID, initialMessage.MessageID, strings.TrimSpace(accumulatedText))
			_, err := bot.Send(editedMsg)
			if err != nil {
				log.Println("Failed to update message:", err)
			}
		}

		if err := <-errChannel; err != nil {
			errorMsg := tgbotapi.NewMessage(chatID, err.Error())
			_, err := bot.Send(errorMsg)
			if err != nil {
				log.Println("Failed to send error message:", err)
			}
			return
		}

		// Generate a simple follow-up hint by truncating or modifying the last AI response
		hint := accumulatedText
		if len(hint) > 100 {
			hint = hint[:100] + "..."
		}
		view.SendMessage(bot, chatID, "💡 AI Hint: "+hint)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
	case "animal":
		chatState.Lock()
		if chatState.User != callback.From.ID && chatState.User != 0 {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, fmt.Sprintf("someone is already explaining the word. %s", callback.From.UserName)))
			chatState.Unlock()
			return
		}
		if chatState.User == 0 {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, fmt.Sprintf("%s Please click on see word/claim Leadership", callback.From.FirstName)))
			chatState.Unlock()
			return
		}
		chatState.User = callback.From.ID
		chatState.Unlock()
		word, err := model.GetRandomAnimal()
		if err != nil {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, "Failed to get an animal word. Please try again later."))
			return
		}
		chatState.Word = word
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

	case "flower":
		chatState.Lock()
		if chatState.User != callback.From.ID && chatState.User != 0 {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, fmt.Sprintf("someone is already explaining the word. %s", callback.From.UserName)))
			chatState.Unlock()
			return
		}
		if chatState.User == 0 {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, fmt.Sprintf("%s Please click on see word/claim Leadership", callback.From.FirstName)))
			chatState.Unlock()
			return
		}
		chatState.User = callback.From.ID
		chatState.Unlock()
		chatState.Word, _ = model.GetRandomFlower()
		bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, chatState.Word))

	case "car":
		chatState.Lock()
		if chatState.User != callback.From.ID && chatState.User != 0 {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, fmt.Sprintf("someone is already explaining the word. %s", callback.From.UserName)))
			chatState.Unlock()
			return
		}
		if chatState.User == 0 {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, fmt.Sprintf("%s Please click on see word/claim Leadership", callback.From.FirstName)))
			chatState.Unlock()
			return
		}
		chatState.User = callback.From.ID
		chatState.Unlock()
		chatState.Word, _ = model.GetRandomCar()
		bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, chatState.Word))

	case "droplead":
		chatState.Lock()
		if chatState.User != callback.From.ID {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, "You are not the current leader, so you cannot drop the lead!"))
			chatState.Unlock()
			return
		}
		// Delete the callback message when user selects "Changed my mind"
		deleteMsg := tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID)
		_, err := bot.DeleteMessage(deleteMsg)
		if err != nil {
			log.Printf("Failed to delete message on droplead: %v", err)
		}
		chatState.Unlock()
		buttons := createSingleButtonKeyboard("🌟 Claim Leadership 🙋", "explain")
		view.SendMessageWithButtons(bot, callback.Message.Chat.ID, fmt.Sprintf("%s refused to lead -> %s \n", callback.From.FirstName, chatState.Word), buttons)
		chatState.reset()

	case "hint":
		chatState.RLock()
		wordEmpty := chatState.Word == ""
		lastHint := chatState.LastHintTimestamp
		lastHintType := chatState.LastHintTypeSent
		chatState.RUnlock()

		if wordEmpty {
			view.SendMessage(bot, callback.Message.Chat.ID, "No active game right now. Click below to start one! \n  __		/start		__")
			bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
			return
		}

		if !lastHint.IsZero() && time.Since(lastHint) < 1*time.Minute {
			sentMsg, err := view.SendMessage(bot, chatID, "Please take a minute before requesting another hint.")
			deleteWarningMessage(bot, callback.Message, sentMsg, err)
			bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))

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

		chatAction := tgbotapi.NewChatAction(callback.Message.Chat.ID, tgbotapi.ChatTyping)
		bot.Send(chatAction)

		escapedHint := escapeMarkdownV2(hint)
		spoilerHint := "||" + escapedHint + "||"
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, spoilerHint)
		msg.ParseMode = "MarkdownV2"
		_, err := bot.Send(msg)
		if err != nil {
			log.Printf("Failed to send hint message with spoiler formatting: %v", err)
		}

		chatState.Lock()
		chatState.LastHintTimestamp = time.Now()
		chatState.LastHintTypeSent = 1 - lastHintType
		chatState.Unlock()

		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
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
