// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package geobus

// GeolocationState tracks the last known geolocation coordinates and accuracy values.
// It provides functionality to detect changes in geolocation data.
type GeolocationState struct {
	last     Coordinate
	haveLast bool
}

// HasChanged determines if the given geolocation data (lat, lon, alt, acc) differs from the stored state.
func (s *GeolocationState) HasChanged(other Coordinate) bool {
	if !s.haveLast {
		return true
	}
	if !s.last.PosHasSignificantChange(other) {
		return false
	}
	return other.Lat != s.last.Lat || other.Lon != s.last.Lon || other.Alt != s.last.Alt || other.Acc != s.last.Acc
}

// Update updates the stored geolocation state with the provided latitude, longitude, altitude, and
// accuracy values.
func (s *GeolocationState) Update(new Coordinate) {
	s.last = new
	s.haveLast = true
}
