package wordgridbot

import (
	"bytes"
	"fmt"
	"image/color"
	"math/rand"
	"strings"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font/gofont/gobold"
)

func GenerateGrid(words []string, size int) ([][]string, []string, map[string]WordPosition) {
	grid := make([][]string, size)
	for i := range grid {
		grid[i] = make([]string, size)
		for j := range grid[i] {
			grid[i][j] = ""
		}
	}

	positions := make(map[string]WordPosition)
	var placedWords []string
	dirs := [][2]int{{0, 1}, {1, 0}, {1, 1}, {-1, 1}, {1, -1}, {-1, -1}, {0, -1}, {-1, 0}}

	for _, word := range words {
		word = strings.ToUpper(word)
		placed := false
		for attempts := 0; attempts < 100 && !placed; attempts++ {
			dir := dirs[rand.Intn(len(dirs))]
			row := rand.Intn(size)
			col := rand.Intn(size)

			canPlace := true
			for i := 0; i < len(word); i++ {
				r := row + i*dir[0]
				c := col + i*dir[1]
				if r < 0 || r >= size || c < 0 || c >= size || (grid[r][c] != "" && grid[r][c] != string(word[i])) {
					canPlace = false
					break
				}
			}

			if canPlace {
				for i := 0; i < len(word); i++ {
					r := row + i*dir[0]
					c := col + i*dir[1]
					grid[r][c] = string(word[i])
				}
				positions[word] = WordPosition{
					StartRow: row,
					StartCol: col,
					EndRow:   row + (len(word)-1)*dir[0],
					EndCol:   col + (len(word)-1)*dir[1],
				}
				placedWords = append(placedWords, word)
				placed = true
			}
		}
	}

	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			if grid[i][j] == "" {
				grid[i][j] = string(rune('A' + rand.Intn(26)))
			}
		}
	}

	return grid, placedWords, positions
}

var wordColors = []color.RGBA{
	{200, 110, 120, 170}, // Muted Rose
	{155, 120, 190, 170}, // Muted Lavender
	{110, 175, 145, 170}, // Muted Mint
	{200, 160, 110, 170}, // Muted Peach
	{120, 165, 195, 170}, // Muted Sky
	{200, 185, 110, 170}, // Muted Butter
	{200, 130, 120, 170}, // Muted Coral
	{175, 130, 180, 170}, // Muted Lilac
}

func GenerateGridImage(grid [][]string, positions map[string]WordPosition, foundWords map[string]bool) ([]byte, error) {
	cellSize := 60
	gridSize := len(grid)
	padding := 30

	width := gridSize*cellSize + 2*padding
	height := gridSize*cellSize + 2*padding

	dc := gg.NewContext(width, height)

	// Background - warm dark tone instead of pure gray
	dc.SetRGB255(16, 16, 16)
	dc.Clear()

	// Draw grid lines - bolder for better visibility
	dc.SetRGBA255(235, 225, 210, 60)
	dc.SetLineWidth(2)
	for i := 0; i <= gridSize; i++ {
		x := padding + i*cellSize
		dc.DrawLine(float64(x), float64(padding), float64(x), float64(height-padding))
		y := padding + i*cellSize
		dc.DrawLine(float64(padding), float64(y), float64(width-padding), float64(y))
	}
	dc.Stroke()

	// Draw highlight lines for found words
	dc.SetLineCapRound()
	dc.SetLineWidth(float64(cellSize) * 0.55)

	colorIdx := 0
	for word, found := range foundWords {
		if found {
			pos := positions[strings.ToUpper(word)]
			c := wordColors[colorIdx%len(wordColors)]
			dc.SetRGBA255(int(c.R), int(c.G), int(c.B), int(c.A))

			startX := float64(padding + pos.StartCol*cellSize + cellSize/2)
			startY := float64(padding + pos.StartRow*cellSize + cellSize/2)
			endX := float64(padding + pos.EndCol*cellSize + cellSize/2)
			endY := float64(padding + pos.EndRow*cellSize + cellSize/2)

			dc.DrawLine(startX, startY, endX, endY)
			dc.Stroke()
			colorIdx++
		}
	}

	// Draw text
	f, err := truetype.Parse(gobold.TTF)
	if err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}
	face := truetype.NewFace(f, &truetype.Options{Size: 30})
	dc.SetFontFace(face)

	for r := 0; r < gridSize; r++ {
		for c := 0; c < gridSize; c++ {
			x := padding + c*cellSize + cellSize/2
			y := padding + r*cellSize + cellSize/2

			dc.SetRGB255(235, 230, 220)
			dc.DrawStringAnchored(grid[r][c], float64(x), float64(y), 0.5, 0.5)
		}
	}

	buf := new(bytes.Buffer)
	err = dc.EncodePNG(buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
