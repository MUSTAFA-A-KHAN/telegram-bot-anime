package modbot

import "testing"

func TestBuildRuleSelectionKeyboard_IncludesConfiguredTriggers(t *testing.T) {
	settings := &ModChatSettings{
		ChatID: 123,
		Rules: map[string]ModRuleDoc{
			"hello": {
				TriggerWord:  "hello",
				ResponseType: "text",
				ResponseText: "Hello there!",
			},
			"file": {
				TriggerWord:    "file",
				ResponseType:   "photo",
				ResponseFileID: "file-id-123",
			},
		},
	}

	keyboard := buildRuleSelectionKeyboard(settings)
	if len(keyboard.Keyboard) == 0 {
		t.Fatal("expected keyboard rows to be generated")
	}

	buttons := make(map[string]bool)
	for _, row := range keyboard.Keyboard {
		for _, button := range row {
			buttons[button.Text] = true
		}
	}

	if !buttons["hello"] {
		t.Fatal("expected hello trigger to appear in the keyboard menu")
	}
	if !buttons["file"] {
		t.Fatal("expected file trigger to appear in the keyboard menu")
	}
}

func TestDeleteRuleGloballyRemovesAcrossCachedChats(t *testing.T) {
	settingsCache = map[int64]*ModChatSettings{
		123: {
			ChatID: 123,
			Rules: map[string]ModRuleDoc{
				"hello": {TriggerWord: "hello", ResponseType: "text", ResponseText: "hi"},
			},
		},
		456: {
			ChatID: 456,
			Rules: map[string]ModRuleDoc{
				"hello": {TriggerWord: "hello", ResponseType: "text", ResponseText: "bye"},
			},
		},
	}

	removed := deleteRuleGlobally(nil, "hello")
	if removed != 2 {
		t.Fatalf("expected rule to be removed from 2 cached chats, got %d", removed)
	}

	if _, exists := settingsCache[123].Rules["hello"]; exists {
		t.Fatal("expected hello rule to be removed from chat 123")
	}
	if _, exists := settingsCache[456].Rules["hello"]; exists {
		t.Fatal("expected hello rule to be removed from chat 456")
	}
}

func TestDeleteRuleGloballyNormalizesArabicTriggerLookup(t *testing.T) {
	settingsCache = map[int64]*ModChatSettings{
		123: {
			ChatID: 123,
			Rules: map[string]ModRuleDoc{
				"العملات الرقمية": {TriggerWord: "العملات الرقمية", ResponseType: "text", ResponseText: "hi"},
			},
		},
	}

	removed := deleteRuleGlobally(nil, "عملات رقمية")
	if removed != 1 {
		t.Fatalf("expected Arabic trigger to be normalized and removed once, got %d", removed)
	}

	if _, exists := settingsCache[123].Rules["العملات الرقمية"]; exists {
		t.Fatal("expected normalized Arabic rule to be removed from chat 123")
	}
}
