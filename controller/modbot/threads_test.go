package modbot

import (
	"encoding/json"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// TestMessageThreadID verifies forum topic extraction from incoming messages.
func TestMessageThreadID(t *testing.T) {
	if got := messageThreadID(nil); got != 0 {
		t.Fatalf("expected 0 for nil message, got %d", got)
	}

	var regular tgbotapi.Message
	_ = json.Unmarshal([]byte(`{"message_id":1,"chat":{"id":-100,"type":"supergroup"}}`), &regular)
	if got := messageThreadID(&regular); got != 0 {
		t.Fatalf("expected 0 for non-topic message, got %d", got)
	}

	var topicMsg tgbotapi.Message
	_ = json.Unmarshal([]byte(`{"message_id":2,"message_thread_id":42,"chat":{"id":-100,"type":"supergroup","is_forum":true}}`), &topicMsg)
	if got := messageThreadID(&topicMsg); got != 42 {
		t.Fatalf("expected 42 for topic message, got %d", got)
	}

	if !topicMsg.Chat.IsForum {
		t.Fatal("expected chat to be flagged as forum")
	}
}
