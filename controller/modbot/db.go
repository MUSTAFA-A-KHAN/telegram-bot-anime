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
	ChatID         int64                 `bson:"_id"`
	BlockLinks     bool                  `bson:"block_links"`
	ScamDetection  bool                  `bson:"scam_detection"`
	ScamKeywords   []string              `bson:"scam_keywords"`
	AllowedDomains []string              `bson:"allowed_domains"`
	Rules          map[string]ModRuleDoc `bson:"rules"` // TriggerWord as key for O(1) lookups
}

// UserViolationDoc tracks infractions for a user in a specific chat
type UserViolationDoc struct {
	ID        string    `bson:"_id"` // Composite key: fmt.Sprintf("%d_%d", chatID, userID)
	ChatID    int64     `bson:"chat_id"`
	UserID    int       `bson:"user_id"`
	Count     int       `bson:"count"`
	UpdatedAt time.Time `bson:"updated_at"`
}

// GlobalBanDoc represents a globally banned user
type GlobalBanDoc struct {
	UserID   int       `bson:"_id"`
	BannedAt time.Time `bson:"banned_at"`
	Reason   string    `bson:"reason"`
	BannedBy int       `bson:"banned_by"`
}

// GlobalAdminDoc represents a global administrator
type GlobalAdminDoc struct {
	UserID  int       `bson:"_id"`
	AddedAt time.Time `bson:"added_at"`
	AddedBy int       `bson:"added_by"`
}

// GlobalKeyboardMenuDoc stores the enabled/disabled state of a global DM keyboard item.
type GlobalKeyboardMenuDoc struct {
	Trigger   string    `bson:"_id"`
	Enabled   bool      `bson:"enabled"`
	UpdatedAt time.Time `bson:"updated_at"`
}

