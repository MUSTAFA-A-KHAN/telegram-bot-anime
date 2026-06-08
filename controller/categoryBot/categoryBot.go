package categorybot

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/controller/scramybot"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/controller/wordlebot"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/model"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/model/validator"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/service"
	installOllama "github.com/MUSTAFA-A-KHAN/telegram-bot-anime/service/installOllama"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/view"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	lev "github.com/texttheater/golang-levenshtein/levenshtein"
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
	var builder strings.Builder
	builder.Grow(len(text) + len(text)/4) // Rough estimate for escaped string
	for _, char := range text {
		switch char {
		case '_', '*', '[', ']', '(', ')', '~', '`', '>', '#', '+', '-', '=', '|', '{', '}', '.', '!':
			builder.WriteByte('\\')
			builder.WriteRune(char)
		default:
			builder.WriteRune(char)
		}
	}
	return builder.String()
}
func cleanWord(word string) string {
	return strings.TrimFunc(word, func(r rune) bool {
		return unicode.IsPunct(r)
	})
}

func replaceWord(original, word string) string {
	maxDistance := 2 // allow 2-character fuzzy difference

	words := strings.Fields(original)
	for i, w := range words {
		cleaned := cleanWord(w)
		distance := lev.DistanceForStrings(
			[]rune(strings.ToLower(cleaned)),
			[]rune(strings.ToLower(word)),
			lev.DefaultOptions,
		)
		if distance <= maxDistance {
			replacement := strings.Repeat("_", len(cleaned))
			words[i] = strings.Replace(w, cleaned, replacement, 1)
		}
	}

	return strings.Join(words, " ")
}

// CategoryChatStateDoc is the MongoDB-serializable version of ChatState
type CategoryChatStateDoc struct {
	ChatID            int64     `bson:"_id"`
	Word              string    `bson:"word"`
	User              int       `bson:"user"`
	LeadTimestamp     time.Time `bson:"lead_timestamp"`
	Leader            string    `bson:"leader"`
	LastHintTimestamp time.Time `bson:"last_hint_timestamp"`
	LastHintTypeSent  int       `bson:"last_hint_type_sent"`
}

// saveCategoryChatStateAsync asynchronously saves a chat state to MongoDB.
func saveCategoryChatStateAsync(chatID int64, state *ChatState) {
	state.RLock()
	doc := CategoryChatStateDoc{
		ChatID:            chatID,
		Word:              state.Word,
		User:              state.User,
		LeadTimestamp:     state.LeadTimestamp,
		Leader:            state.Leader,
		LastHintTimestamp: state.LastHintTimestamp,
		LastHintTypeSent:  state.LastHintTypeSent,
	}
	state.RUnlock()

	go func() {
		client := repository.DbManager()
		if client != nil {
			repository.SaveGameState(client, "CategoryChatStates", chatID, doc)
		}
	}()
}

// loadSavedCategoryChatStates loads states from MongoDB into the chatStates map
func loadSavedCategoryChatStates(client *mongo.Client) {
	var results []CategoryChatStateDoc
	err := repository.LoadAllGameStates(client, "CategoryChatStates", &results)
	if err != nil {
		log.Printf("Failed to load saved category chat states: %v", err)
		return
	}

	stateMutex.Lock()
	defer stateMutex.Unlock()

	for _, doc := range results {
		cs := &ChatState{
			Word:              doc.Word,
			User:              doc.User,
			LeadTimestamp:     doc.LeadTimestamp,
			Leader:            doc.Leader,
			LastHintTimestamp: doc.LastHintTimestamp,
			LastHintTypeSent:  doc.LastHintTypeSent,
		}
		chatStates[doc.ChatID] = cs
	}
	log.Printf("Loaded %d active Word Guess games from MongoDB (Category Bot)", len(results))
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

func createCategoryBotKeyboard(botUsername string, chatID int64) tgbotapi.InlineKeyboardMarkup {
	customWordLink := fmt.Sprintf("https://t.me/%s?start=custom_word_%d", botUsername, chatID)
	return tgbotapi.NewInlineKeyboardMarkup(
		// First line with a single button
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("See word 👀", "explain"),
			tgbotapi.NewInlineKeyboardButtonURL("Custom Word ✍️", customWordLink),
		),
		// Second line with multiple buttons
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("nxt⏭️", "next"),
			// tgbotapi.NewInlineKeyboardButtonData("flower❀", "flower"),
			tgbotapi.NewInlineKeyboardButtonData("car🏎️𖦹 ׂ 𓈒", "car"),
			// tgbotapi.NewInlineKeyboardButtonData("animal🐾", "animal"),
			tgbotapi.NewInlineKeyboardButtonData("AI Hint 💡", "ai_hint"),
		),
		// Third line with a single button
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Wordle 🟩🟨", "wordle_start"),
		),
		// Fourth line with a single button
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
func (cs *ChatState) reset(chatID int64) {
	cs.Lock()
	cs.Word = ""
	cs.User = 0
	cs.LeadTimestamp = time.Time{}
	cs.Leader = ""
	cs.Unlock()
	saveCategoryChatStateAsync(chatID, cs)
}

