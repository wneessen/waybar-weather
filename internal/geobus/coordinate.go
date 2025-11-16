// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package geobus

import (
	"math"
)

const (
	EarthRadius       = 6371000.0 // meters
	DistanceThreshold = 2500.0    // 2.5km
)

// Coordinate represents a geographic coordinate.
type Coordinate struct {
	Lat float64
	Lon float64
	Alt float64
	Acc float64
}

// PosHasSignificantChange checks if the geographic position differs significantly from
// another based on the distance threshold. We are using the Haversine formula to calculate
// great-circle distance between two points on a sphere (in our case: Earth).
func (c Coordinate) PosHasSignificantChange(other Coordinate) bool {
	dLat := (c.Lat - other.Lat) * math.Pi / 180
	dLon := (c.Lon - other.Lon) * math.Pi / 180
	lat1 := c.Lat * math.Pi / 180
	lat2 := other.Lat * math.Pi / 180
	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	distance := 2 * EarthRadius * math.Asin(math.Sqrt(h))

	return distance > DistanceThreshold
}
