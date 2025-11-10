// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package geobus

// GeolocationState tracks the last known geolocation coordinates and accuracy values.
// It provides functionality to detect changes in geolocation data.
type GeolocationState struct {
	lastLat, lastLon float64
	lastAlt, lastAcc float64
	haveLast         bool
}

// HasChanged determines if the given geolocation data (lat, lon, alt, acc) differs from the stored state.
func (s *GeolocationState) HasChanged(lat, lon, alt, acc float64) bool {
	if !s.haveLast {
		return true
	}
	return lat != s.lastLat || lon != s.lastLon || alt != s.lastAlt || acc != s.lastAcc
}

// Update updates the stored geolocation state with the provided latitude, longitude, altitude, and
// accuracy values.
func (s *GeolocationState) Update(lat, lon, alt, acc float64) {
	s.lastLat, s.lastLon, s.lastAlt, s.lastAcc = lat, lon, alt, acc
	s.haveLast = true
}