// StartBot initializes and starts the bot
func StartBot(token string) error {
	// Create a single MongoDB client instance once
	client := repository.DbManager()
	if client == nil {
		return fmt.Errorf("failed to connect to MongoDB")
	}

	loadSavedCategoryChatStates(client)
	wordlebot.LoadSavedStates(client)
	scramybot.LoadSavedStates(client)

	if err := wordlebot.LoadWordleWords(); err != nil {
		log.Printf("Warning: failed to load Wordle words: %v", err)
	}

	if err := scramybot.LoadScramyWords(); err != nil {
		log.Printf("Warning: failed to load Scramy words: %v", err)
	}

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

var customWordState = make(map[int64]int64)
var customWordMutex = &sync.Mutex{}

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
			args := message.CommandArguments()
			if args == "shop" {
				showShop(bot, message.Chat.ID)
				return
			} else if strings.HasPrefix(args, "custom_word_") {
				parts := strings.Split(args, "_")
				if len(parts) == 3 {
					groupChatIDStr := parts[2]
					var groupChatID int64
					if _, err := fmt.Sscanf(groupChatIDStr, "%d", &groupChatID); err == nil {
						groupState := getOrCreateChatState(groupChatID)
						groupState.RLock()
						isLeader := groupState.User == message.From.ID
						groupState.RUnlock()
						if isLeader {
							customWordMutex.Lock()
							customWordState[int64(message.From.ID)] = groupChatID
							customWordMutex.Unlock()
							view.SendMessage(bot, chatID, "Type the word you want the group to guess (must be a valid English word):")
						} else {
							view.SendMessage(bot, chatID, "You are not the current leader in that group.")
						}
					}
				}
				return
			} else {
				buttons := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Get Hint"+telegramReactions[20], "hint")))
				view.SendMessageWithButtons(bot, message.Chat.ID, "Heyyy! Got a word for ya 😏 Tap the button below if you need a lil hint 👇", buttons)
				return
			}
		case "wordle":
			wordlebot.HandleWordleCommand(bot, chatID, message.From.FirstName)
			return
		case "scramy":
			scramybot.HandleScramyCommand(bot, chatID, message.From.FirstName)
			return
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
			view.SendMessage(bot, chatID, "Group stats are not available in a DM. You can view global stats using /statsglobal or /leaderstatsglobal.")
		case "leaderstats":
			view.SendMessage(bot, chatID, "Group stats are not available in a DM. You can view global stats using /statsglobal or /leaderstatsglobal.")
		case "statsglobal":
			buttons := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Word Guess Global", "statsglobal_wordguess"), tgbotapi.NewInlineKeyboardButtonData("Wordle Global", "statsglobal_wordle")), tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Scramy Global", "statsglobal_scramy")))
			view.SendMessageWithButtons(bot, chatID, "🐊🇮🇳\n📊 Choose global stats to view:", buttons)
		case "statsimageglobal":
			markup := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Word Guess Image Global", "statsimg_global_wordguess"), tgbotapi.NewInlineKeyboardButtonData("Wordle Image Global", "statsimg_global_wordle")), tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Scramy Image Global", "statsimg_global_scramy")))
			imgBytes, err := service.GenerateLeaderboardImage(client, "CrocEn", 0, "Word Guess Global Leaderboard")
			if err == nil {
				photo := tgbotapi.NewPhotoUpload(chatID, tgbotapi.FileBytes{Name: "leaderboard.png", Bytes: imgBytes})
				photo.ReplyMarkup = markup
				bot.Send(photo)
			}
		case "leaderstatsglobal":
			result := service.LeaderBoardList(client, "CrocEnLeader", 0)
			view.SendMessagehtml(bot, message.Chat.ID, result)
		case "shop":
			if message.Chat.IsPrivate() {
				// Handle shop in DM
				showShop(bot, message.Chat.ID)
			} else {
				// Send a link to DM
				botUsername := bot.Self.UserName
				link := fmt.Sprintf("https://t.me/%s?start=shop", botUsername)
				markup := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonURL("🛒 Go to Shop", link),
					),
				)
				view.SendMessageWithButtons(bot, message.Chat.ID, "Click the button below to visit the Emoji Shop!", markup)
			}
		case "mystats":
			buttons := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Word Guess", "stats_wordguess"),
					tgbotapi.NewInlineKeyboardButtonData("Wordle", "stats_wordle"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Scramy", "stats_scramy"),
				),
			)
			view.SendMessageWithButtons(bot, chatID, "🐊🇮🇳\n📊 Choose game stats to view:", buttons)
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
		case "addwordlepoints":
			if message.From.ID != int(adminID) {
				return
			}
			parts := strings.Fields(message.Text)
			if len(parts) < 3 {
				view.SendMessage(bot, chatID, "Usage: /addwordlepoints <userID> <points> [name]")
				return
			}
			var userID int
			var points int
			if _, err := fmt.Sscanf(parts[1], "%d", &userID); err != nil {
				view.SendMessage(bot, chatID, "Invalid userID. Must be a number.")
				return
			}
			if _, err := fmt.Sscanf(parts[2], "%d", &points); err != nil {
				view.SendMessage(bot, chatID, "Invalid points. Must be a number.")
				return
			}
			name := "Unknown"
			if len(parts) > 3 {
				name = strings.Join(parts[3:], " ")
			}
			go repository.InsertWordleBonusDoc(userID, name, chatID, client, "WordleEn", points)
			view.SendMessage(bot, chatID, fmt.Sprintf("Added %d Wordle points for user %d (%s)", points, userID, name))
			return
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
			saveCategoryChatStateAsync(chatID, chatState)
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
			saveCategoryChatStateAsync(chatID, chatState)
			return
		}

		// Handle /reveal command in DM
		if message.Command() == "reveal" {
			if time.Since(chatState.LeadTimestamp) >= 6*time.Second {
				view.SendMessage(bot, chatID, fmt.Sprintf("The word was: %s", chatState.Word))
				chatState.reset(chatID)
			} else {
				sentMsg, err := view.SendMessage(bot, chatID, "Please try to read the hint before revealing the word.")
				deleteWarningMessage(bot, message, sentMsg, err)
			}
			return
		}

		// Check if user is in custom word state
		customWordMutex.Lock()
		groupChatID, ok := customWordState[int64(message.From.ID)]
		customWordMutex.Unlock()
		if ok && message.Chat.IsPrivate() {
			if message.Command() == "cancel" {
				customWordMutex.Lock()
				delete(customWordState, int64(message.From.ID))
				customWordMutex.Unlock()
				view.SendMessage(bot, chatID, "Custom word entry cancelled.")
				return
			}

			if !message.IsCommand() {
				cleanWord := strings.TrimSpace(message.Text)
				if validator.IsValidWord(cleanWord) {
					groupState := getOrCreateChatState(groupChatID)
					groupState.Lock()
					if groupState.User == message.From.ID {
						groupState.Word = strings.ToUpper(cleanWord)
						customWordMutex.Lock()
						delete(customWordState, int64(message.From.ID))
						customWordMutex.Unlock()
						view.SendMessage(bot, chatID, fmt.Sprintf("Your custom word '%s' has been set for the group!", cleanWord))
						// view.SendMessageWithEffectID(bot, chatID, fmt.Sprintf("Your custom word '%s' has been set for the group!", cleanWord), view.CustomWordMessageEffectID)
					} else {
						customWordMutex.Lock()
						delete(customWordState, int64(message.From.ID))
						customWordMutex.Unlock()
						view.SendMessage(bot, chatID, "You are no longer the leader of the group, so you cannot set the word.")
					}
					groupState.Unlock()
				} else {
					view.SendMessage(bot, chatID, "Invalid word. Please send a valid English word. Or type /cancel to abort.")
				}
				return
			}
		}

		// Check if Wordle is active for DM
		if wordlebot.IsWordleActive(chatID) {
			wordlebot.HandleGuess(bot, message, client, chatID, message.Text)
		}

		if scramybot.IsScramyActive(chatID) {
			scramybot.HandleGuess(bot, message, client, chatID, message.Text)
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

			chatState.reset(chatID)
			return
		}
		aiModeMutex.Lock()
		aiOn := aiModeUsers[chatID]
		aiModeMutex.Unlock()
		if aiOn && strings.Contains(text, "Jarvis") {
			// AI processing here
			wordChannel, errChannel := installOllama.RunOllama(text)

			// Send the initial message (could be an empty string or placeholder)
			initialMsg := tgbotapi.NewMessage(chatID, "Thinking...")
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
		fmt.Print(text, "escaped jarvis-----------------------------------------")
		return
	}

	// Existing group chat handling
	text := message.Text
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
		if message.CommandArguments() == "shop" {
			showShop(bot, message.Chat.ID)
		} else {
			view.SendMessage(bot, message.Chat.ID, "Welcome! Type /word to start a new game.")
		}
	case "setting":
		buttons := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Wordle View 🖼️", "setting_wordle_view"),
				tgbotapi.NewInlineKeyboardButtonData("Scramy Letters 🔠", "setting_scramy_letters"),
			),
		)
		view.SendMessageWithButtons(bot, message.Chat.ID, "⚙️ **Settings**\nChoose a setting to configure:", buttons)
		return
	case "wordle":
		wordlebot.HandleWordleCommand(bot, chatID, message.From.FirstName)
		return
	case "scramy":
		scramybot.HandleScramyCommand(bot, chatID, message.From.FirstName)
		return
	case "stats":
		buttons := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Word Guess Group", "statsgroup_wordguess"), tgbotapi.NewInlineKeyboardButtonData("Wordle Group", "statsgroup_wordle")), tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Scramy Group", "statsgroup_scramy")))
		view.SendMessageWithButtons(bot, chatID, "🐊🇮🇳\n📊 Choose group stats to view:", buttons)
	case "statsimage":
		markup := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Word Guess Image Group", "statsimg_group_wordguess"), tgbotapi.NewInlineKeyboardButtonData("Wordle Image Group", "statsimg_group_wordle")), tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Scramy Image Group", "statsimg_group_scramy")))
		imgBytes, err := service.GenerateLeaderboardImage(client, "CrocEn", chatID, "Word Guess Group Leaderboard")
		if err == nil {
			photo := tgbotapi.NewPhotoUpload(chatID, tgbotapi.FileBytes{Name: "leaderboard.png", Bytes: imgBytes})
			photo.ReplyMarkup = markup
			bot.Send(photo)
		}
	case "leaderstats":
		result := service.LeaderBoardList(client, "CrocEnLeader", message.Chat.ID)
		view.SendMessagehtml(bot, message.Chat.ID, result)
	case "shop":
		if message.Chat.IsPrivate() {
			// Handle shop in DM
			showShop(bot, message.Chat.ID)
		} else {
			// Send a link to DM
			botUsername := bot.Self.UserName
			link := fmt.Sprintf("https://t.me/%s?start=shop", botUsername)
			markup := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL("🛒 Go to Shop", link),
				),
			)
			view.SendMessageWithButtons(bot, message.Chat.ID, "Click the button below to visit the Emoji Shop!", markup)
		}
	case "statsglobal":
		buttons := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Word Guess Global", "statsglobal_wordguess"), tgbotapi.NewInlineKeyboardButtonData("Wordle Global", "statsglobal_wordle")), tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Scramy Global", "statsglobal_scramy")))
		view.SendMessageWithButtons(bot, chatID, "🐊🇮🇳\n📊 Choose global stats to view:", buttons)
	case "statsimageglobal":
		markup := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Word Guess Image Global", "statsimg_global_wordguess"), tgbotapi.NewInlineKeyboardButtonData("Wordle Image Global", "statsimg_global_wordle")), tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Scramy Image Global", "statsimg_global_scramy")))
		imgBytes, err := service.GenerateLeaderboardImage(client, "CrocEn", 0, "Word Guess Global Leaderboard")
		if err == nil {
			photo := tgbotapi.NewPhotoUpload(chatID, tgbotapi.FileBytes{Name: "leaderboard.png", Bytes: imgBytes})
			photo.ReplyMarkup = markup
			bot.Send(photo)
		}
	case "leaderstatsglobal":
		result := service.LeaderBoardList(client, "CrocEnLeader", 0)
		view.SendMessagehtml(bot, message.Chat.ID, result)
	case "mystats":
		buttons := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Word Guess", "stats_wordguess"),
				tgbotapi.NewInlineKeyboardButtonData("Wordle", "stats_wordle"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Scramy", "stats_scramy"),
			),
		)
		view.SendMessageWithButtons(bot, chatID, "🐊🇮🇳\n📊 Choose game stats to view:", buttons)
		// view.ReplyToMessage(bot, message.MessageID, chatID, result)
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
	case "addwordlepoints":
		if message.From.ID != int(adminID) {
			return
		}
		parts := strings.Fields(message.Text)
		if len(parts) < 3 {
			view.SendMessage(bot, chatID, "Usage: /addwordlepoints <userID> <points> [name]")
			return
		}
		var userID int
		var points int
		if _, err := fmt.Sscanf(parts[1], "%d", &userID); err != nil {
			view.SendMessage(bot, chatID, "Invalid userID. Must be a number.")
			return
		}
		if _, err := fmt.Sscanf(parts[2], "%d", &points); err != nil {
			view.SendMessage(bot, chatID, "Invalid points. Must be a number.")
			return
		}
		name := "Unknown"
		if len(parts) > 3 {
			name = strings.Join(parts[3:], " ")
		}
		go repository.InsertWordleBonusDoc(userID, name, chatID, client, "WordleEn", points)
		view.SendMessage(bot, chatID, fmt.Sprintf("Added %d Wordle points for user %d (%s)", points, userID, name))
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
	case "game":
		buttons := createMultiButtonKeyboard([][]string{
			{"Lead Croc 🗣️  ", "explain"},
			{"Wordle 🟩🟨", "wordle_start"},
			{"Scramy 𒅒𒈔𒅒", "scramy_start"},
		})
		view.SendMessageWithButtons(bot, chatID, "🐊 *The Crocodile peeks from the reeds, waiting...*", buttons)
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
			saveCategoryChatStateAsync(chatID, chatState)

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
			saveCategoryChatStateAsync(chatID, chatState)
	case "reveal":
		chatState.RLock()
		word := chatState.Word
		leadTime := chatState.LeadTimestamp
		chatState.RUnlock()

		if time.Since(leadTime) >= 600*time.Second {
			buttons := createSingleButtonKeyboard(" 🗣️ Explain ", "explain")
			view.SendMessageWithButtons(bot, message.Chat.ID, fmt.Sprintf("The word was: %s", word), buttons)

			chatState.reset(chatID)
		} else {
			sentMsg, err := view.SendMessage(bot, message.Chat.ID, "Please wait for 10 minutes before revealing the word.")
			deleteWarningMessage(bot, message, sentMsg, err)
		}
	default:
		// Check if Wordle is active for group chat
		if wordlebot.IsWordleActive(chatID) {
			wordlebot.HandleGuess(bot, message, client, chatID, message.Text)
		}

		if scramybot.IsScramyActive(chatID) {
			scramybot.HandleGuess(bot, message, client, chatID, message.Text)
		}

		chatState.RLock()
		word := chatState.Word
		user := chatState.User
		leader := chatState.Leader
		chatState.RUnlock()

		if user != 0 && service.NormalizeAndCompare(message.Text, word) && message.From.ID != user {
			chatState.reset(chatID)
			buttons := createSingleButtonKeyboard("🌟 Claim Leadership 🙋", "explain")
			view.SendMessageWithButtons(bot, message.Chat.ID, fmt.Sprintf("%s! %s guessed the word %s.\n /word", telegramReactions[7], message.From.FirstName, word), buttons)
			go view.ReactToMessage(bot.Token, chatID, message.MessageID, telegramReactions[rand.Intn(8)+13], true)
			go view.ReactToMessage(bot.Token, chatID, message.MessageID, telegramReactions[rand.Intn(8)+13], true)
			go repository.InsertDoc(message.From.ID, message.From.FirstName, message.Chat.ID, client, "CrocEn")
			go repository.InsertDoc(user, leader, message.Chat.ID, client, "CrocEnLeader")

		}
		aiModeMutex.Lock()
		aiOn := aiModeUsers[chatID]
		aiModeMutex.Unlock()
		if aiOn && strings.Contains(text, "Jarvis") {
			// AI processing here
			wordChannel, errChannel := installOllama.RunOllama(text)

			// Send the initial message (could be an empty string or placeholder)
			initialMsg := tgbotapi.NewMessage(chatID, "Thinking...")
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
		fmt.Print(text, "escaped jarvis-----------------------------------------")
		return
	}
}

