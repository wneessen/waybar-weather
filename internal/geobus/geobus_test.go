// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package geobus

import "testing"

func TestGeolocationState_HasChanged(t *testing.T) {
	t.Run("empty state always returns true", func(t *testing.T) {
		state := GeolocationState{}
		if !state.HasChanged(Coordinate{Lat: 1, Lon: 1, Acc: AccuracyZip}) {
			t.Error("expected state to have changed")
		}
	})
	t.Run("same coordinate return false", func(t *testing.T) {
		state := GeolocationState{}
		state.Update(Coordinate{Lat: 1, Lon: 1, Acc: AccuracyZip})
		if state.HasChanged(Coordinate{Lat: 1, Lon: 1, Acc: AccuracyZip}) {
			t.Error("expected state to not have changed")
		}
	})
	t.Run("different coordinate return true", func(t *testing.T) {
		tests := []struct {
			name    string
			lat     float64
			lon     float64
			acc     float64
			changed bool
		}{
			{"lat changes", 2, 1, AccuracyZip, true},
			{"lon changes", 1, 2, AccuracyZip, true},
			// an accuracy change is not considered a significant positional change
			{"acc changes", 1, 1, AccuracyCity, false},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				state := GeolocationState{}
				state.Update(Coordinate{Lat: 1, Lon: 1, Acc: AccuracyZip})
				if state.HasChanged(Coordinate{Lat: tc.lat, Lon: tc.lon, Acc: tc.acc}) != tc.changed {
					t.Error("expected state change to be", tc.changed, "but it wasn't")
				}
			})
		}
	})
}
