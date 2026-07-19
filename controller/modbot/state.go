package modbot

import (
	"sync"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// AddRuleState represents the current state of a user's interactive addrule session
type AddRuleState struct {
	Step        int    // 1: Waiting for trigger, 2: Waiting for response
	TriggerWord string // Store the trigger word between steps
}

var (
	interactiveState     = make(map[int64]map[int]AddRuleState) // ChatID -> UserID -> State
	interactiveStateLock sync.RWMutex
)

// GetInteractiveState gets the current state for a user in a chat
func GetInteractiveState(chatID int64, userID int) (AddRuleState, bool) {
	interactiveStateLock.RLock()
	defer interactiveStateLock.RUnlock()

	if chatMap, exists := interactiveState[chatID]; exists {
		state, userExists := chatMap[userID]
		return state, userExists
	}
	return AddRuleState{}, false
}

// SetInteractiveState sets the current state for a user in a chat
func SetInteractiveState(chatID int64, userID int, state AddRuleState) {
	interactiveStateLock.Lock()
	defer interactiveStateLock.Unlock()

	if _, exists := interactiveState[chatID]; !exists {
		interactiveState[chatID] = make(map[int]AddRuleState)
	}
	interactiveState[chatID][userID] = state
}

// ClearInteractiveState removes the state for a user in a chat
func ClearInteractiveState(chatID int64, userID int) {
	interactiveStateLock.Lock()
	defer interactiveStateLock.Unlock()

	if chatMap, exists := interactiveState[chatID]; exists {
		delete(chatMap, userID)
		if len(chatMap) == 0 {
			delete(interactiveState, chatID)
		}
	}
}

// We can store a reference to the message the admin wants to add a rule for.
var (
	pendingRuleMessage     = make(map[int64]map[int]*tgbotapi.Message) // ChatID -> UserID -> Message
	pendingRuleMessageLock sync.RWMutex
)

// SetPendingRuleMessage stores a message that an admin wants to use for a rule response
func SetPendingRuleMessage(chatID int64, userID int, msg *tgbotapi.Message) {
	pendingRuleMessageLock.Lock()
	defer pendingRuleMessageLock.Unlock()

	if _, exists := pendingRuleMessage[chatID]; !exists {
		pendingRuleMessage[chatID] = make(map[int]*tgbotapi.Message)
	}
	pendingRuleMessage[chatID][userID] = msg
}

// GetAndClearPendingRuleMessage retrieves and clears the pending message
func GetAndClearPendingRuleMessage(chatID int64, userID int) (*tgbotapi.Message, bool) {
	pendingRuleMessageLock.Lock()
	defer pendingRuleMessageLock.Unlock()

	if chatMap, exists := pendingRuleMessage[chatID]; exists {
		if msg, userExists := chatMap[userID]; userExists {
			delete(chatMap, userID)
			if len(chatMap) == 0 {
				delete(pendingRuleMessage, chatID)
			}
			return msg, true
		}
	}
	return nil, false
}
