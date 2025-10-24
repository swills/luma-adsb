package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/waxdred/go-i2c-oled"
	"github.com/waxdred/go-i2c-oled/ssd1306"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

type stage2stats struct {
	Pps         float32 `json:"pps"`
	Mps         float32 `json:"mps"`
	Uptime      int     `json:"uptime"`
	Planes      int     `json:"planes"`
	TotalPlanes int     `json:"tplanes"`
}

func main() {
	// Initialize the OLED display with the provided parameters
	oled, err := goi2coled.NewI2c(ssd1306.SSD1306_SWITCHCAPVCC, 64, 128, 0x3C, 1)
	if err != nil {
		panic(err)
	}

	defer func(oled *goi2coled.I2c) {
		_ = oled.Close()
	}(oled)

	black := color.RGBA{
		R: 0,
		G: 0,
		B: 0,
		A: 255,
	}

	// Set the entire OLED image to black
	draw.Draw(oled.Img, oled.Img.Bounds(), &image.Uniform{C: black}, image.Point{}, draw.Src)

	// Define a white color
	colWhite := color.RGBA{R: 255, G: 255, B: 255, A: 255}

	// Set the starting point for drawing text
	point := fixed.Point26_6{
		X: fixed.Int26_6(0 * 64),
		Y: fixed.Int26_6(15 * 64),
	} // x = 0, y = 15

	var drawer *font.Drawer

	// Configure the font drawer with the chosen font and color
	drawer = &font.Drawer{
		Dst:  oled.Img,
		Src:  &image.Uniform{C: colWhite},
		Face: basicfont.Face7x13,
		Dot:  point,
	}

	host := os.Getenv("ADSBFEED_HOST")
	if host == "" {
		fmt.Printf("ADSBFEED_HOST not set (probably YOURHOSTNAME)\n")

		return
	}

	var stats *stage2stats

	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-ticker.C:
			stats, err = getStage2Stats(host)
			if err != nil {
				fmt.Printf("error: %s\n", err)

				continue
			}

			draw.Draw(oled.Img, oled.Img.Bounds(), &image.Uniform{C: color.Black}, image.Point{}, draw.Src)

			drawer = &font.Drawer{
				Dst:  oled.Img,
				Src:  &image.Uniform{C: colWhite},
				Face: basicfont.Face7x13,
				Dot:  point,
			}

			now := time.Now()
			formattedTime := now.Format("15:04:05")
			timeStr := fmt.Sprintf("Time: %s", formattedTime)

			drawer.DrawString(timeStr)

			h := basicfont.Face7x13.Metrics().Height

			drawer.Dot.X = fixed.Int26_6(0)
			drawer.Dot.Y += h

			drawer.DrawString(fmt.Sprintf("Planes: %d", stats.Planes))

			drawer.Dot.X = fixed.Int26_6(0)
			drawer.Dot.Y += h

			drawer.DrawString(fmt.Sprintf("Planes Today: %d", stats.TotalPlanes))

			oled.Clear()

			oled.Draw()

			err = oled.Display()
			if err != nil {
				fmt.Printf("error: %s\n", err)
			}
		}
	}
}

func getStage2Stats(host string) (*stage2stats, error) {
	var err error

	var req *http.Request

	var res *http.Response

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	url := fmt.Sprintf("http://%s/api/stage2_stats", host)

	req, err = http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http req: %w", err)
	}

	res, err = http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making http request: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status making request: %w", err)
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(res.Body)

	var body []byte

	body, err = io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var foundStage2Stats []stage2stats

	err = json.Unmarshal(body, &foundStage2Stats)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling: %w", err)
	}

	if len(foundStage2Stats) < 1 {
		return nil, errors.New("failed to get stats")
	}

	return &foundStage2Stats[0], nil
}
