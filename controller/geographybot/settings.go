package geographybot

import (
	"context"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ChatSettings struct {
	ChatID        int64  `bson:"_id"`
	GeographyMode string `bson:"geography_mode"` // "mcq" or "text"
}

var (
	settingsCache = make(map[int64]*ChatSettings)
	settingsMutex sync.RWMutex
)

func GetChatSettings(chatID int64, client *mongo.Client) *ChatSettings {
	settingsMutex.RLock()
	settings, ok := settingsCache[chatID]
	settingsMutex.RUnlock()

	if ok {
		// Return a copy to prevent data races on concurrent field reads/writes
		copySettings := *settings
		return &copySettings
	}

	// Default settings
	settings = &ChatSettings{
		ChatID:        chatID,
		GeographyMode: "mcq", // Default to mcq
	}

	if client != nil {
		collection := client.Database("TelegramBot").Collection("GeographySettings")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := collection.FindOne(ctx, bson.M{"_id": chatID}).Decode(settings)
		if err != nil && err != mongo.ErrNoDocuments {
			// Log error if needed
		}
	}

	settingsMutex.Lock()
	// Store a copy in the cache
	cacheSettings := *settings
	settingsCache[chatID] = &cacheSettings
	settingsMutex.Unlock()

	return settings
}

func UpdateGeographyMode(chatID int64, mode string, client *mongo.Client) error {
	settings := GetChatSettings(chatID, client)
	settings.GeographyMode = mode

	settingsMutex.Lock()
	// Store a copy in the cache
	cacheSettings := *settings
	settingsCache[chatID] = &cacheSettings
	settingsMutex.Unlock()

	if client != nil {
		collection := client.Database("TelegramBot").Collection("GeographySettings")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		opts := options.Update().SetUpsert(true)
		update := bson.M{"$set": bson.M{"geography_mode": mode}}
		_, err := collection.UpdateOne(ctx, bson.M{"_id": chatID}, update, opts)
		return err
	}
	return nil
}
