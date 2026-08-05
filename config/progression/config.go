package progression

import "sort"

// LevelThresholds defines the XP required for each level.
// This is a simple configurable map mapping Level -> Total XP required to reach it.
var LevelThresholds = map[int]int{
	1:  0,
	2:  100,
	3:  250,
	4:  500,
	5:  900,
	6:  1400,
	7:  2000,
	8:  2700,
	9:  3500,
	10: 4500,
	// Additional levels can be added here
}

// Config holds all the reward configurations.
var Config = struct {
	PlayXP              int
	WinXP               int
	DailyRewardXP       int
	DailyRewardCoinsMin int
	DailyRewardCoinsMax int
	WeeklyRewardXP      int
	WeeklyRewardCoins   int
	PlayCoins           int
	WinCoins            int
}{
	PlayXP:              10,
	WinXP:               20,
	DailyRewardXP:       25,
	DailyRewardCoinsMin: 20,
	DailyRewardCoinsMax: 50,
	WeeklyRewardXP:      100,
	WeeklyRewardCoins:   100,
	PlayCoins:           5,
	WinCoins:            10,
}

// GetLevelFromXP calculates the current level based on total XP.
func GetLevelFromXP(xp int) int {
	var levels []int
	for level := range LevelThresholds {
		levels = append(levels, level)
	}
	// Ensure levels are sorted
	sort.Ints(levels)

	currentLevel := 1
	for _, level := range levels {
		if xp >= LevelThresholds[level] {
			currentLevel = level
		} else {
			break
		}
	}
	return currentLevel
}
