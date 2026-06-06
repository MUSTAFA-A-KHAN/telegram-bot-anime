package image_generator

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// GenerateWordleImage creates an image from Wordle guesses and their feedback
func GenerateWordleImage(guesses []string, targetWord string) ([]byte, error) {
	// Constants
	cellSize := 60
	margin := 10
	padding := 20
	rows := 6
	cols := 5

	// If more than 6 guesses, expand rows
	if len(guesses) > 6 {
		rows = len(guesses)
	}
	if len(guesses) > 0 && len(guesses) < 6 && len(guesses) == 6 {
		rows = 6
	}

	width := cols*cellSize + (cols-1)*margin + 2*padding
	height := rows*cellSize + (rows-1)*margin + 2*padding

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Colors
	bgColor := color.RGBA{18, 18, 19, 255} // Dark background
	emptyCellColor := color.RGBA{58, 58, 60, 255}
	greenColor := color.RGBA{83, 141, 78, 255}
	yellowColor := color.RGBA{181, 159, 59, 255}
	grayColor := color.RGBA{58, 58, 60, 255}
	textColor := color.RGBA{255, 255, 255, 255}

	// Fill background
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	// Load font
	f, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return nil, err
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    32,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, err
	}

	for r := 0; r < rows; r++ {
		var guess string
		if r < len(guesses) {
			guess = strings.ToUpper(guesses[r])
		}

		targetUpper := strings.ToUpper(targetWord)

		// Create a slice to track matched letters in the target word
		matched := make([]bool, len(targetUpper))

		// First pass: Find exact matches (Green)
		colors := make([]color.Color, cols)
		for c := 0; c < cols; c++ {
			colors[c] = emptyCellColor
			if r < len(guesses) && c < len(guess) {
				char := guess[c]
				if char == targetUpper[c] {
					colors[c] = greenColor
					matched[c] = true
				}
			}
		}

		// Second pass: Find partial matches (Yellow) or mismatches (Gray)
		for c := 0; c < cols; c++ {
			if r < len(guesses) && c < len(guess) {
				char := guess[c]
				if colors[c] == greenColor {
					continue // Already green
				}

				// Check if the character exists elsewhere and isn't already matched
				foundYellow := false
				for i, targetChar := range targetUpper {
					if char == byte(targetChar) && !matched[i] {
						colors[c] = yellowColor
						matched[i] = true
						foundYellow = true
						break
					}
				}

				if !foundYellow {
					colors[c] = grayColor
				}
			}
		}

		for c := 0; c < cols; c++ {
			x0 := padding + c*(cellSize+margin)
			y0 := padding + r*(cellSize+margin)
			x1 := x0 + cellSize
			y1 := y0 + cellSize

			cellRect := image.Rect(x0, y0, x1, y1)

			var char rune
			if r < len(guesses) && c < len(guess) {
				char = rune(guess[c])
			}

			// Draw cell background
			draw.Draw(img, cellRect, &image.Uniform{colors[c]}, image.Point{}, draw.Src)

			// Draw border for empty cells
			if r >= len(guesses) || c >= len(guess) {
				borderRect := image.Rect(x0+2, y0+2, x1-2, y1-2)
				draw.Draw(img, borderRect, &image.Uniform{bgColor}, image.Point{}, draw.Src)
			}

			// Draw text
			if char != 0 {
				d := &font.Drawer{
					Dst:  img,
					Src:  image.NewUniform(textColor),
					Face: face,
				}

				// Calculate text bounds to center it
				bounds, _ := d.BoundString(string(char))
				textWidth := (bounds.Max.X - bounds.Min.X).Ceil()
				textHeight := (bounds.Max.Y - bounds.Min.Y).Ceil()

				textX := x0 + (cellSize-textWidth)/2
				textY := y0 + (cellSize+textHeight)/2 - 4 // small adjustment for visual centering

				d.Dot = fixed.Point26_6{X: fixed.I(textX), Y: fixed.I(textY)}
				d.DrawString(string(char))
			}
		}
	}

	var buf bytes.Buffer
	err = png.Encode(&buf, img)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
