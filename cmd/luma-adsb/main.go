package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/swills/luma-adsb/internal/adsb"
	"github.com/swills/luma-adsb/internal/oled"
	goi2coled "github.com/waxdred/go-i2c-oled"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func main() {
	var err error

	var myLatFloat float64

	var myLonFloat float64

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

	myLatFloat, err = strconv.ParseFloat(myLatStr, 64)
	if err != nil {
		fmt.Printf("error parsing ADSBFEED_LAT: %s\n", err)

		initError = true
	}

	myLonStr := os.Getenv("ADSBFEED_LON")
	if myLonStr == "" {
		fmt.Printf("ADSBFEED_LON not set (probably something lke \"12.345678\")\n")

		initError = true
	}

	myLonFloat, err = strconv.ParseFloat(myLonStr, 64)
	if err != nil {
		fmt.Printf("error parsing ADSBFEED_LON: %s\n", err)

		initError = true
	}

	minAltStr := os.Getenv("ADSBFEED_MIN_ALT")
	if minAltStr == "" {
		fmt.Printf("ADSBFEED_MIN_ALT not set (probably something like \"50\")\n")
		initError = true
	}

	minAltFloat, err := strconv.ParseFloat(minAltStr, 64)
	if err != nil {
		fmt.Printf("error parsing ADSBFEED_MIN_ALT: %s\n", err)
		initError = true
	}

	maxAltStr := os.Getenv("ADSBFEED_MAX_ALT")
	if maxAltStr == "" {
		fmt.Printf("ADSBFEED_MAX_ALT not set (probably something like \"5000\")\n")
		initError = true
	}

	maxAltFloat, err := strconv.ParseFloat(maxAltStr, 64)
	if err != nil {
		fmt.Printf("error parsing ADSBFEED_MAX_ALT: %s\n", err)
		initError = true
	}

	maxDistStr := os.Getenv("ADSBFEED_MAX_DISTANCE")
	if maxDistStr == "" {
		fmt.Printf("ADSBFEED_MAX_DISTANCE not set (probably something like \"0.5\")\n")
		initError = true
	}

	maxDistFloat, err := strconv.ParseFloat(maxDistStr, 64)
	if err != nil {
		fmt.Printf("error parsing ADSBFEED_MAX_DISTANCE: %s\n", err)
		initError = true
	}

	if initError {
		os.Exit(1)
	}

	sigChan := make(chan os.Signal, 1)

	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGABRT, syscall.SIGBUS)

	oledData := oled.InitDisplay()

	stats := adsb.Stage2stats{
		Pps:         0,
		Mps:         0,
		Uptime:      0,
		Planes:      0,
		TotalPlanes: 0,
	}

	myADSBData := adsb.Data{
		Planes: make([]adsb.Aircraft, 0),
	}

	displayTicker := time.NewTicker(125 * time.Millisecond) // faster causes issues
	stage2Ticker := time.NewTicker(1 * time.Second)
	aircraftDataTicker := time.NewTicker(1 * time.Second)

	go func() {
		<-sigChan
		displayTicker.Stop()
		stage2Ticker.Stop()
		aircraftDataTicker.Stop()
		cleanup(oledData)
		os.Exit(0)
	}()

	for {
		select {
		case <-stage2Ticker.C:
			go getAndUpdateStats(&stats, host)
		case <-aircraftDataTicker.C:
			go getAndUpdateADSBData(&myADSBData, host)
		case <-displayTicker.C:
			go buildDisplayInfoAndUpdateDisplay(
				&myADSBData,
				&stats,
				myLatFloat,
				myLonFloat,
				minAltFloat,
				maxAltFloat,
				maxDistFloat,
				oledData)
		}
	}
}

func getAndUpdateADSBData(data *adsb.Data, host string) {
	newADSBData, err := adsb.GetADSBData(host)
	if err != nil {
		fmt.Printf("error getting adsb data: %s", err)
	} else {
		*data = *newADSBData
	}
}

func getAndUpdateStats(stats *adsb.Stage2stats, host string) {
	newStats, err := adsb.GetStage2Stats(host)
	if err != nil {
		fmt.Printf("error: %s\n", err)
	} else {
		*stats = *newStats
	}
}

