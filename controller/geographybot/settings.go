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
	ChatID        int64           `bson:"_id"`
	GeographyMode string          `bson:"geography_mode"` // "mcq" or "text"
	QuestionTypes map[string]bool `bson:"question_types"` // which question types are enabled
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
		QuestionTypes: map[string]bool{
			"capital":              true,
			"flag":                 true,
			"region":               true,
			"landmark":             true,
			"country_from_capital": true,
			"landmark_name":        true,
		},
	}

	if client != nil {
		collection := client.Database("TelegramBot").Collection("GeographySettings")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := collection.FindOne(ctx, bson.M{"_id": chatID}).Decode(settings)
		if err != nil && err != mongo.ErrNoDocuments {
			// Log error if needed
		}

		// Ensure map is initialized if missing from DB
		if settings.QuestionTypes == nil {
			settings.QuestionTypes = map[string]bool{
				"capital":              true,
				"flag":                 true,
				"region":               true,
				"landmark":             true,
				"country_from_capital": true,
				"landmark_name":        true,
			}
		} else {
			// Ensure new question types are added to existing DB records
			defaults := []string{"capital", "flag", "region", "landmark", "country_from_capital", "landmark_name"}
			for _, def := range defaults {
				if _, exists := settings.QuestionTypes[def]; !exists {
					settings.QuestionTypes[def] = true
				}
			}
		}
	}

	settingsMutex.Lock()
	// Store a copy in the cache
	cacheSettings := *settings
	settingsCache[chatID] = &cacheSettings
	settingsMutex.Unlock()

	return settings
}

func ToggleGeographyQuestionType(chatID int64, qType string, client *mongo.Client) error {
	settings := GetChatSettings(chatID, client)

	// Toggle value
	if val, exists := settings.QuestionTypes[qType]; exists {
		settings.QuestionTypes[qType] = !val
	} else {
		settings.QuestionTypes[qType] = true
	}

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
		update := bson.M{"$set": bson.M{"question_types": settings.QuestionTypes}}
		_, err := collection.UpdateOne(ctx, bson.M{"_id": chatID}, update, opts)
		return err
	}
	return nil
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