var (
	settingsCache             = make(map[int64]*ModChatSettings)
	settingsMutex             sync.RWMutex
	violationsCache           = make(map[string]*UserViolationDoc)
	violationsMutex           sync.RWMutex
	globalBansCache           = make(map[int]*GlobalBanDoc)
	globalBansMutex           sync.RWMutex
	globalAdminsCache         = make(map[int]bool)
	globalAdminsMutex         sync.RWMutex
	globalKeyboardConfigCache = make(map[string]bool)
	globalKeyboardConfigMutex sync.RWMutex
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
		ScamKeywords:   []string{"paid survey", "crypto research"},           // Default scam words
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
			s.ScamKeywords = []string{"paid survey", "crypto research", "وظيفة", "عمل", "شغل", "فرصة", "وظايف", "شواغر", "التسويق", "تسويق", "من البيت", "عن بعد", "بدون خبرة", "دخل يومي", "ارباح", "ربح", "شركة عالمية", "التوظيف", "توظيف"}
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

	// Load Global Bans
	gbCollection := client.Database("Telegram").Collection("ModGlobalBans")
	gbCursor, err := gbCollection.Find(context.TODO(), bson.M{})
	if err != nil {
		log.Printf("Failed to load global bans: %v", err)
		return
	}
	defer gbCursor.Close(context.TODO())

	var gbResults []GlobalBanDoc
	if err = gbCursor.All(context.TODO(), &gbResults); err != nil {
		log.Printf("Failed to decode global bans: %v", err)
		return
	}

	globalBansMutex.Lock()
	for _, gb := range gbResults {
		copyGB := gb
		globalBansCache[gb.UserID] = &copyGB
	}
	globalBansMutex.Unlock()
	log.Printf("Loaded %d globally banned users from MongoDB", len(gbResults))

	// Load Global Admins
	gaCollection := client.Database("Telegram").Collection("ModGlobalAdmins")
	gaCursor, err := gaCollection.Find(context.TODO(), bson.M{})
	if err != nil {
		log.Printf("Failed to load global admins: %v", err)
		return
	}
	defer gaCursor.Close(context.TODO())

	var gaResults []GlobalAdminDoc
	if err = gaCursor.All(context.TODO(), &gaResults); err != nil {
		log.Printf("Failed to decode global admins: %v", err)
		return
	}

	globalAdminsMutex.Lock()
	for _, ga := range gaResults {
		globalAdminsCache[ga.UserID] = true
	}
	globalAdminsMutex.Unlock()
	log.Printf("Loaded %d global admins from MongoDB", len(gaResults))

	// Load global keyboard menu configuration
	gkCollection := client.Database("Telegram").Collection("ModGlobalKeyboardMenu")
	gkCursor, err := gkCollection.Find(context.TODO(), bson.M{})
	if err != nil {
		log.Printf("Failed to load global keyboard menu settings: %v", err)
		return
	}
	defer gkCursor.Close(context.TODO())

	var gkResults []GlobalKeyboardMenuDoc
	if err = gkCursor.All(context.TODO(), &gkResults); err != nil {
		log.Printf("Failed to decode global keyboard menu settings: %v", err)
		return
	}

	globalKeyboardConfigMutex.Lock()
	for _, item := range gkResults {
		globalKeyboardConfigCache[item.Trigger] = item.Enabled
	}
	globalKeyboardConfigMutex.Unlock()
	log.Printf("Loaded %d global keyboard menu states from MongoDB", len(gkResults))
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
			ID:     id,
			ChatID: chatID,
			UserID: userID,
			Count:  0,
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

// IsGloballyBanned checks if a user is globally banned
func IsGloballyBanned(userID int) bool {
	globalBansMutex.RLock()
	defer globalBansMutex.RUnlock()
	_, exists := globalBansCache[userID]
	return exists
}

// GlobalBanUser adds a user to the global ban list and saves to DB
func GlobalBanUser(client *mongo.Client, userID int, reason string, bannedBy int) error {
	globalBansMutex.Lock()
	_, exists := globalBansCache[userID]
	if exists {
		globalBansMutex.Unlock()
		return fmt.Errorf("user %d is already globally banned", userID)
	}

	ban := &GlobalBanDoc{
		UserID:   userID,
		BannedAt: time.Now(),
		Reason:   reason,
		BannedBy: bannedBy,
	}
	globalBansCache[userID] = ban
	globalBansMutex.Unlock()

	if client != nil {
		go func() {
			collection := client.Database("Telegram").Collection("ModGlobalBans")
			filter := bson.M{"_id": userID}
			update := bson.M{"$set": ban}
			opts := options.Update().SetUpsert(true)

			_, err := collection.UpdateOne(context.TODO(), filter, update, opts)
			if err != nil {
				log.Printf("Failed to save global ban for user %d: %v", userID, err)
			}
		}()
	}

	return nil
}

// GlobalUnbanUser removes a user from the global ban list and DB
func GlobalUnbanUser(client *mongo.Client, userID int) error {
	globalBansMutex.Lock()
	delete(globalBansCache, userID)
	globalBansMutex.Unlock()

	if client != nil {
		go func() {
			collection := client.Database("Telegram").Collection("ModGlobalBans")
			_, err := collection.DeleteOne(context.TODO(), bson.M{"_id": userID})
			if err != nil {
				log.Printf("Failed to remove global ban for user %d: %v", userID, err)
			}
		}()
	}

	return nil
}

// GetGloballyBannedUsers returns a copy of the global ban list
func GetGloballyBannedUsers() map[int]*GlobalBanDoc {
	globalBansMutex.RLock()
	defer globalBansMutex.RUnlock()

	result := make(map[int]*GlobalBanDoc, len(globalBansCache))
	for k, v := range globalBansCache {
		copy := *v
		result[k] = &copy
	}
	return result
}

// hardcodedAdminID is the bot owner's ID, used as a built-in global admin fallback.
// This matches the adminID used in controller/bot.go, controller/categoryBot/categoryBot.go,
// and controller/translator/bot.go.
const hardcodedAdminID = 1006461736

// IsGlobalAdmin checks if a user is a global admin.
// The hardcoded bot owner ID (1006461736) is always treated as a global admin,
// even if not in the MongoDB global admins list.
func IsGlobalAdmin(userID int) bool {
	// Built-in fallback: the bot owner is always a global admin
	if userID == hardcodedAdminID {
		return true
	}
	globalAdminsMutex.RLock()
	defer globalAdminsMutex.RUnlock()
	return globalAdminsCache[userID]
}

// AddGlobalAdmin adds a user to the global admin list
func AddGlobalAdmin(client *mongo.Client, userID int, addedBy int) error {
	globalAdminsMutex.Lock()
	_, exists := globalAdminsCache[userID]
	if exists {
		globalAdminsMutex.Unlock()
		return fmt.Errorf("user %d is already a global admin", userID)
	}
	globalAdminsCache[userID] = true
	globalAdminsMutex.Unlock()

	if client != nil {
		doc := GlobalAdminDoc{
			UserID:  userID,
			AddedAt: time.Now(),
			AddedBy: addedBy,
		}
		go func() {
			collection := client.Database("Telegram").Collection("ModGlobalAdmins")
			filter := bson.M{"_id": userID}
			update := bson.M{"$set": doc}
			opts := options.Update().SetUpsert(true)

			_, err := collection.UpdateOne(context.TODO(), filter, update, opts)
			if err != nil {
				log.Printf("Failed to save global admin %d: %v", userID, err)
			}
		}()
	}

	return nil
}

// RemoveGlobalAdmin removes a user from the global admin list.
// The hardcoded bot owner (1006461736) cannot be removed.
func RemoveGlobalAdmin(client *mongo.Client, userID int) error {
	if userID == hardcodedAdminID {
		return fmt.Errorf("the built-in global admin %d cannot be removed", hardcodedAdminID)
	}
	globalAdminsMutex.Lock()
	delete(globalAdminsCache, userID)
	globalAdminsMutex.Unlock()

	if client != nil {
		go func() {
			collection := client.Database("Telegram").Collection("ModGlobalAdmins")
			_, err := collection.DeleteOne(context.TODO(), bson.M{"_id": userID})
			if err != nil {
				log.Printf("Failed to remove global admin %d: %v", userID, err)
			}
		}()
	}

	return nil
}

// GetGlobalAdmins returns a copy of the global admin list, always including the hardcoded admin
func GetGlobalAdmins() []int {
	globalAdminsMutex.RLock()
	defer globalAdminsMutex.RUnlock()

	seen := make(map[int]bool)
	result := make([]int, 0, len(globalAdminsCache)+1)

	// Always include the hardcoded admin
	if !globalAdminsCache[hardcodedAdminID] {
		result = append(result, hardcodedAdminID)
		seen[hardcodedAdminID] = true
	}

	for id := range globalAdminsCache {
		if !seen[id] {
			result = append(result, id)
			seen[id] = true
		}
	}
	return result
}

func isGlobalKeyboardItemEnabled(trigger string) bool {
	globalKeyboardConfigMutex.RLock()
	defer globalKeyboardConfigMutex.RUnlock()

	enabled, exists := globalKeyboardConfigCache[trigger]
	if !exists {
		return true
	}
	return enabled
}

func getEnabledGlobalKeyboardTriggers(triggers []string) []string {
	globalKeyboardConfigMutex.RLock()
	defer globalKeyboardConfigMutex.RUnlock()

	enabled := make([]string, 0, len(triggers))
	for _, trigger := range triggers {
		if enabledState, exists := globalKeyboardConfigCache[trigger]; exists {
			if !enabledState {
				continue
			}
		} else {
			// Newly discovered triggers default to enabled.
		}
		enabled = append(enabled, trigger)
	}
	return enabled
}

func SetGlobalKeyboardMenuState(client *mongo.Client, trigger string, enabled bool) {
	globalKeyboardConfigMutex.Lock()
	globalKeyboardConfigCache[trigger] = enabled
	globalKeyboardConfigMutex.Unlock()

	if client == nil {
		return
	}

	go func() {
		collection := client.Database("Telegram").Collection("ModGlobalKeyboardMenu")
		filter := bson.M{"_id": trigger}
		update := bson.M{"$set": GlobalKeyboardMenuDoc{Trigger: trigger, Enabled: enabled, UpdatedAt: time.Now()}}
		opts := options.Update().SetUpsert(true)

		_, err := collection.UpdateOne(context.TODO(), filter, update, opts)
		if err != nil {
			log.Printf("Failed to save global keyboard menu state for %q: %v", trigger, err)
		}
	}()
}
