package modbot

import (
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
)

// StartModBot initializes and starts the moderator bot
func StartModBot(token string) error {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return err
	}

	bot.Debug = true
	log.Printf("ModBot authorized on account %s", bot.Self.UserName)

	client := repository.DbManager()
	if client == nil {
		log.Fatal("Failed to connect to MongoDB for ModBot")
	}

	// Load settings from DB
	loadSettings(client)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		return err
	}

	for update := range updates {
		if update.Message != nil {
			go handleMessage(bot, update.Message, client)
		} else if update.EditedMessage != nil {
			// Apply filters to edited messages to prevent filter bypassing
			go handleFilters(bot, update.EditedMessage, client)
		} else if update.CallbackQuery != nil {
			go handleCallbackQuery(bot, update.CallbackQuery, client)
		}
	}

	return nil
}
