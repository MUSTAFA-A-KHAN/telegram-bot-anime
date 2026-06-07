package service

import (
	"bytes"
	"fmt"
	"image/color"
	"log"
	"strings"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/gofont/gobold"
)

// GenerateLeaderboardImage queries the leaderboard and generates an image scorecard.
func GenerateLeaderboardImage(client *mongo.Client, collection string, chatID int64, title string) ([]byte, error) {
	idCounts, err := repository.CountIDOccurrences(client, collection, chatID)
	if err != nil {
		log.Printf("Error getting leaderboard for image: %v", err)
	}

	limit := 10
	if len(idCounts) < limit {
		limit = len(idCounts)
	}

	// Layout parameters
	width := 800
	height := 150 + (limit * 50)
	if limit == 0 {
		height = 200
	}

	dc := gg.NewContext(width, height)

	// Draw background
	dc.SetColor(color.RGBA{R: 20, G: 25, B: 30, A: 255})
	dc.Clear()

	// Load fonts
	fontReg, err := truetype.Parse(goregular.TTF)
	if err != nil {
		return nil, err
	}
	fontBold, err := truetype.Parse(gobold.TTF)
	if err != nil {
		return nil, err
	}

	faceTitle := truetype.NewFace(fontBold, &truetype.Options{Size: 32})
	faceHeader := truetype.NewFace(fontBold, &truetype.Options{Size: 22})
	faceRow := truetype.NewFace(fontReg, &truetype.Options{Size: 20})

	// Draw title
	dc.SetFontFace(faceTitle)
	dc.SetColor(color.RGBA{R: 255, G: 215, B: 0, A: 255}) // Gold
	dc.DrawStringAnchored(title, float64(width/2), 50, 0.5, 0.5)

	// Draw table header
	dc.SetFontFace(faceHeader)
	dc.SetColor(color.White)

	y := 110.0
	dc.DrawStringAnchored("Rank", 100, y, 0.5, 0.5)
	dc.DrawStringAnchored("Player", 350, y, 0.5, 0.5)
	dc.DrawStringAnchored("Score", 650, y, 0.5, 0.5)

	// Draw line under header
	dc.SetLineWidth(2)
	dc.DrawLine(50, y+15, float64(width-50), y+15)
	dc.SetColor(color.RGBA{R: 100, G: 100, B: 100, A: 255})
	dc.Stroke()

	if limit == 0 {
		dc.SetFontFace(faceRow)
		dc.SetColor(color.White)
		dc.DrawStringAnchored("No stats found yet!", float64(width/2), y+50, 0.5, 0.5)
	} else {
		dc.SetFontFace(faceRow)
		for i := 0; i < limit; i++ {
			y += 50
			count := idCounts[i]
			name := fmt.Sprintf("%v", count["Name"])

			// Try getting emojis, but `gg` might not render unicode emojis well.
			// We will try our best.
			var userID int
			if id, ok := count["_id"]; ok {
				switch v := id.(type) {
				case int32:
					userID = int(v)
				case int64:
					userID = int(v)
				case int:
					userID = v
				}
			}

			equippedEmojis, err := repository.GetEquippedEmojis(client, userID)
			if err == nil && len(equippedEmojis) > 0 {
				name += " " + strings.Join(equippedEmojis, "")
			}

			score := fmt.Sprintf("%v", count["count"])
			if collection == "WordleEn" {
				score += " pts"
			} else if collection == "ScramyEn" {
				score += " pts"
			}

			rankDisplay := fmt.Sprintf("#%d", i+1)

			// Highlight top 3
			if i == 0 {
				dc.SetColor(color.RGBA{R: 255, G: 215, B: 0, A: 255}) // Gold
				rankDisplay = "1st"
			} else if i == 1 {
				dc.SetColor(color.RGBA{R: 192, G: 192, B: 192, A: 255}) // Silver
				rankDisplay = "2nd"
			} else if i == 2 {
				dc.SetColor(color.RGBA{R: 205, G: 127, B: 50, A: 255}) // Bronze
				rankDisplay = "3rd"
			} else {
				dc.SetColor(color.White)
			}

			// Background striping
			if i%2 == 0 {
				dc.SetColor(color.RGBA{R: 40, G: 45, B: 50, A: 255})
				dc.DrawRectangle(50, y-20, float64(width-100), 40)
				dc.Fill()
			}

			// Restore color for text based on rank
			if i == 0 {
				dc.SetColor(color.RGBA{R: 255, G: 215, B: 0, A: 255})
			} else if i == 1 {
				dc.SetColor(color.RGBA{R: 192, G: 192, B: 192, A: 255})
			} else if i == 2 {
				dc.SetColor(color.RGBA{R: 205, G: 127, B: 50, A: 255})
			} else {
				dc.SetColor(color.White)
			}

			dc.DrawStringAnchored(rankDisplay, 100, y, 0.5, 0.5)

			dc.SetColor(color.White)
			// Draw string left-aligned for player name
			dc.DrawStringAnchored(name, 350, y, 0.5, 0.5)

			// Score
			if i == 0 {
				dc.SetColor(color.RGBA{R: 255, G: 215, B: 0, A: 255})
			} else if i == 1 {
				dc.SetColor(color.RGBA{R: 192, G: 192, B: 192, A: 255})
			} else if i == 2 {
				dc.SetColor(color.RGBA{R: 205, G: 127, B: 50, A: 255})
			} else {
				dc.SetColor(color.RGBA{R: 200, G: 255, B: 200, A: 255})
			}
			dc.DrawStringAnchored(score, 650, y, 0.5, 0.5)
		}
	}

	buf := new(bytes.Buffer)
	err = dc.EncodePNG(buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
