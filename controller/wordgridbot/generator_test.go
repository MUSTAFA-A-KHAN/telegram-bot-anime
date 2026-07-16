package wordgridbot

import (
	"testing"
)

func TestGenerateGridAndImage(t *testing.T) {
	words := []string{"TEST", "HELLO", "WORLD", "GOLANG"}
	grid, placedWords, positions := GenerateGrid(words, 10)

	if len(grid) != 10 {
		t.Errorf("Expected grid size 10, got %d", len(grid))
	}

	if len(placedWords) == 0 {
		t.Errorf("Expected at least one word to be placed, got 0")
	}

	// Verify that all placed words are actually findable in the grid
	for _, word := range placedWords {
		found := false
		dirs := [][2]int{{0, 1}, {1, 0}, {1, 1}, {-1, 1}, {1, -1}, {-1, -1}, {0, -1}, {-1, 0}}
		for r := 0; r < len(grid); r++ {
			for c := 0; c < len(grid); c++ {
				if grid[r][c] == string(word[0]) {
					for _, dir := range dirs {
						match := true
						for i := 0; i < len(word); i++ {
							nr := r + i*dir[0]
							nc := c + i*dir[1]
							if nr < 0 || nr >= len(grid) || nc < 0 || nc >= len(grid) || grid[nr][nc] != string(word[i]) {
								match = false
								break
							}
						}
						if match {
							found = true
							break
						}
					}
				}
				if found {
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			t.Errorf("Placed word %q was not found in the grid", word)
		}
	}

	// Verify that each placed word has a valid position entry
	for _, word := range placedWords {
		pos, ok := positions[word]
		if !ok {
			t.Errorf("Placed word %q has no position entry", word)
		}
		if pos.StartRow < 0 || pos.StartRow >= len(grid) || pos.StartCol < 0 || pos.StartCol >= len(grid) {
			t.Errorf("Placed word %q has invalid start position (%d,%d)", word, pos.StartRow, pos.StartCol)
		}
		if pos.EndRow < 0 || pos.EndRow >= len(grid) || pos.EndCol < 0 || pos.EndCol >= len(grid) {
			t.Errorf("Placed word %q has invalid end position (%d,%d)", word, pos.EndRow, pos.EndCol)
		}
	}

	// Only test the image if we placed at least "TEST" and "WORLD" (they might not all fit)
	foundWords := map[string]bool{}
	for _, w := range placedWords {
		if w == "TEST" || w == "WORLD" {
			foundWords[w] = true
		}
	}

	imgBytes, err := GenerateGridImage(grid, positions, foundWords)
	if err != nil {
		t.Fatalf("Failed to generate image: %v", err)
	}
	if len(imgBytes) == 0 {
		t.Errorf("Generated image is empty")
	}
}
