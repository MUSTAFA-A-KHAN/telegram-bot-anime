package service

import (
	"fmt"
	"time"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/config/progression"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// AwardGameResult rewards a user with XP and Coins after playing a game.
// It also increments their games played and wins, and automatically levels them up if necessary.
func AwardGameResult(client *mongo.Client, userID int64, username string, won bool) error {
	// Automatically record that they were active today
	_ = UpdateActiveDay(client, userID, username)

	profile, err := repository.GetUserProfile(client, userID, username)
	if err != nil {
		return err
	}

	xpEarned := progression.Config.PlayXP
	coinsEarned := progression.Config.PlayCoins

	incFields := bson.M{
		"games_played": 1,
	}

	if won {
		xpEarned += progression.Config.WinXP
		coinsEarned += progression.Config.WinCoins
		incFields["wins"] = 1
	}

	incFields["xp"] = xpEarned
	incFields["coins"] = coinsEarned

	newXP := profile.XP + xpEarned
	newLevel := progression.GetLevelFromXP(newXP)

	if newLevel > profile.Level {
		err = repository.UpdateUserProfileFields(client, userID, bson.M{
			"level": newLevel,
		})
		if err != nil {
			return err
		}
		// Notice: In the future we can trigger a level up notification here
	}

	return repository.IncrementUserProfileStats(client, userID, incFields)
}

// ClaimDailyReward allows a user to claim their daily reward once every 24 hours.
// Returns an error if they are on cooldown, otherwise returns the xp and coins earned.
func ClaimDailyReward(client *mongo.Client, userID int64, username string) (int, int, error) {
	profile, err := repository.GetUserProfile(client, userID, username)
	if err != nil {
		return 0, 0, err
	}

	now := time.Now()
	timeSinceLastDaily := now.Sub(profile.LastDailyReward)

	if timeSinceLastDaily < 24*time.Hour {
		remaining := (24 * time.Hour) - timeSinceLastDaily
		hours := int(remaining.Hours())
		minutes := int(remaining.Minutes()) % 60
		return 0, 0, fmt.Errorf("You can claim your next daily reward in %d hours and %d minutes.", hours, minutes)
	}

	// Pseudo-random coins between Min and Max for simplicity (could use math/rand)
	// For this phase we'll just award the max to keep it simple, or an average.
	// We'll award the max configured
	coinsEarned := progression.Config.DailyRewardCoinsMax
	xpEarned := progression.Config.DailyRewardXP

	newXP := profile.XP + xpEarned
	newLevel := progression.GetLevelFromXP(newXP)

	updateFields := bson.M{
		"last_daily_reward": now,
	}
	if newLevel > profile.Level {
		updateFields["level"] = newLevel
	}

	err = repository.UpdateUserProfileFields(client, userID, updateFields)
	if err != nil {
		return 0, 0, err
	}

	err = repository.IncrementUserProfileStats(client, userID, bson.M{
		"xp":    xpEarned,
		"coins": coinsEarned,
	})
	if err != nil {
		return 0, 0, err
	}

	// Update active days for weekly reward
	err = UpdateActiveDay(client, userID, username)
	if err != nil {
		fmt.Println("Error updating active day:", err)
	}

	return xpEarned, coinsEarned, nil
}

// ClaimWeeklyReward allows a user to claim their weekly reward once every 7 days,
// provided they have been active on at least 5 different days.
func ClaimWeeklyReward(client *mongo.Client, userID int64, username string) (int, int, error) {
	profile, err := repository.GetUserProfile(client, userID, username)
	if err != nil {
		return 0, 0, err
	}

	now := time.Now()
	timeSinceLastWeekly := now.Sub(profile.LastWeeklyReward)

	if timeSinceLastWeekly < 7*24*time.Hour {
		return 0, 0, fmt.Errorf("You can claim your next weekly reward in %d days.",
			int(7-(timeSinceLastWeekly.Hours()/24)))
	}

	if len(profile.ActiveDaysThisWeek) < 5 {
		return 0, 0, fmt.Errorf("You must be active on at least 5 different days this week to claim the weekly reward. You have been active on %d days.", len(profile.ActiveDaysThisWeek))
	}

	coinsEarned := progression.Config.WeeklyRewardCoins
	xpEarned := progression.Config.WeeklyRewardXP

	newXP := profile.XP + xpEarned
	newLevel := progression.GetLevelFromXP(newXP)

	updateFields := bson.M{
		"last_weekly_reward":    now,
		"active_days_this_week": []string{}, // reset for the new week
	}
	if newLevel > profile.Level {
		updateFields["level"] = newLevel
	}

	err = repository.UpdateUserProfileFields(client, userID, updateFields)
	if err != nil {
		return 0, 0, err
	}

	err = repository.IncrementUserProfileStats(client, userID, bson.M{
		"xp":    xpEarned,
		"coins": coinsEarned,
	})

	return xpEarned, coinsEarned, err
}

// UpdateActiveDay records today as an active day if not already present.
func UpdateActiveDay(client *mongo.Client, userID int64, username string) error {
	profile, err := repository.GetUserProfile(client, userID, username)
	if err != nil {
		return err
	}

	todayStr := time.Now().Format("2006-01-02")

	for _, day := range profile.ActiveDaysThisWeek {
		if day == todayStr {
			return nil // Already active today
		}
	}

	// Reset the array if they missed the weekly window?
	// For simplicity, we assume the array just accumulates until they claim weekly reward,
	// or we prune old days. Let's just append for now, and claim resets it.

	newDays := append(profile.ActiveDaysThisWeek, todayStr)

	return repository.UpdateUserProfileFields(client, userID, bson.M{
		"active_days_this_week": newDays,
	})
}
