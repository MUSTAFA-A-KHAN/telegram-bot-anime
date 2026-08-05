package progression

import "testing"

func TestGetLevelFromXP(t *testing.T) {
	tests := []struct {
		xp       int
		expected int
	}{
		{0, 1},
		{50, 1},
		{100, 2},
		{150, 2},
		{250, 3},
		{500, 4},
		{900, 5},
		{4500, 10},
		{5000, 10}, // Assuming max defined level for now
	}

	for _, test := range tests {
		level := GetLevelFromXP(test.xp)
		if level != test.expected {
			t.Errorf("For XP %d, expected level %d, got %d", test.xp, test.expected, level)
		}
	}
}
