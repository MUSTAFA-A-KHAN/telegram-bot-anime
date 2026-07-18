package config

import (
	"log"
)

type Config struct {
	CatTelegramToken string
	MongoURI         string
	DatabaseName     string
}

var App Config

func Load(token string) {
	App = Config{
		CatTelegramToken: token,
	}

	if App.CatTelegramToken == "" {
		log.Fatal("BOT_TOKEN is not set")
	}
}
