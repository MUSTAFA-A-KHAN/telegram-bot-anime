package scramybot

import (
	"context"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ChatSettings struct {
	ChatID           int64  `bson:"_id"`
	ScramyLetterView string `bson:"scramy_letter_view"` // "squared" or "normal"
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
		return settings
	}

	// Default settings
	settings = &ChatSettings{
		ChatID:           chatID,
		ScramyLetterView: "squared", // Default to squared
	}

	if client != nil {
		collection := client.Database("TelegramBot").Collection("ScramySettings")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := collection.FindOne(ctx, bson.M{"_id": chatID}).Decode(settings)
		if err != nil && err != mongo.ErrNoDocuments {
			// Log error if needed
		}
	}

	settingsMutex.Lock()
	settingsCache[chatID] = settings
	settingsMutex.Unlock()

	return settings
}

func UpdateScramyLetterView(chatID int64, viewType string, client *mongo.Client) error {
	settings := GetChatSettings(chatID, client)

	settingsMutex.Lock()
	settings.ScramyLetterView = viewType
	settingsCache[chatID] = settings
	settingsMutex.Unlock()

	if client != nil {
		collection := client.Database("TelegramBot").Collection("ScramySettings")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		opts := options.Update().SetUpsert(true)
		update := bson.M{"$set": bson.M{"scramy_letter_view": viewType}}
		_, err := collection.UpdateOne(ctx, bson.M{"_id": chatID}, update, opts)
		return err
	}
	return nil
}
