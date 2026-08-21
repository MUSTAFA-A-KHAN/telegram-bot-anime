package service

import (
	"fmt"
	"html"

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

	escapedUsername := html.EscapeString(profile.Username)

	htmlStr := fmt.Sprintf(`<b>👤 %s's Profile</b>
<blockquote>
<b>%s %s</b>

⭐ <b>Level:</b> %d
✨ <b>XP:</b> %d
🪙 <b>Coins:</b> %d
</blockquote>

<b>📊 Stats</b>
<blockquote>
🎮 <b>Games Played:</b> %d
🏆 <b>Wins:</b> %d
📈 <b>Win Rate:</b> %.1f%%
🔥 <b>Current Streak:</b> %d
</blockquote>`,
		escapedUsername, badge, title, profile.Level, profile.XP, profile.Coins,
		profile.GamesPlayed, profile.Wins, winRate, profile.CurrentStreak)

	return htmlStr
}
