package main

import (
	"context"
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
	initError, host, myLatFloat, myLonFloat, myAltFloat, minAltFloat, maxAltFloat, maxDistFloat := initEnv()

	if initError {
		os.Exit(1)
	}

	sigChan := make(chan os.Signal, 1)

	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGABRT, syscall.SIGBUS)

	oledData := oled.InitDisplay()

	myADSBData := adsb.Data{
		Planes: make([]adsb.Aircraft, 0),
	}

	var feederStatus map[string]adsb.FeederInfo

	var updateStatus bool

	var cpuTempC int

	ctx := context.Background()

	displayUpdateInterval := 125 * time.Millisecond // faster causes issues
	aircraftDataInterval := 500 * time.Millisecond
	feederStatusInterval := 30 * time.Second
	updateStatusInterval := 5 * time.Minute
	updateCPUTempInterval := 1 * time.Minute

	displayTicker := time.NewTicker(displayUpdateInterval)
	aircraftDataTicker := time.NewTicker(aircraftDataInterval)
	feederStatusTicker := time.NewTicker(feederStatusInterval)
	updateStatusTicker := time.NewTicker(updateStatusInterval)
	updateCPUTempTicker := time.NewTicker(updateCPUTempInterval)

	go func() {
		<-sigChan
		displayTicker.Stop()
		aircraftDataTicker.Stop()
		cleanup(oledData)
		os.Exit(0)
	}()
	go updateFeederStatus(ctx, &feederStatus, host, feederStatusInterval/2)
	go updateUpdateStatus(ctx, &updateStatus, host, updateStatusInterval/2)
	go updateCPUTemp(ctx, &cpuTempC, host, updateCPUTempInterval)

	for {
		select {
		case <-aircraftDataTicker.C:
			go getAndUpdateADSBData(ctx, &myADSBData, host, aircraftDataInterval/2)
		case <-displayTicker.C:
			go buildDisplayInfoAndUpdateDisplay(
				&myADSBData,
				&feederStatus,
				&updateStatus,
				&cpuTempC,
				myLatFloat,
				myLonFloat,
				myAltFloat,
				minAltFloat,
				maxAltFloat,
				maxDistFloat,
				oledData)
		case <-feederStatusTicker.C:
			go updateFeederStatus(ctx, &feederStatus, host, feederStatusInterval/2)
		case <-updateStatusTicker.C:
			go updateUpdateStatus(ctx, &updateStatus, host, updateStatusInterval/2)
		case <-updateCPUTempTicker.C:
			go updateCPUTemp(ctx, &cpuTempC, host, updateCPUTempInterval)
		}
	}
}

