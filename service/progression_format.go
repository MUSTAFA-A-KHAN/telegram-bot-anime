package service

import (
	"fmt"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/model"
)

// FormatUserProfile formats the user profile into an HTML string for the Telegram message.
func FormatUserProfile(profile *model.UserProfile) string {
	title := profile.EquippedTitle
	if title == "" {
		title = "Novice"
	}

	badge := profile.EquippedBadge
	if badge == "" {
		badge = "🔰" // Default badge
	}

	winRate := 0.0
	if profile.GamesPlayed > 0 {
		winRate = float64(profile.Wins) / float64(profile.GamesPlayed) * 100
	}

	html := fmt.Sprintf(`<b>👤 %s's Profile</b>

<b>%s %s</b>
Level: <b>%d</b>
XP: <b>%d</b>
Coins: <b>%d</b>

<b>📊 Stats</b>
Games Played: <b>%d</b>
Wins: <b>%d</b>
Win Rate: <b>%.1f%%</b>
Current Streak: <b>%d</b>`,
		profile.Username, badge, title, profile.Level, profile.XP, profile.Coins,
		profile.GamesPlayed, profile.Wins, winRate, profile.CurrentStreak)

	return html
}
