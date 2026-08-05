package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserProfile struct {
	ID                 primitive.ObjectID `bson:"_id,omitempty"`
	UserID             int64              `bson:"user_id"`
	Username           string             `bson:"username"`
	Level              int                `bson:"level"`
	XP                 int                `bson:"xp"`
	Coins              int                `bson:"coins"`
	EquippedTitle      string             `bson:"equipped_title"`
	EquippedBadge      string             `bson:"equipped_badge"`
	NameColor          string             `bson:"name_color"`
	ChatFlair          string             `bson:"chat_flair"`
	CurrentStreak      int                `bson:"current_streak"`
	LastDailyReward    time.Time          `bson:"last_daily_reward"`
	LastWeeklyReward   time.Time          `bson:"last_weekly_reward"`
	ActiveDaysThisWeek []string           `bson:"active_days_this_week"` // e.g. "2023-10-01"
	GamesPlayed        int                `bson:"games_played"`
	Wins               int                `bson:"wins"`
	CreatedAt          time.Time          `bson:"created_at"`
	UpdatedAt          time.Time          `bson:"updated_at"`
}