//nolint:funlen
func initEnv() (bool, string, float64, float64, float64, float64, float64, float64) {
	var err error

	var initError bool

	var myLatFloat float64

	var myLonFloat float64

	var myAltFloat float64

	host := os.Getenv("LUMAADSB_HOST")
	if host == "" {
		fmt.Printf("LUMAADSB_HOST not set (probably YOURHOSTNAME)\n")

		initError = true
	}

	myLatStr := os.Getenv("LUMAADSB_LAT")
	if myLatStr == "" {
		fmt.Printf("LUMAADSB_LAT not set (probably something like \"12.345678\")\n")

		initError = true
	}

	myLatFloat, err = strconv.ParseFloat(myLatStr, 64)
	if err != nil {
		fmt.Printf("error parsing LUMAADSB_LAT: %s\n", err)

		initError = true
	}

	myLonStr := os.Getenv("LUMAADSB_LON")
	if myLonStr == "" {
		fmt.Printf("LUMAADSB_LON not set (probably something lke \"12.345678\")\n")

		initError = true
	}

	myLonFloat, err = strconv.ParseFloat(myLonStr, 64)
	if err != nil {
		fmt.Printf("error parsing LUMAADSB_LON: %s\n", err)

		initError = true
	}

	myAltStr := os.Getenv("LUMAADSB_ALT")
	if myAltStr == "" {
		fmt.Printf("LUMAADSB_ALT not set (probably something lke \"12345\")\n")

		initError = true
	}

	myAltFloat, err = strconv.ParseFloat(myAltStr, 64)
	if err != nil {
		fmt.Printf("error parsing LUMAADSB_ALT: %s\n", err)

		initError = true
	}

	minAltStr := os.Getenv("LUMAADSB_MIN_ALT")
	if minAltStr == "" {
		fmt.Printf("LUMAADSB_MIN_ALT not set (probably something like \"50\")\n")

		initError = true
	}

	minAltFloat, err := strconv.ParseFloat(minAltStr, 64)
	if err != nil {
		fmt.Printf("error parsing LUMAADSB_MIN_ALT: %s\n", err)

		initError = true
	}

	maxAltStr := os.Getenv("LUMAADSB_MAX_ALT")
	if maxAltStr == "" {
		fmt.Printf("LUMAADSB_MAX_ALT not set (probably something like \"5000\")\n")

		initError = true
	}

	maxAltFloat, err := strconv.ParseFloat(maxAltStr, 64)
	if err != nil {
		fmt.Printf("error parsing LUMAADSB_MAX_ALT: %s\n", err)

		initError = true
	}

	maxDistStr := os.Getenv("LUMAADSB_MAX_DISTANCE")
	if maxDistStr == "" {
		fmt.Printf("LUMAADSB_MAX_DISTANCE not set (probably something like \"0.5\")\n")

		initError = true
	}

	maxDistFloat, err := strconv.ParseFloat(maxDistStr, 64)
	if err != nil {
		fmt.Printf("error parsing LUMAADSB_MAX_DISTANCE: %s\n", err)

		initError = true
	}

	return initError, host, myLatFloat, myLonFloat, myAltFloat, minAltFloat, maxAltFloat, maxDistFloat
}

func getAndUpdateADSBData(ctx context.Context, data *adsb.Data, host string, timeout time.Duration) {
	newADSBData, err := adsb.GetADSBData(ctx, host, timeout)
	if err != nil {
		fmt.Printf("error getting adsb data: %s\n", err)
	} else {
		*data = *newADSBData
	}
}

func updateFeederStatus(ctx context.Context, feederStatus *map[string]adsb.FeederInfo, host string, timeout time.Duration) {
	config, err := adsb.GetMicroConfig(ctx, host, timeout)
	if err != nil {
		fmt.Printf("error getting micro config: %s\n", err)
		return
	}

	newFeederStatus, err := adsb.GetAllFeederStatus(ctx, host, timeout, config)
	if err != nil {
		fmt.Printf("error getting feeder status info: %s\n", err)
	} else {
		*feederStatus = *newFeederStatus
	}
}

func updateUpdateStatus(ctx context.Context, updateStatus *bool, host string, timeout time.Duration) {
	updateAvailable, err := adsb.GetUpdateAvailable(ctx, host, timeout)
	if err != nil {
		fmt.Printf("error getting update status: %s\n", err)
	} else {
		*updateStatus = updateAvailable
	}
}

func updateCPUTemp(ctx context.Context, cpuTempC *int, host string, timeout time.Duration) {
	newCPUTempC, err := adsb.GetCPUTempC(ctx, host, timeout)
	if err != nil {
		fmt.Printf("error getting CPU Temp: %s\n", err)
	} else {
		*cpuTempC = newCPUTempC
	}
}

var messagePrinter = message.NewPrinter(language.English)

