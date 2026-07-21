package modbot

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ModRuleDoc represents a single auto-responder rule
type ModRuleDoc struct {
	TriggerWord    string `bson:"trigger_word"`
	ResponseType   string `bson:"response_type"` // "text", "photo", "video", "document", "audio", "animation"
	ResponseText   string `bson:"response_text,omitempty"`
	ResponseFileID string `bson:"response_file_id,omitempty"`
}

// ModChatSettings represents the moderator settings for a specific chat
type ModChatSettings struct {
	ChatID        int64                 `bson:"_id"`
	BlockLinks    bool                  `bson:"block_links"`
	ScamDetection bool                  `bson:"scam_detection"`
	ScamKeywords  []string              `bson:"scam_keywords"`
	AllowedDomains []string             `bson:"allowed_domains"`
	Rules         map[string]ModRuleDoc `bson:"rules"` // TriggerWord as key for O(1) lookups
}

// UserViolationDoc tracks infractions for a user in a specific chat
type UserViolationDoc struct {
	ID        string    `bson:"_id"` // Composite key: fmt.Sprintf("%d_%d", chatID, userID)
	ChatID    int64     `bson:"chat_id"`
	UserID    int       `bson:"user_id"`
	Count     int       `bson:"count"`
	UpdatedAt time.Time `bson:"updated_at"`
}

var (
	settingsCache = make(map[int64]*ModChatSettings)
	settingsMutex sync.RWMutex
	violationsCache = make(map[string]*UserViolationDoc)
	violationsMutex sync.RWMutex
)

// GetChatSettings retrieves the settings for a chat (from cache or creates new)
func GetChatSettings(chatID int64) *ModChatSettings {
	settingsMutex.RLock()
	settings, exists := settingsCache[chatID]
	settingsMutex.RUnlock()

	if exists {
		// Return a copy to avoid data races when reading/writing concurrently
		copySettings := &ModChatSettings{
			ChatID:         settings.ChatID,
			BlockLinks:     settings.BlockLinks,
			ScamDetection:  settings.ScamDetection,
			ScamKeywords:   append([]string(nil), settings.ScamKeywords...),
			AllowedDomains: append([]string(nil), settings.AllowedDomains...),
			Rules:          make(map[string]ModRuleDoc),
		}
		for k, v := range settings.Rules {
			copySettings.Rules[k] = v
		}
		return copySettings
	}

	// Create default settings
	newSettings := &ModChatSettings{
		ChatID:         chatID,
		BlockLinks:     false,
		ScamDetection:  false,
		ScamKeywords:   []string{"paid survey", "crypto research"}, // Default scam words
		AllowedDomains: []string{"youtube.com", "wikipedia.org", "youtu.be"}, // Default allowed domains
		Rules:          make(map[string]ModRuleDoc),
	}

	settingsMutex.Lock()
	settingsCache[chatID] = newSettings
	settingsMutex.Unlock()

	copySettings := &ModChatSettings{
		ChatID:         newSettings.ChatID,
		BlockLinks:     newSettings.BlockLinks,
		ScamDetection:  newSettings.ScamDetection,
		ScamKeywords:   append([]string(nil), newSettings.ScamKeywords...),
		AllowedDomains: append([]string(nil), newSettings.AllowedDomains...),
		Rules:          make(map[string]ModRuleDoc),
	}
	return copySettings
}

// SaveChatSettings saves settings to MongoDB and updates cache
func SaveChatSettings(client *mongo.Client, settings *ModChatSettings) {
	settingsMutex.Lock()
	settingsCache[settings.ChatID] = settings
	settingsMutex.Unlock()

	if client == nil {
		return
	}

	go func() {
		collection := client.Database("Telegram").Collection("ModSettings")
		filter := bson.M{"_id": settings.ChatID}
		update := bson.M{"$set": settings}
		opts := options.Update().SetUpsert(true)

		_, err := collection.UpdateOne(context.TODO(), filter, update, opts)
		if err != nil {
			log.Printf("Failed to save mod settings for chat %d: %v", settings.ChatID, err)
		}
	}()
}

func loadSettings(client *mongo.Client) {
	if client == nil {
		return
	}

	// Load Settings
	collection := client.Database("Telegram").Collection("ModSettings")
	cursor, err := collection.Find(context.TODO(), bson.M{})
	if err != nil {
		log.Printf("Failed to load mod settings: %v", err)
		return
	}
	defer cursor.Close(context.TODO())

	var results []ModChatSettings
	if err = cursor.All(context.TODO(), &results); err != nil {
		log.Printf("Failed to decode mod settings: %v", err)
		return
	}

	settingsMutex.Lock()
	for _, s := range results {
		// Fix potentially nil maps/slices
		if s.Rules == nil {
			s.Rules = make(map[string]ModRuleDoc)
		}
		if s.ScamKeywords == nil {
			s.ScamKeywords = []string{"paid survey", "crypto research"}
		}
		if s.AllowedDomains == nil {
			s.AllowedDomains = []string{"youtube.com", "wikipedia.org", "youtu.be"}
		}

		copyS := s
		settingsCache[s.ChatID] = &copyS
	}
	settingsMutex.Unlock()
	log.Printf("Loaded %d ModBot chat settings from MongoDB", len(results))

	// Load Violations
	vCollection := client.Database("Telegram").Collection("ModViolations")
	vCursor, err := vCollection.Find(context.TODO(), bson.M{})
	if err != nil {
		log.Printf("Failed to load mod violations: %v", err)
		return
	}
	defer vCursor.Close(context.TODO())

	var vResults []UserViolationDoc
	if err = vCursor.All(context.TODO(), &vResults); err != nil {
		log.Printf("Failed to decode mod violations: %v", err)
		return
	}

	violationsMutex.Lock()
	for _, v := range vResults {
		copyV := v
		violationsCache[v.ID] = &copyV
	}
	violationsMutex.Unlock()
	log.Printf("Loaded %d ModBot user violations from MongoDB", len(vResults))
}

// GetUserViolations returns the number of violations a user has
func GetUserViolations(chatID int64, userID int) int {
	id := fmt.Sprintf("%d_%d", chatID, userID)

	violationsMutex.RLock()
	defer violationsMutex.RUnlock()

	if v, exists := violationsCache[id]; exists {
		return v.Count
	}
	return 0
}

// IncrementUserViolations adds 1 to a user's violation count and saves to DB
func IncrementUserViolations(client *mongo.Client, chatID int64, userID int) int {
	id := fmt.Sprintf("%d_%d", chatID, userID)

	violationsMutex.Lock()
	v, exists := violationsCache[id]
	if !exists {
		v = &UserViolationDoc{
			ID:        id,
			ChatID:    chatID,
			UserID:    userID,
			Count:     0,
		}
		violationsCache[id] = v
	}

	v.Count++
	v.UpdatedAt = time.Now()
	newCount := v.Count

	// Create a copy for async saving to prevent data races
	copyV := *v
	violationsMutex.Unlock()

	if client != nil {
		go func() {
			collection := client.Database("Telegram").Collection("ModViolations")
			filter := bson.M{"_id": id}
			update := bson.M{"$set": copyV}
			opts := options.Update().SetUpsert(true)

			_, err := collection.UpdateOne(context.TODO(), filter, update, opts)
			if err != nil {
				log.Printf("Failed to save mod violation for %s: %v", id, err)
			}
		}()
	}

	return newCount
}