// handleCallbackQuery processes incoming callback queries and handles the "explain" action.
func handleCallbackQuery(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, client *mongo.Client) {
	if handleShopCallback(bot, callback, client) {
		return
	}

	chatID := callback.Message.Chat.ID
	chatState := getOrCreateChatState(chatID)

	switch callback.Data {
	case "statsglobal_wordguess":
		markup := service.LeaderBoardListButtons(client, "CrocEn", 0, callback.Data)
		err := view.EditMessageTextWithStyledButtons(bot.Token, chatID, callback.Message.MessageID, "🏆 <b>Top 10 Players Leaderboard</b> 🏆\n\n✨ <b>Keep it up and aim for the top!</b> ✨", markup)
		if err != nil {
			log.Printf("Failed to send styled buttons message: %v", err)
			view.SendMessagehtml(bot, chatID, "Failed to load leaderboard.")
		}
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
		return
	case "statsglobal_wordle":
		markup := service.LeaderBoardListButtons(client, "WordleEn", 0, callback.Data)
		err := view.EditMessageTextWithStyledButtons(bot.Token, chatID, callback.Message.MessageID, "🏆 <b>Top 10 Players Leaderboard</b> 🏆\n\n✨ <b>Keep it up and aim for the top!</b> ✨", markup)
		if err != nil {
			log.Printf("Failed to send styled buttons message: %v", err)
			view.SendMessagehtml(bot, chatID, "Failed to load leaderboard.")
		}
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
		return
	case "statsglobal_scramy":
		markup := service.LeaderBoardListButtons(client, "ScramyEn", 0, callback.Data)
		err := view.EditMessageTextWithStyledButtons(bot.Token, chatID, callback.Message.MessageID, "🏆 <b>Top 10 Players Leaderboard</b> 🏆\n\n✨ <b>Keep it up and aim for the top!</b> ✨", markup)
		if err != nil {
			log.Printf("Failed to send styled buttons message: %v", err)
			view.SendMessagehtml(bot, chatID, "Failed to load leaderboard.")
		}
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
		return
	case "statsgroup_wordguess":
		markup := service.LeaderBoardListButtons(client, "CrocEn", chatID, callback.Data)
		err := view.EditMessageTextWithStyledButtons(bot.Token, chatID, callback.Message.MessageID, "🏆 <b>Top 10 Players Leaderboard</b> 🏆\n\n✨ <b>Keep it up and aim for the top!</b> ✨", markup)
		if err != nil {
			log.Printf("Failed to send styled buttons message: %v", err)
			view.SendMessagehtml(bot, chatID, "Failed to load leaderboard.")
		}
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
		return
	case "statsgroup_wordle":
		markup := service.LeaderBoardListButtons(client, "WordleEn", chatID, callback.Data)
		err := view.EditMessageTextWithStyledButtons(bot.Token, chatID, callback.Message.MessageID, "🏆 <b>Top 10 Players Leaderboard</b> 🏆\n\n✨ <b>Keep it up and aim for the top!</b> ✨", markup)
		if err != nil {
			log.Printf("Failed to send styled buttons message: %v", err)
			view.SendMessagehtml(bot, chatID, "Failed to load leaderboard.")
		}
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
		return
	case "statsgroup_scramy":
		markup := service.LeaderBoardListButtons(client, "ScramyEn", chatID, callback.Data)
		err := view.EditMessageTextWithStyledButtons(bot.Token, chatID, callback.Message.MessageID, "🏆 <b>Top 10 Players Leaderboard</b> 🏆\n\n✨ <b>Keep it up and aim for the top!</b> ✨", markup)
		if err != nil {
			log.Printf("Failed to send styled buttons message: %v", err)
			view.SendMessagehtml(bot, chatID, "Failed to load leaderboard.")
		}
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
		return
	case "statsimg_global_wordguess":
		markup := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Word Guess Image Global", "statsimg_global_wordguess"), tgbotapi.NewInlineKeyboardButtonData("Wordle Image Global", "statsimg_global_wordle")), tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Scramy Image Global", "statsimg_global_scramy")))
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, "Generating image..."))
		bot.Send(tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID, "Generating image... Please wait ⏳"))
		imgBytes, err := service.GenerateLeaderboardImage(client, "CrocEn", 0, "Word Guess Global Leaderboard")
		if err == nil {
			photo := tgbotapi.NewPhotoUpload(chatID, tgbotapi.FileBytes{Name: "leaderboard.png", Bytes: imgBytes})
			photo.ReplyMarkup = markup
			bot.Send(photo)
		} else {
			view.SendMessage(bot, chatID, "Failed to generate image.")
		}
		bot.Send(tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID))
		return
	case "statsimg_global_wordle":
		markup := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Word Guess Image Global", "statsimg_global_wordguess"), tgbotapi.NewInlineKeyboardButtonData("Wordle Image Global", "statsimg_global_wordle")), tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Scramy Image Global", "statsimg_global_scramy")))
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, "Generating image..."))
		bot.Send(tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID, "Generating image... Please wait ⏳"))
		imgBytes, err := service.GenerateLeaderboardImage(client, "WordleEn", 0, "Wordle Global Leaderboard")
		if err == nil {
			photo := tgbotapi.NewPhotoUpload(chatID, tgbotapi.FileBytes{Name: "leaderboard.png", Bytes: imgBytes})
			photo.ReplyMarkup = markup
			bot.Send(photo)
		} else {
			view.SendMessage(bot, chatID, "Failed to generate image.")
		}
		bot.Send(tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID))
		return
	case "statsimg_global_scramy":
		markup := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Word Guess Image Global", "statsimg_global_wordguess"), tgbotapi.NewInlineKeyboardButtonData("Wordle Image Global", "statsimg_global_wordle")), tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Scramy Image Global", "statsimg_global_scramy")))
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, "Generating image..."))
		bot.Send(tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID, "Generating image... Please wait ⏳"))
		imgBytes, err := service.GenerateLeaderboardImage(client, "ScramyEn", 0, "Scramy Global Leaderboard")
		if err == nil {
			photo := tgbotapi.NewPhotoUpload(chatID, tgbotapi.FileBytes{Name: "leaderboard.png", Bytes: imgBytes})
			photo.ReplyMarkup = markup
			bot.Send(photo)
		} else {
			view.SendMessage(bot, chatID, "Failed to generate image.")
		}
		bot.Send(tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID))
		return
	case "statsimg_group_wordguess":
		markup := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Word Guess Image Group", "statsimg_group_wordguess"), tgbotapi.NewInlineKeyboardButtonData("Wordle Image Group", "statsimg_group_wordle")), tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Scramy Image Group", "statsimg_group_scramy")))
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, "Generating image..."))
		bot.Send(tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID, "Generating image... Please wait ⏳"))
		imgBytes, err := service.GenerateLeaderboardImage(client, "CrocEn", chatID, "Word Guess Group Leaderboard")
		if err == nil {
			photo := tgbotapi.NewPhotoUpload(chatID, tgbotapi.FileBytes{Name: "leaderboard.png", Bytes: imgBytes})
			photo.ReplyMarkup = markup
			bot.Send(photo)
		} else {
			view.SendMessage(bot, chatID, "Failed to generate image.")
		}
		bot.Send(tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID))
		return
	case "statsimg_group_wordle":
		markup := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Word Guess Image Group", "statsimg_group_wordguess"), tgbotapi.NewInlineKeyboardButtonData("Wordle Image Group", "statsimg_group_wordle")), tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Scramy Image Group", "statsimg_group_scramy")))
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, "Generating image..."))
		bot.Send(tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID, "Generating image... Please wait ⏳"))
		imgBytes, err := service.GenerateLeaderboardImage(client, "WordleEn", chatID, "Wordle Group Leaderboard")
		if err == nil {
			photo := tgbotapi.NewPhotoUpload(chatID, tgbotapi.FileBytes{Name: "leaderboard.png", Bytes: imgBytes})
			photo.ReplyMarkup = markup
			bot.Send(photo)
		} else {
			view.SendMessage(bot, chatID, "Failed to generate image.")
		}
		bot.Send(tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID))
		return
	case "statsimg_group_scramy":
		markup := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Word Guess Image Group", "statsimg_group_wordguess"), tgbotapi.NewInlineKeyboardButtonData("Wordle Image Group", "statsimg_group_wordle")), tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Scramy Image Group", "statsimg_group_scramy")))
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, "Generating image..."))
		bot.Send(tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID, "Generating image... Please wait ⏳"))
		imgBytes, err := service.GenerateLeaderboardImage(client, "ScramyEn", chatID, "Scramy Group Leaderboard")
		if err == nil {
			photo := tgbotapi.NewPhotoUpload(chatID, tgbotapi.FileBytes{Name: "leaderboard.png", Bytes: imgBytes})
			photo.ReplyMarkup = markup
			bot.Send(photo)
		} else {
			view.SendMessage(bot, chatID, "Failed to generate image.")
		}
		bot.Send(tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID))
		return
	case "mystats_main":
		buttons := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Word Guess", "stats_wordguess"),
				tgbotapi.NewInlineKeyboardButtonData("Wordle", "stats_wordle"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Scramy", "stats_scramy"),
			),
		)
		editMsg := tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID, "🐊🇮🇳\n📊 Choose game stats to view:")
		editMsg.ReplyMarkup = &buttons
		bot.Send(editMsg)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
		return
	case "setting_scramy_letters":
		buttons := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Squared 🔠", "set_scramy_squared"),
				tgbotapi.NewInlineKeyboardButtonData("Normal abc", "set_scramy_normal"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "settings_main"),
			),
		)
		editMsg := tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID, "⚙️ **Scramy Letters Setting**\nChoose the letter style for Scramy:")
		editMsg.ReplyMarkup = &buttons
		editMsg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(editMsg)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
		return
	case "setting_scramy_letters_new":
		buttons := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Squared 🔠", "set_scramy_squared_new"),
				tgbotapi.NewInlineKeyboardButtonData("Normal abc", "set_scramy_normal_new"),
			),
		)
		editMsg := tgbotapi.NewEditMessageReplyMarkup(chatID, callback.Message.MessageID, buttons)
		bot.Send(editMsg)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
		return
	case "set_scramy_squared":
		scramybot.UpdateScramyLetterView(chatID, "squared", client)
		editMsg := tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID, "✅ Scramy letters updated to **Squared**.")
		editMsg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(editMsg)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, "Scramy set to Squared"))
		return
	case "set_scramy_normal":
		scramybot.UpdateScramyLetterView(chatID, "normal", client)
		editMsg := tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID, "✅ Scramy letters updated to **Normal**.")
		editMsg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(editMsg)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, "Scramy set to Normal"))
		return
	case "stats_wordguess":
		result := service.GetUserStatsByID(client, callback.From.ID)
		buttons := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "mystats_main")))
		editMsg := tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID, result)
		editMsg.ReplyMarkup = &buttons
		editMsg.ParseMode = tgbotapi.ModeHTML
		bot.Send(editMsg)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
		return
	case "stats_scramy":
		result := service.GetScramyUserStatsByID(client, callback.From.ID)
		buttons := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "mystats_main")))
		editMsg := tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID, result)
		editMsg.ReplyMarkup = &buttons
		editMsg.ParseMode = tgbotapi.ModeHTML
		bot.Send(editMsg)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
		return
	case "stats_wordle":
		result := service.GetWordleUserStatsByID(client, callback.From.ID)
		buttons := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "mystats_main")))
		editMsg := tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID, result)
		editMsg.ReplyMarkup = &buttons
		editMsg.ParseMode = tgbotapi.ModeHTML
		bot.Send(editMsg)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
		return
	case "explain":
		chatState.Lock()
		if chatState.User != callback.From.ID && chatState.User != 0 && time.Since(chatState.LeadTimestamp) < 600*time.Second {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, fmt.Sprintf("%s is already explaining the word. Please wait for your turn, %s.", chatState.Leader, callback.From.UserName)))
			chatState.Unlock()
			saveCategoryChatStateAsync(chatID, chatState)
			return
		}
		if chatState.User == callback.From.ID {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, chatState.Word))
			chatState.Unlock()
			saveCategoryChatStateAsync(chatID, chatState)
			return
		}
		if chatState.User == 0 || (time.Since(chatState.LeadTimestamp) >= 600*time.Second && chatState.User != callback.From.ID) {
			chatState.User = callback.From.ID
			word, err := model.GetRandomWord()
			if err != nil {
				chatState.Unlock()
			saveCategoryChatStateAsync(chatID, chatState)
				return
			}
			buttons := createCategoryBotKeyboard(bot.Self.UserName, chatID)
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
			saveCategoryChatStateAsync(chatID, chatState)
		bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, chatState.Word))
	case "settings_main":
		buttons := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Wordle View 🖼️", "setting_wordle_view"),
				tgbotapi.NewInlineKeyboardButtonData("Scramy Letters 🔠", "setting_scramy_letters"),
			),
		)
		editMsg := tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID, "⚙️ *Settings*\nChoose a setting to configure:")
		editMsg.ReplyMarkup = &buttons
		editMsg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(editMsg)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
		return
	case "setting_wordle_view":
		buttons := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Text View 📝", "set_wordle_view_text"),
				tgbotapi.NewInlineKeyboardButtonData("Image View 🖼️", "set_wordle_view_image"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "settings_main"),
			),
		)
		editMsg := tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID, "⚙️ **Wordle View Setting**\nChoose how you want Wordle results to be displayed:")
		editMsg.ReplyMarkup = &buttons
		editMsg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(editMsg)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
		return
	case "setting_wordle_view_new":
		buttons := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Text View 📝", "set_wordle_view_text_new"),
				tgbotapi.NewInlineKeyboardButtonData("Image View 🖼️", "set_wordle_view_image_new"),
			),
		)
		editMsg := tgbotapi.NewEditMessageReplyMarkup(chatID, callback.Message.MessageID, buttons)
		bot.Send(editMsg)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
		return
	case "set_wordle_view_text":
		wordlebot.UpdateWordleViewType(chatID, "text", client)
		editMsg := tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID, "✅ Wordle view updated to **Text**.")
		editMsg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(editMsg)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, "View set to Text"))
		return
	case "set_wordle_view_image":
		wordlebot.UpdateWordleViewType(chatID, "image", client)
		editMsg := tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID, "✅ Wordle view updated to **Image**.")
		editMsg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(editMsg)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, "View set to Image"))
		return
	case "set_scramy_squared_new":
		scramybot.UpdateScramyLetterView(chatID, "squared", client)
		scramybot.RefreshActiveGameMessage(bot, chatID, callback.Message.MessageID, client)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, "Scramy set to Squared"))
		return
	case "set_scramy_normal_new":
		scramybot.UpdateScramyLetterView(chatID, "normal", client)
		scramybot.RefreshActiveGameMessage(bot, chatID, callback.Message.MessageID, client)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, "Scramy set to Normal"))
		return
	case "set_wordle_view_text_new":
		wordlebot.UpdateWordleViewType(chatID, "text", client)
		wordlebot.RefreshActiveGameMessage(bot, chatID, callback.Message.MessageID, client)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, "View set to Text"))
		return
	case "set_wordle_view_image_new":
		wordlebot.UpdateWordleViewType(chatID, "image", client)
		wordlebot.RefreshActiveGameMessage(bot, chatID, callback.Message.MessageID, client)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, "View set to Image"))
		return
	case "wordle_start":
		wordlebot.HandleWordleCommand(bot, chatID, callback.From.FirstName)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, "Wordle Started!"))
		return
	case "cancel_new_wordle":
		if wordlebot.CancelPendingGame(bot, chatID, callback.From.FirstName) {
			bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, "Cancelled new game request."))
		} else {
			bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, "No pending game request to cancel."))
		}
		return
	case "scramy_start":
		scramybot.HandleScramyCommand(bot, chatID, callback.From.FirstName)
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, "Scramy Started!"))
		return
	case "cancel_new_scramy":
		if scramybot.CancelPendingGame(bot, chatID, callback.From.FirstName) {
			bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, "Cancelled new Scramy game request."))
		} else {
			bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, "No pending game request to cancel."))
		}
		return
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
			word = replaceWord(word, chatState.Word)
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
			saveCategoryChatStateAsync(chatID, chatState)
			return
		}
		if chatState.User == 0 {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, fmt.Sprintf("%s Please click on see word/claim Leadership", callback.From.FirstName)))
			chatState.Unlock()
			saveCategoryChatStateAsync(chatID, chatState)
			return
		}
		chatState.User = callback.From.ID
		word, err := model.GetRandomAnimal()
		if err != nil {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, "Failed to get an animal word. Please try again later."))
			chatState.Unlock()
			saveCategoryChatStateAsync(chatID, chatState)
			return
		}
		chatState.Word = word
		chatState.Unlock()
		saveCategoryChatStateAsync(chatID, chatState)
		bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, chatState.Word))

	case "next":
		chatState.Lock()
		if chatState.User != callback.From.ID && chatState.User != 0 {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, fmt.Sprintf("%s is already explaining the word. %s", chatState.Leader, callback.From.UserName)))
			chatState.Unlock()
			saveCategoryChatStateAsync(chatID, chatState)
			return
		}
		if chatState.User == 0 {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, fmt.Sprintf("%s Please click on see word/claim Leadership", callback.From.FirstName)))
			chatState.Unlock()
			saveCategoryChatStateAsync(chatID, chatState)
			return
		}
		chatState.User = callback.From.ID
		chatState.Leader = callback.From.FirstName
		chatState.Word, _ = model.GetRandomWord()
		chatState.Unlock()
		saveCategoryChatStateAsync(chatID, chatState)
		bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, chatState.Word))

	case "flower":
		chatState.Lock()
		if chatState.User != callback.From.ID && chatState.User != 0 {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, fmt.Sprintf("someone is already explaining the word. %s", callback.From.UserName)))
			chatState.Unlock()
			saveCategoryChatStateAsync(chatID, chatState)
			return
		}
		if chatState.User == 0 {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, fmt.Sprintf("%s Please click on see word/claim Leadership", callback.From.FirstName)))
			chatState.Unlock()
			saveCategoryChatStateAsync(chatID, chatState)
			return
		}
		chatState.User = callback.From.ID
		chatState.Word, _ = model.GetRandomFlower()
		chatState.Unlock()
		saveCategoryChatStateAsync(chatID, chatState)
		bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, chatState.Word))

	case "car":
		chatState.Lock()
		if chatState.User != callback.From.ID && chatState.User != 0 {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, fmt.Sprintf("someone is already explaining the word. %s", callback.From.UserName)))
			chatState.Unlock()
			saveCategoryChatStateAsync(chatID, chatState)
			return
		}
		if chatState.User == 0 {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, fmt.Sprintf("%s Please click on see word/claim Leadership", callback.From.FirstName)))
			chatState.Unlock()
			saveCategoryChatStateAsync(chatID, chatState)
			return
		}
		chatState.User = callback.From.ID
		chatState.Word, _ = model.GetRandomCar()
		chatState.Unlock()
		saveCategoryChatStateAsync(chatID, chatState)
		bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, chatState.Word))

	case "droplead":
		chatState.Lock()
		if chatState.User != callback.From.ID {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, "You are not the current leader, so you cannot drop the lead!"))
			chatState.Unlock()
			saveCategoryChatStateAsync(chatID, chatState)
			return
		}
		// Delete the callback message when user selects "Changed my mind"
		deleteMsg := tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID)
		_, err := bot.DeleteMessage(deleteMsg)
		if err != nil {
			log.Printf("Failed to delete message on droplead: %v", err)
		}
		chatState.Unlock()
			saveCategoryChatStateAsync(chatID, chatState)
		buttons := createSingleButtonKeyboard("🌟 Claim Leadership 🙋", "explain")
		view.SendMessageWithButtons(bot, callback.Message.Chat.ID, fmt.Sprintf("%s refused to lead -> %s \n", callback.From.FirstName, chatState.Word), buttons)
		chatState.reset(chatID)

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
			saveCategoryChatStateAsync(chatID, chatState)

		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
	default:
		chatState.RLock()
		word := chatState.Word
		chatState.RUnlock()
		if service.NormalizeAndCompare(callback.Message.Text, word) {
			buttons := createSingleButtonKeyboard("🌟 Claim Leadership 🙋", "explain")
			view.SendMessageWithButtons(bot, callback.Message.Chat.ID, fmt.Sprintf("%s! %s guessed the word correctly.", telegramReactions[0], callback.From.FirstName), buttons)
			chatState.reset(chatID)
		}
	}
	bot.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
}

// createMultiButtonKeyboard creates an inline keyboard markup with multiple buttons
func createMultiButtonKeyboard(buttonsData [][]string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, rowData := range buttonsData {
		var row []tgbotapi.InlineKeyboardButton
		for i := 0; i < len(rowData); i += 2 {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(rowData[i], rowData[i+1]))
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(row...))
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}
