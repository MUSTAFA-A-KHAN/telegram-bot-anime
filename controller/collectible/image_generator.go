package collectible

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/http"

	model "github.com/MUSTAFA-A-KHAN/telegram-bot-anime/model/collectible"
	"github.com/fogleman/gg"
	"golang.org/x/image/font/gofont/goregular"
	"github.com/golang/freetype/truetype"
)

// GenerateCollectibleCard creates a composite card image for a collectible
func GenerateCollectibleCard(tmpl model.Template, serialNumber int, isMarket bool, price int) ([]byte, error) {
	const (
		W = 600
		H = 800
	)

	dc := gg.NewContext(W, H)

	// Draw background
	dc.SetHexColor("#0f172a") // dark blue
	dc.Clear()

	// 1. Download and draw the image
	if tmpl.ImageURL != "" {
		resp, err := http.Get(tmpl.ImageURL)
		if err == nil {
			defer resp.Body.Close()
			img, _, err := image.Decode(resp.Body)
			if err == nil {
				// Draw image at top
				// Scale to fit width
				iw := img.Bounds().Dx()

				scale := float64(W) / float64(iw)

				// create scaled version
				dc.Push()
				dc.Scale(scale, scale)
				dc.DrawImage(img, 0, 0)
				dc.Pop()
			}
		}
	}

	// 2. Load Font
	font, err := truetype.Parse(goregular.TTF)
	if err == nil {
		face := truetype.NewFace(font, &truetype.Options{Size: 32})
		dc.SetFontFace(face)
	}

	// 3. Draw Top Pills
	// Serial Pill (Top Left)
	dc.SetHexColor("#f59e0b") // amber
	dc.DrawRoundedRectangle(20, 20, 100, 50, 10)
	dc.Fill()
	dc.SetHexColor("#000000")
	if font != nil {
		face := truetype.NewFace(font, &truetype.Options{Size: 28})
		dc.SetFontFace(face)
	}
	dc.DrawStringAnchored(fmt.Sprintf("#%d", serialNumber), 70, 45, 0.5, 0.5)

	// Rarity Pill (Top Right)
	dc.SetHexColor("#1e293b") // dark slate
	dc.DrawRoundedRectangle(W-220, 20, 200, 50, 10)
	dc.Fill()
	dc.SetHexColor("#fbbf24") // gold text
	if font != nil {
		face := truetype.NewFace(font, &truetype.Options{Size: 24})
		dc.SetFontFace(face)
	}
	dc.DrawStringAnchored(fmt.Sprintf("⭐ %s", string(tmpl.Rarity)), W-120, 45, 0.5, 0.5)

	// 4. Draw Bottom Info Pane (over the bottom of the image)
	paneY := float64(H - 250)
	dc.SetHexColor("#0f172a")
	dc.DrawRoundedRectangle(0, paneY, W, 250, 20)
	dc.Fill()

	// Name
	dc.SetHexColor("#ffffff")
	if font != nil {
		face := truetype.NewFace(font, &truetype.Options{Size: 40})
		dc.SetFontFace(face)
	}
	dc.DrawStringAnchored(fmt.Sprintf("%s %s #%d", tmpl.Emoji, tmpl.Name, serialNumber), W/2, paneY+50, 0.5, 0.5)

	// Details
	if font != nil {
		face := truetype.NewFace(font, &truetype.Options{Size: 24})
		dc.SetFontFace(face)
	}
	dc.SetHexColor("#94a3b8")

	if isMarket {
		dc.DrawStringAnchored("Market Listing", W/2, paneY+110, 0.5, 0.5)
		dc.SetHexColor("#fbbf24")
		if font != nil {
			face := truetype.NewFace(font, &truetype.Options{Size: 32})
			dc.SetFontFace(face)
		}
		dc.DrawStringAnchored(fmt.Sprintf("💰 Price: %d Coins", price), W/2, paneY+160, 0.5, 0.5)
	} else {
		dc.DrawStringAnchored("In your collection", W/2, paneY+110, 0.5, 0.5)
		dc.SetHexColor("#38bdf8")
		dc.DrawStringAnchored(fmt.Sprintf("Rarity: %s", string(tmpl.Rarity)), W/2, paneY+150, 0.5, 0.5)
	}

	// 5. Encode
	buf := new(bytes.Buffer)
	err = dc.EncodePNG(buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
