package categorybot

import (
	"container/heap"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/model"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/service"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/view"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"go.mongodb.org/mongo-driver/mongo"
)

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
	cs.LastHintTimestamp = time.Time{}
	cs.LastHintTypeSent = 0
}

func StartBot(token string) error {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return err
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

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

	// Initialize priority queue for leaderboard
	pq := &service.PriorityQueue{}
	heap.Init(pq)

	// Map to keep track of user scores for quick updates
	userScores := make(map[int]*service.Item)

	for update := range updates {
		if update.Message != nil {
			go handleMessage(client, bot, update.Message, pq, userScores)
		} else if update.CallbackQuery != nil {
			go handleCallbackQuery(client, bot, update.CallbackQuery, pq, userScores)
		}
	}

	return nil
}

func handleMessage(client *mongo.Client, bot *tgbotapi.BotAPI, message *tgbotapi.Message, pq *service.PriorityQueue, userScores map[int]*service.Item) {
	chatID := message.Chat.ID
	adminID := int64(1006461736)

	chatState := getOrCreateChatState(chatID)

	log.Printf("[%s] %s", message.From.UserName, message.Text)

	switch message.Command() {
	case "start":
		view.SendMessage(bot, message.Chat.ID, "Welcome! Type /word to start a new game.")
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
	case "stats":
		result := service.LeaderBoardList(client, "CrocEn")
		view.SendMessage(bot, message.Chat.ID, result)
	case "leaderstats":
		result := service.LeaderBoardList(client, "CrocEnLeader")
		view.SendMessage(bot, message.Chat.ID, result)
	case "report":
		if len(message.Text) > 7 {
			reportMessage := message.Text[7:]
			adminMessage := fmt.Sprintf("Report from @%s (%d):\n%s", message.From.UserName, message.From.ID, reportMessage)
			view.SendMessage(bot, adminID, adminMessage)
			view.SendMessage(bot, chatID, "Thank you! Your report has been submitted.")
		} else {
			view.SendMessage(bot, chatID, "Please provide a message with your report. Usage: /report [your message]")
		}
	case "word":
		word, err := model.GetRandomWord()
		if err != nil {
			view.SendMessage(bot, message.Chat.ID, "Oops! Couldn't fetch a word. Please try again later.")
			return
		}
		buttons := createSingleButtonKeyboard(" 🗣️ Explain ", "explain")
		chatState.Lock()
		chatState.Word = word
		chatState.User = 0
		chatState.Unlock()
		view.SendSticker(bot, chatID, "CAACAgUAAxkBAAEwCnNnYW-OkgV7Odt9osVwoBSzLC6vsAACMhMAAj45CFdCstMoIYiPfjYE")
		view.SendMessageWithButtons(bot, message.Chat.ID, fmt.Sprintf("The word is ready! Click 'Explain' to explain the word."), buttons)
	case "hint":
		chatState.RLock()
		wordEmpty := chatState.Word == ""
		lastHint := chatState.LastHintTimestamp
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
		if chatState.LastHintTypeSent == 0 {
			hint = model.GenerateMeaningHint(chatState.Word)
		} else {
			hint = model.GenerateMeaningHint(chatState.Word)
			hint = hint + "\n" + model.GenerateHint(chatState.Word)
			hint = hint + "\n" + model.GenerateAuroraHint(chatState.Word)
		}
		chatState.RUnlock()

		chatAction := tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping)
		bot.Send(chatAction)

		escapedHint := escapeMarkdownV2(hint)
		spoilerHint := "||" + escapedHint + "||"
		msg := tgbotapi.NewMessage(chatID, spoilerHint)
		msg.ParseMode = "MarkdownV2"
		_, err := bot.Send(msg)
		if err != nil {
			log.Printf("Failed to send hint message with spoiler formatting: %v", err)
		}

		chatState.Lock()
		chatState.LastHintTimestamp = time.Now()
		chatState.LastHintTypeSent = 1 - chatState.LastHintTypeSent
		chatState.Unlock()

	default:
		chatState.RLock()
		word := chatState.Word
		user := chatState.User
		chatState.RUnlock()

		if user != 0 && service.NormalizeAndCompare(message.Text, word) && message.From.ID != user {
			buttons := createSingleButtonKeyboard("🌟 Claim Leadership 🙋", "explain")
			view.SendMessageWithButtons(bot, message.Chat.ID, fmt.Sprintf("Congratulations! %s guessed the word %s.\n /word", message.From.FirstName, chatState.Word), buttons)
			repository.InsertDoc(message.From.ID, message.From.FirstName, message.Chat.ID, client, "CrocEn")
			repository.InsertDoc(chatState.User, chatState.Leader, message.Chat.ID, client, "CrocEnLeader")
			chatState.Lock()
			chatState.reset()
			chatState.Unlock()
		}
	}
}

// handleCallbackQuery processes incoming callback queries and handles the "explain" action.
func handleCallbackQuery(client *mongo.Client, bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, pq *service.PriorityQueue, userScores map[int]*service.Item) {
	chatID := callback.Message.Chat.ID

	chatState := getOrCreateChatState(chatID)

	switch callback.Data {
	case "explain":
		chatState.Lock()
		if chatState.User != callback.From.ID && chatState.User != 0 && chatState.isLeaderActive(120*time.Second) {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, fmt.Sprintf("someone already explaining the word. %s", callback.From.UserName)))
			chatState.Unlock()
			return
		}
		if chatState.User == callback.From.ID {
			bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, chatState.Word))
		}
		if chatState.User == 0 || !chatState.isLeaderActive(120*time.Second) && chatState.User != callback.From.ID {
			word, err := model.GetRandomWord()
			if err != nil {
				chatState.Unlock()
				return
			}
			buttons := createSingleButtonKeyboard("See word 👀", "explain")
			chatState.Word = word
			view.SendMessageWithButtons(bot, callback.Message.Chat.ID, fmt.Sprintf(" [%s](tg://user?id=%d) is explaining the word!", callback.From.FirstName, callback.From.ID), buttons)
		}
		chatState.User = callback.From.ID
		chatState.Leader = callback.From.FirstName
		chatState.LeadTimestamp = time.Now()
		chatState.Unlock()
		bot.AnswerCallbackQuery(tgbotapi.NewCallbackWithAlert(callback.ID, chatState.Word))

	case "next":
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
