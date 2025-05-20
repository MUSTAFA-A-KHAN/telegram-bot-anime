package instagrambot

import (
	"fmt"
	"log"
	"strings"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/service"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/view"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func StartBot(token string) error {
	// Create a new instance of the bot using the provided token.
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return err
	}

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
		}
	}

	return nil
}

func handleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID

	// Expect a username from the user
	if strings.HasPrefix(message.Text, "/getinfo ") {
		username := strings.TrimSpace(strings.TrimPrefix(message.Text, "/getinfo "))
		if username == "" {
view.SendMessage(bot, chatID, "Please provide a valid Instagram username. Format: /getinfo <username>")
			return
		}

		info, err := service.GetInstagramUserInfo(username)
		if err != nil {
view.SendMessage(bot, chatID, "Sorry, we couldn't retrieve the user information. Please verify the username and try again. "+err.Error())
			log.Println("Error fetching Instagram user info:", err)
			return
		}
		client := repository.DbManager()
		repository.InsertUserInfo(*info, client)
		// Format the response with the user's information
		response := fmt.Sprintf(
			"Name: %s\nBio: %s\nFollowers: %d\n",
			info.Data.User.FullName,
			info.Data.User.Biography,
			info.Data.User.EdgeFollowedBy.Count,
		)

		// Check for the latest video
		if len(info.Data.User.EdgeOwnerToTimelineMedia.Edges) > 0 {
			for _, edge := range info.Data.User.EdgeOwnerToTimelineMedia.Edges {
				if edge.Node.IsVideo && edge.Node.VideoURL != "" {
					response += fmt.Sprintf("Latest Video: %s\n", edge.Node.VideoURL)
					break
				}
			}
		} else {
response += "No videos found for this user.\n"
		}

		// Send the response to the user
		view.SendMessage(bot, chatID, response)
	} else {
view.SendMessage(bot, chatID, "Invalid command. Please use /getinfo <username> to fetch Instagram user information.")
	}
}
