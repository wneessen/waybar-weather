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

// Update updates the stored geolocation state with the provided latitude, longitude, altitude, and
// accuracy values.
func (s *GeolocationState) Update(new Coordinate) {
	s.last = new
	s.haveLast = true
}
