package wordgridbot

import (
	"testing"
)

func TestGenerateGridAndImage(t *testing.T) {
	words := []string{"TEST", "HELLO", "WORLD", "GOLANG"}
	grid, positions := GenerateGrid(words, 10)

	if len(grid) != 10 {
		t.Errorf("Expected grid size 10, got %d", len(grid))
	}

	foundWords := map[string]bool{"TEST": true, "WORLD": true}
	imgBytes, err := GenerateGridImage(grid, positions, foundWords)

	if err != nil {
		t.Fatalf("Failed to generate image: %v", err)
	}

	if len(imgBytes) == 0 {
		t.Errorf("Generated image is empty")
	}
}
