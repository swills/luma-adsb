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
			go buildDisplayInfoAndUpdateDisplay(&myADSBData, &stats, myLatFloat, myLonFloat, oledData)
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

	if len(myADSBData.Planes) > 0 {
		closestPlane, dist := adsb.FindClosest(*myADSBData, myLatFloat, myLonFloat)

		closest := strings.TrimSpace(closestPlane.CallSign)
		if closest == "" {
			closest = "none"
		}

		dispLines = append(dispLines, fmt.Sprintf("%s (%s)", closest, closestPlane.Hex))
		if closestPlane.Category != "" {
			dispLines = append(dispLines, fmt.Sprintf("%2.2fmi (%s)", dist, closestPlane.Category))
		} else {
			dispLines = append(dispLines, fmt.Sprintf("%2.2fmi", dist))
		}

		if _, ok := closestPlane.Altitude.(float64); ok {
			dispLines = append(dispLines, messagePrinter.Sprintf("%5.0fft", closestPlane.Altitude))
		}
	}

	oled.UpdateDisplayLines(dispLines, oledData)
}

func cleanup(oledData *goi2coled.I2c) {
	fmt.Printf("Clearing screen\n")
	oled.ClearDisplay(oledData)
	time.Sleep(time.Millisecond * 500) // wait for other go routines to finish

	_ = oledData.Close()
}
