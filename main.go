package main

import (
	"log"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/controller"
)

func main() {
	botToken := "6898558980:AAFIMHCvgfDugzk8NSxVPoZ-TKMNyahVPOI"
	err := controller.StartBot(botToken)
	if err != nil {
		log.Panic("error strting bot: ", err)
	}
}
