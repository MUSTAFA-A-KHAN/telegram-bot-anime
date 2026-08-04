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
