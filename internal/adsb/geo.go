package adsb

import (
	"math"

	"github.com/jftuga/geodist"
)

func FindClosest(myADSBData Data, myLatFloat, myLonFloat, myAltFloat float64) (Aircraft, float64) {
	myLoc := geodist.Coord{Lat: myLatFloat, Lon: myLonFloat}

	var closestDist = math.MaxFloat64

	var closestPlane Aircraft

	for _, flight := range myADSBData.Planes {
		if flight.Latitude == 0 || flight.Longitude == 0 {
			continue
		}

		planeLoc := geodist.Coord{Lat: flight.Latitude, Lon: flight.Longitude}

		distanceMiles, _, err := geodist.VincentyDistance(myLoc, planeLoc)
		if err != nil {
			return closestPlane, 0
		}

		if planeAlt, ok := flight.Altitude.(float64); ok && flight.Altitude != 0 {
			distanceMiles = threeDDistance(distanceMiles, myAltFloat/5280.0, planeAlt/5280.0)
		}

		if distanceMiles < closestDist {
			closestDist = distanceMiles
			closestPlane = flight
		}
	}

	if closestDist == math.MaxFloat64 {
		closestDist = 0
	}

	return closestPlane, closestDist
}

// threeDDistance calculates the distance to the plane in 3d space. all 3 args must be in the
//
//	same units, probably miles
func threeDDistance(horizontalDistance, myAlt, objectAlt float64) float64 {
	altitudeDifference := myAlt - objectAlt

	distThreeD := math.Sqrt(horizontalDistance*horizontalDistance + altitudeDifference*altitudeDifference)

	return distThreeD
}