func buildDisplayInfoAndUpdateDisplay(
	myADSBData *adsb.Data,
	feederStatus *map[string]adsb.FeederInfo,
	updateStatus *bool,
	cpuTemp *int,
	myLatFloat float64,
	myLonFloat float64,
	myAltFloat float64,
	minAltitude float64,
	maxAltitude float64,
	maxDistance float64,
	oledData *goi2coled.I2c,
) {
	var numPlanes int

	var numPlanesWithPos int

	numPlanes = len(myADSBData.Planes)
	for _, v := range myADSBData.Planes {
		if v.Latitude != 0 || v.Longitude != 0 || v.Last.Latitude != 0 || v.Last.Longitude != 0 {
			numPlanesWithPos++
		}
	}
	numPlanes = len(myADSBData.Planes)

	dispLines := []string{
		fmt.Sprintf("%s    %2d(%2d)", time.Now().Format("15:04:05"), numPlanes, numPlanesWithPos),
	}

	var audible bool

	if len(myADSBData.Planes) > 0 {
		dispLines = addClosest(myADSBData, feederStatus, updateStatus, cpuTemp, myLatFloat, myLonFloat, myAltFloat, minAltitude,
			maxDistance, maxAltitude, audible, dispLines)
	}

	oled.UpdateDisplayLines(dispLines, oledData)
}

//nolint:cyclop
func addClosest(
	myADSBData *adsb.Data,
	feederStatus *map[string]adsb.FeederInfo,
	updateStatus *bool,
	cpuTemp *int,
	myLatFloat float64,
	myLonFloat float64,
	myAltFloat float64,
	minAltitude float64,
	maxDistance float64,
	maxAltitude float64,
	audible bool,
	dispLines []string,
) []string {
	closestPlane, dist := adsb.FindClosest(*myADSBData, myLatFloat, myLonFloat, myAltFloat)

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
	//nolint:nestif
	if closestPlane.Hex != "" {
		if closest == "" {
			closest = "none"
		}

		dispLines = append(dispLines, messagePrinter.Sprintf("%s (%s)", closest, closestPlane.Hex))

		if closestPlane.Category != "" {
			dispLines = append(dispLines, messagePrinter.Sprintf("%4.1fmi (%s)    %dC", dist, closestPlane.Category, *cpuTemp))
		} else {
			dispLines = append(dispLines, messagePrinter.Sprintf("%4.1fmi         %dC", dist, *cpuTemp))
		}

		var goodCount int

		var badCount int

		if _, ok := closestPlane.Altitude.(float64); ok {
			for _, v := range *feederStatus {
				if v.Enabled {
					if v.BeastStatus == "good" {
						goodCount++
					} else if v.BeastStatus != "unknown" {
						badCount++
					}
					if v.MLATStatus == "good" {
						goodCount++
					} else if v.MLATStatus != "disabled" {
						badCount++
					}
				}
			}
			if *updateStatus {
				dispLines = append(dispLines, messagePrinter.Sprintf("%6.0fft   %2d(%2d)U", closestPlane.Altitude, goodCount, badCount))
			} else {
				dispLines = append(dispLines, messagePrinter.Sprintf("%6.0fft   %2d(%2d)", closestPlane.Altitude, goodCount, badCount))
			}
		}
	}

	return dispLines
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

	minAltStr := os.Getenv("LUMAADSB_MIN_ALT_CATEGORY_" + category)
	if minAltStr != "" {
		minAltFloatTmp, err := strconv.ParseFloat(minAltStr, 64)
		if err == nil {
			minAltFloat = minAltFloatTmp
		}
	}

	maxAltStr := os.Getenv("LUMAADSB_MAX_ALT_CATEGORY_" + category)
	if maxAltStr != "" {
		maxAltFloatTmp, err := strconv.ParseFloat(maxAltStr, 64)
		if err == nil {
			maxAltFloat = maxAltFloatTmp
		}
	}

	distAltStr := os.Getenv("LUMAADSB_MAX_DISTANCE_CATEGORY_" + category)
	if distAltStr != "" {
		distAltFloatTmp, err := strconv.ParseFloat(distAltStr, 64)
		if err == nil {
			maxDistFloat = distAltFloatTmp
		}
	}

	return minAltFloat, maxAltFloat, maxDistFloat
}
