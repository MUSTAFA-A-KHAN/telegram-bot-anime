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

func GenerateGrid(words []string, size int) ([][]string, map[string]WordPosition) {
	grid := make([][]string, size)
	for i := range grid {
		grid[i] = make([]string, size)
		for j := range grid[i] {
			grid[i][j] = ""
		}
	}

	positions := make(map[string]WordPosition)
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

	return grid, positions
}

var wordColors = []color.RGBA{
	{255, 105, 97, 180},  // Red
	{255, 180, 128, 180}, // Orange
	{248, 243, 141, 180}, // Yellow
	{66, 214, 164, 180},  // Green
	{8, 202, 209, 180},   // Cyan
	{89, 173, 246, 180},  // Blue
	{157, 148, 255, 180}, // Purple
	{199, 128, 232, 180}, // Pink
}

func GenerateGridImage(grid [][]string, positions map[string]WordPosition, foundWords map[string]bool) ([]byte, error) {
	cellSize := 60
	gridSize := len(grid)
	padding := 30

	width := gridSize*cellSize + 2*padding
	height := gridSize*cellSize + 2*padding

	dc := gg.NewContext(width, height)

	// Background
	dc.SetRGB255(15, 15, 15)
	dc.Clear()

	// Draw grid lines
	dc.SetRGBA255(255, 255, 255, 30)
	dc.SetLineWidth(1)
	for i := 0; i <= gridSize; i++ {
		x := padding + i*cellSize
		dc.DrawLine(float64(x), float64(padding), float64(x), float64(height-padding))
		y := padding + i*cellSize
		dc.DrawLine(float64(padding), float64(y), float64(width-padding), float64(y))
	}
	dc.Stroke()

	// Draw lines for found words
	dc.SetLineCapRound()
	dc.SetLineWidth(float64(cellSize) * 0.7)

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

			dc.SetRGB255(255, 255, 255)
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