var messagePrinter = message.NewPrinter(language.English)

func buildDisplayInfoAndUpdateDisplay(
	myADSBData *adsb.Data,
	stats *adsb.Stage2stats,
	myLatFloat float64,
	myLonFloat float64,
	minAltitude float64,
	maxAltitude float64,
	maxDistance float64,
	oledData *goi2coled.I2c,
) {
	var numPlanes int

	if len(myADSBData.Planes) > 0 {
		numPlanes = len(myADSBData.Planes)
	} else {
		numPlanes = stats.Planes
	}

	dispLines := []string{
		fmt.Sprintf("%s P: %d", time.Now().Format("15:04:05"), numPlanes),
	}

	var audible bool

	if len(myADSBData.Planes) > 0 {
		closestPlane, dist := adsb.FindClosest(*myADSBData, myLatFloat, myLonFloat)

		if closestPlane.Category != "" {
			minAltitudeCat, maxAltitudeCat, maxDistanceCat := getCategoryOverrides(closestPlane.Category)
			if minAltitudeCat != 0 {
				minAltitude = minAltitudeCat
			}
			if maxAltitudeCat != 0 {
				minAltitude = maxAltitudeCat
			}
			if maxDistanceCat != 0 {
				maxDistance = maxDistanceCat
			}
		}

		altitudeNum, valid := closestPlane.Altitude.(float64)
		if !valid {
			switch closestPlane.Altitude.(type) {
			case nil:
				altitudeNum = 0
			case string:
				if closestPlane.Altitude == "ground" {
					altitudeNum = 0
				}
			}
		}

		if altitudeNum >= minAltitude && altitudeNum <= maxAltitude && dist < maxDistance {
			audible = true
		}

		closest := strings.TrimSpace(closestPlane.CallSign)
		if closest == "" {
			closest = "none"
		}

		dispLines = append(dispLines, fmt.Sprintf("%s (%s)", closest, closestPlane.Hex))
		if closestPlane.Category != "" {
			dispLines = append(dispLines, fmt.Sprintf("%3.1fmi (%s)", dist, closestPlane.Category))
		} else {
			dispLines = append(dispLines, fmt.Sprintf("%3.1fmi", dist))
		}

		if _, ok := closestPlane.Altitude.(float64); ok {
			if audible {
				dispLines = append(dispLines, messagePrinter.Sprintf("%5.0fft (close)", closestPlane.Altitude))
			} else {
				dispLines = append(dispLines, messagePrinter.Sprintf("%5.0fft", closestPlane.Altitude))
			}
		}
	}

	oled.UpdateDisplayLines(dispLines, oledData)
}

func cleanup(oledData *goi2coled.I2c) {
	fmt.Printf("Clearing screen\n")
	oled.ClearDisplay(oledData)
	time.Sleep(time.Millisecond * 500) // wait for other go routines to finish

	_, _ = oledData.DisplayOff()

	_ = oledData.Close()
}

func getCategoryOverrides(category string) (float64, float64, float64) {
	var minAltFloat float64

	var maxAltFloat float64

	var maxDistFloat float64

	minAltStr := os.Getenv(fmt.Sprintf("ADSBFEED_MIN_ALT_CATEGORY_%s", category))
	if minAltStr != "" {
		minAltFloatTmp, err := strconv.ParseFloat(minAltStr, 64)
		if err == nil {
			minAltFloat = minAltFloatTmp
		}
	}

	maxAltStr := os.Getenv(fmt.Sprintf("ADSBFEED_MAX_ALT_CATEGORY_%s", category))
	if maxAltStr != "" {
		maxAltFloatTmp, err := strconv.ParseFloat(maxAltStr, 64)
		if err == nil {
			maxAltFloat = maxAltFloatTmp
		}
	}

	distAltStr := os.Getenv(fmt.Sprintf("ADSBFEED_MAX_DISTANCE_CATEGORY_%s", category))
	if distAltStr != "" {
		distAltFloatTmp, err := strconv.ParseFloat(distAltStr, 64)
		if err == nil {
			maxDistFloat = distAltFloatTmp
		}
	}

	return minAltFloat, maxAltFloat, maxDistFloat
}
