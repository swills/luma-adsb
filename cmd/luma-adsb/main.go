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
	"log/slog"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jftuga/geodist"
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

	var initError bool

	host := os.Getenv("ADSBFEED_HOST")
	if host == "" {
		fmt.Printf("ADSBFEED_HOST not set (probably YOURHOSTNAME)\n")
		initError = true
	}

	myLatStr := os.Getenv("ADSBFEED_LAT")
	if myLatStr == "" {
		fmt.Printf("ADSBFEED_LAT not set (probably something like \"12.345678\")\n")
		initError = true
	}

	myLatFloat, err := strconv.ParseFloat(myLatStr, 64)
	if err != nil {
		fmt.Printf("error parsing ADSBFEED_LAT: %s\n", err)

		initError = true
	}

	myLonStr := os.Getenv("ADSBFEED_LON")
	if myLonStr == "" {
		fmt.Printf("ADSBFEED_LON not set (probably something lke \"12.345678\")\n")
		initError = true
	}

	myLonFloat, err := strconv.ParseFloat(myLonStr, 64)
	if err != nil {
		fmt.Printf("error parsing ADSBFEED_LON: %s\n", err)
		initError = true
	}

	if initError {
		os.Exit(1)
	}

	stats := stage2stats{
		Pps:         0,
		Mps:         0,
		Uptime:      0,
		Planes:      0,
		TotalPlanes: 0,
	}

	myADSBData := ADSBData{
		Planes: make([]Aircraft, 0),
	}

	displayTicker := time.NewTicker(500 * time.Millisecond) // faster than 300ms causes issues
	stage2Ticker := time.NewTicker(1 * time.Second)
	aircraftDataTicker := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-stage2Ticker.C:
			go getAndUpdateStats(&stats, host)
		case <-aircraftDataTicker.C:
			go getAndUpdateADSBData(&myADSBData, host)
		case <-displayTicker.C:
			go buildDisplayInfoAndUpdateDisplay(&myADSBData, &stats, myLatFloat, myLonFloat, oled)
		}
	}
}

func getAndUpdateADSBData(data *ADSBData, host string) {
	newADSBData, err := getADSBData(host)
	if err != nil {
		fmt.Printf("error getting adsb data: %s", err)
	} else {
		*data = *newADSBData
	}
}

func getADSBData(host string) (*ADSBData, error) {
	var err error

	var req *http.Request

	var res *http.Response

	url := fmt.Sprintf("http://%s:8080/data/aircraft.json", host)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()

	req, err = http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return &ADSBData{}, fmt.Errorf("error creating http req: %w", err)
	}

	res, err = http.DefaultClient.Do(req)
	if err != nil {
		return &ADSBData{}, fmt.Errorf("error making http request: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		slog.Error("bad status making request", "status", res.StatusCode)
		return &ADSBData{}, fmt.Errorf("bad status making request: %w", err)
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(res.Body)

	var body []byte

	body, err = io.ReadAll(res.Body)
	if err != nil {
		return &ADSBData{}, fmt.Errorf("error reading response body: %w", err)
	}

	var myADSBData ADSBData

	err = json.Unmarshal(body, &myADSBData)
	if err != nil {
		return &ADSBData{}, fmt.Errorf("failed unmarshalling: %w", err)
	}

	return &myADSBData, nil
}

func findClosest(myADSBData ADSBData, myLatFloat float64, myLonFloat float64) (Aircraft, float64) {
	myLoc := geodist.Coord{Lat: myLatFloat, Lon: myLonFloat}

	var closestDist = math.MaxFloat64
	var closestPlane Aircraft

	for _, flight := range myADSBData.Planes {
		planeLoc := geodist.Coord{Lat: flight.Latitude, Lon: flight.Longitude}
		distanceMiles, _ := geodist.HaversineDistance(myLoc, planeLoc)
		if distanceMiles < closestDist {
			closestDist = distanceMiles
			closestPlane = flight
		}
	}

	return closestPlane, closestDist
}

func getAndUpdateStats(stats *stage2stats, host string) {
	newStats, err := getStage2Stats(host)
	if err != nil {
		fmt.Printf("error: %s\n", err)
	} else {
		*stats = *newStats
	}
}

func getStage2Stats(host string) (*stage2stats, error) {
	var err error

	var req *http.Request

	var res *http.Response

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
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

func buildDisplayInfoAndUpdateDisplay(myADSBData *ADSBData, stats *stage2stats, myLatFloat float64, myLonFloat float64, oled *goi2coled.I2c) {
	var err error

	var numPlanes int

	if len(myADSBData.Planes) > 0 {
		numPlanes = len(myADSBData.Planes)
	} else {
		numPlanes = stats.Planes
	}

	dispLines := []string{
		fmt.Sprintf("%s P: %d", time.Now().Format("15:04:05"), numPlanes),
	}

	if len(myADSBData.Planes) > 0 {
		closestPlane, dist := findClosest(*myADSBData, myLatFloat, myLonFloat)
		closest := strings.TrimSpace(closestPlane.CallSign)
		if closest == "" {
			closest = "none"
		}
		dispLines = append(dispLines, fmt.Sprintf("C: %s (%s)", closest, closestPlane.Hex))
		if closestPlane.Category != "" {
			dispLines = append(dispLines, fmt.Sprintf("D: %2.2f (%s)", dist, closestPlane.Category))
		} else {
			dispLines = append(dispLines, fmt.Sprintf("D: %2.2f", dist))
		}
	}

	err = updateDisplayLines(dispLines, oled)
	if err != nil {
		fmt.Printf("error updating display: %s", err)
	}
}

func clearDisplay(oled *goi2coled.I2c) {
	draw.Draw(oled.Img, oled.Img.Bounds(), &image.Uniform{C: color.Black}, image.Point{}, draw.Src)
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

	clearDisplay(oled)

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
