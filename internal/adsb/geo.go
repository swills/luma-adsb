package adsb

import (
	"math"

	"github.com/jftuga/geodist"
)

func FindClosest(myADSBData Data, myLatFloat float64, myLonFloat float64) (Aircraft, float64) {
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
