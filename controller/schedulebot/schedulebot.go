package schedulebot

import (
	"log"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/robfig/cron/v3"
)

var (
	botInstance *tgbotapi.BotAPI
	cronParser  *cron.Cron
)

// StartScheduleBot initializes and starts the schedule bot
func StartScheduleBot(token string) error {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return err
	}

	botInstance = bot
	bot.Debug = true
	log.Printf("ScheduleBot authorized on account %s", bot.Self.UserName)

	client := repository.DbManager()
	if client == nil {
		log.Fatal("Failed to connect to MongoDB for ScheduleBot")
	}

	// Initialize cron with timezone support, seconds optional but standard cron parsing.
	// standard parser is used to support "0 10 * * *"
	cronParser = cron.New()
	cronParser.Start()

	// Load existing schedules from database and attach to cron
	loadSchedulesFromDB(client)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		return err
	}

	for update := range updates {
		if update.Message != nil {
			go handleMessage(bot, update.Message, client)
		}
	}

	return nil
}
