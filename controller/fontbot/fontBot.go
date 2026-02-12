package fontbot

import (
	"fmt"
	"log"
	"strings"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/service"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// FormatBot handles inline queries for text formatting
type FormatBot struct {
	bot *tgbotapi.BotAPI
}

// StartFormatBot initializes and starts the format bot for inline queries
func StartFormatBot(botToken string) error {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return err
	}

	formatBot := &FormatBot{bot: bot}

	bot.Debug = true
	log.Printf("Format Bot authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		return err
	}

	// Handle both inline queries and regular messages
	for update := range updates {
		if update.InlineQuery != nil {
			go formatBot.handleInlineQuery(update.InlineQuery)
		}
	}

	return nil
}

// handleInlineQuery processes incoming inline queries and returns formatted text results
func (fb *FormatBot) handleInlineQuery(inlineQuery *tgbotapi.InlineQuery) {
	query := strings.TrimSpace(inlineQuery.Query)

	// If query is empty, send a hint
	if query == "" {
		answer := tgbotapi.NewInlineQueryResultArticle(
			"1",
			"🎨 Text Formatter",
			"Type some text after @BotName to see font style options!",
		)
		answer.Description = "Type text to format"
		answer.InputMessageContent = tgbotapi.InputTextMessageContent{
			Text: "🎨 *Text Formatter Bot*\n\n" +
				"Use me in any chat by typing:\n" +
				"`@YourBotName Your text here`\n\n" +
				"I'll show you different font styles to choose from!",
			ParseMode: "Markdown",
		}

		config := tgbotapi.InlineConfig{
			InlineQueryID: inlineQuery.ID,
			Results:       []interface{}{answer},
		}
		_, err := fb.bot.AnswerInlineQuery(config)
		if err != nil {
			log.Printf("Failed to answer empty inline query: %v", err)
		}
		return
	}

	// Get all font styles and create inline results
	styles := service.GetAllFontStyles()
	results := make([]interface{}, 0, len(styles))

	for i, style := range styles {
		formattedText := style.Converter(query)

		// Create a unique ID for each result
		id := fmt.Sprintf("%d_%s", i, style.Name)

		// Create inline query result
		result := tgbotapi.NewInlineQueryResultArticle(
			id,
			fmt.Sprintf("%s %s", getStyleEmoji(style.Name), style.Name),
			formattedText,
		)

		result.Description = fmt.Sprintf("%s: %s", style.Description, truncateString(formattedText, 50))

		// Set the input message content to send the formatted text
		result.InputMessageContent = tgbotapi.InputTextMessageContent{
			Text:      formattedText,
			ParseMode: "",
		}

		results = append(results, result)
	}

	// Answer the inline query with all results
	config := tgbotapi.InlineConfig{
		InlineQueryID: inlineQuery.ID,
		Results:       results,
	}
	_, err := fb.bot.AnswerInlineQuery(config)
	if err != nil {
		log.Printf("Failed to answer inline query: %v", err)
	}
}

// getStyleEmoji returns an emoji for each font style
func getStyleEmoji(styleName string) string {
	emojis := map[string]string{
		"Bold":             "𝐁",
		"Italic":           "𝑰",
		"Bold Italic":      "𝑩𝑰",
		"Monospace":        "𝙼",
		"Double Struck":    "𝔻",
		"Sans Serif":       "𝖲",
		"Bold Sans":        "𝗕",
		"Italic Sans":      "𝘒",
		"Bold Italic Sans": "𝙱",
		"Script":           "𝒮",
		"Bold Script":      "𝓑",
		"Fraktur":          "𝔉",
		"Bold Fraktur":     "𝕭",
		"Small Caps":       "ᴀ",
		"Reversed":         "🔄",
		"Wide":             "Ｆ",
	}
	if emoji, ok := emojis[styleName]; ok {
		return emoji
	}
	return "📝"
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

