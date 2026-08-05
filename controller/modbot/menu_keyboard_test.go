package modbot

import "testing"

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

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

func TestGetEnabledGlobalKeyboardTriggers_FiltersDisabledItems(t *testing.T) {
	globalKeyboardConfigCache = map[string]bool{
		"hello": true,
		"file":  false,
	}

	enabled := getEnabledGlobalKeyboardTriggers([]string{"hello", "file", "test"})
	if len(enabled) != 2 {
		t.Fatalf("expected 2 enabled triggers, got %d", len(enabled))
	}
	if !containsString(enabled, "hello") {
		t.Fatal("expected hello trigger to remain enabled")
	}
	if containsString(enabled, "file") {
		t.Fatal("expected file trigger to be filtered out when disabled")
	}
}

func TestGlobalKeyboardConfigKeyboardUsesSafeCallbackPayloads(t *testing.T) {
	triggers := []string{"hello from bot", "file"}
	keyboard := getGlobalKeyboardConfigInlineKeyboard(triggers)
	if len(keyboard.InlineKeyboard) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(keyboard.InlineKeyboard))
	}
	if keyboard.InlineKeyboard[0][0].CallbackData == nil || *keyboard.InlineKeyboard[0][0].CallbackData != "toggle_global_keyboard_item:0" {
		t.Fatalf("expected first row callback data to be safe numeric payload")
	}
	if keyboard.InlineKeyboard[1][0].CallbackData == nil || *keyboard.InlineKeyboard[1][0].CallbackData != "toggle_global_keyboard_item:1" {
		t.Fatalf("expected second row callback data to be safe numeric payload")
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
