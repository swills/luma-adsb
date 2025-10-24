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

type Aircraft struct {
	Hex        string     `json:"hex"`
	MarkerType string     `json:"type"`
	CallSign   string     `json:"flight"`
	Latitude   float64    `json:"lat"`
	Longitude  float64    `json:"lon"`
	Altitude   json.Token `json:"alt_baro"`
	Category   string     `json:"category,omitempty"`
}

type ADSBData struct {
	Planes []Aircraft `json:"aircraft"`
}

type displayLines []string

func initDisplay() *goi2coled.I2c {
	// Initialize the OLED display with the provided parameters
	oled, err := goi2coled.NewI2c(ssd1306.SSD1306_SWITCHCAPVCC, 64, 128, 0x3C, 1)
	if err != nil {
		panic(err)
	}

	black := color.RGBA{
		R: 0,
		G: 0,
		B: 0,
		A: 255,
	}

	// Set the entire OLED image to black
	draw.Draw(oled.Img, oled.Img.Bounds(), &image.Uniform{C: black}, image.Point{}, draw.Src)

	return oled
}

func main() {
	oled := initDisplay()
	defer func(oled *goi2coled.I2c) {
		_ = oled.Close()
	}(oled)

	host := os.Getenv("ADSBFEED_HOST")
	if host == "" {
		fmt.Printf("ADSBFEED_HOST not set (probably YOURHOSTNAME)\n")

		return
	}

	stats := &stage2stats{
		Pps:         0,
		Mps:         0,
		Uptime:      0,
		Planes:      0,
		TotalPlanes: 0,
	}

	displayTicker := time.NewTicker(1 * time.Second)
	stage2Ticker := time.NewTicker(5 * time.Second)
	//aircraftDataTicker := time.NewTicker(5 * time.Second)

	var err error

	for {
		select {
		case <-stage2Ticker.C:
			stats, err = getStage2Stats(host)
			if err != nil {
				fmt.Printf("error: %s\n", err)

				continue
			}
		case <-displayTicker.C:
			now := time.Now()
			dispLines := []string{
				fmt.Sprintf("Time: %s", now.Format("15:04:05")),
				fmt.Sprintf("Planes: %d", stats.Planes),
				fmt.Sprintf("Planes Today: %d", stats.TotalPlanes),
				"",
				"",
			}

			err = updateDisplayLines(dispLines, oled)
			if err != nil {
				fmt.Printf("error updating display: %s", err)
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

func updateDisplayLines(dispLines displayLines, oled *goi2coled.I2c) error {
	h := basicfont.Face7x13.Metrics().Height
	var err error

	var drawer *font.Drawer

	point := fixed.Point26_6{
		X: fixed.Int26_6(0 * 64),
		Y: fixed.Int26_6(15 * 64),
	} // x = 0, y = 15

	drawer = &font.Drawer{
		Dst:  oled.Img,
		Src:  &image.Uniform{C: color.RGBA{R: 255, G: 255, B: 255, A: 255}},
		Face: basicfont.Face7x13,
		Dot:  point,
	}

	draw.Draw(oled.Img, oled.Img.Bounds(), &image.Uniform{C: color.Black}, image.Point{}, draw.Src)

	for i, line := range dispLines {
		if line == "" || i > 5 {
			break
		}

		drawer.DrawString(line)
		drawer.Dot.X = fixed.Int26_6(0)
		drawer.Dot.Y += h
	}

	oled.Clear()

	oled.Draw()

	err = oled.Display()
	if err != nil {
		fmt.Printf("error: %s\n", err)
	}

	return nil
}
