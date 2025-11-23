// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package geolocation_file

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/wneessen/waybar-weather/internal/geobus"
)

const (
	name = "geolocation_file"
)

var ErrNoCoordinates = fmt.Errorf("no valid coordinates found in geolocation file")

// GeolocationFileProvider reads geolocation data from a file and emits updates via a stream.
// It periodically reads a specified file, parses its data, and updates geolocation results based on changes.
// Each result includes details about the location, accuracy, confidence, and timestamp of the data.
// Results are subject to a time-to-live (TTL) duration, ensuring outdated data is discarded.
type GeolocationFileProvider struct {
	name     string
	path     string
	period   time.Duration
	ttl      time.Duration
	locateFn func() (lat, lon float64, err error)
}

// NewGeolocationFileProvider initializes a GeolocationFileProvider with a file path and default update
// interval and TTL settings.
func NewGeolocationFileProvider(path string) *GeolocationFileProvider {
	provider := &GeolocationFileProvider{
		name:   name,
		path:   path,
		period: time.Minute * 2,
		ttl:    time.Hour * 1,
	}
	provider.locateFn = provider.readFile
	return provider
}

// Name returns the name of the GeolocationFileProvider instance.
func (p *GeolocationFileProvider) Name() string {
	return p.name
}

// LookupStream continuously streams geolocation results from a file, emitting updates when data changes
// or context ends.
func (p *GeolocationFileProvider) LookupStream(ctx context.Context, key string) <-chan geobus.Result {
	out := make(chan geobus.Result)
	go func() {
		defer close(out)
		state := geobus.GeolocationState{}
		firstRun := true

		for {
			if !firstRun {
				select {
				case <-ctx.Done():
					return
				case <-time.After(p.period):
				}
			}
			firstRun = false

			lat, lon, err := p.locateFn()
			if err != nil {
				continue
			}
			coord := geobus.Coordinate{Lat: lat, Lon: lon, Acc: geobus.AccuracyZip}

			// Only emit if values changed or it's the first read
			if state.HasChanged(coord) {
				state.Update(coord)
				r := p.createResult(key, coord)

				select {
				case <-ctx.Done():
					return
				case out <- r:
				}
			}
		}
	}()
	return out
}

// createResult composes and returns a Result using provided geolocation data and metadata.
func (p *GeolocationFileProvider) createResult(key string, coord geobus.Coordinate) geobus.Result {
	return geobus.Result{
		Key:            key,
		Lat:            coord.Lat,
		Lon:            coord.Lon,
		AccuracyMeters: coord.Acc,
		Source:         p.name,
		At:             time.Now(),
		TTL:            p.ttl,
	}
}

// readFile reads geolocation data from the file at the configured path.
// Returns latitude, longitude, altitude, accuracy, or an error if the file cannot be
// read or parsed correctly.
func (p *GeolocationFileProvider) readFile() (lat, lon float64, err error) {
	data, err := os.ReadFile(p.path)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read geolocation file %q: %w", p.path, err)
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			continue
		}
		coords := strings.Split(line, ",")
		if len(coords) != 2 {
			continue
		}
		lat, err = strconv.ParseFloat(strings.TrimSpace(coords[0]), 64)
		if err != nil {
			continue
		}
		lon, err = strconv.ParseFloat(strings.TrimSpace(coords[1]), 64)
		if err != nil {
			continue
		}
		return lat, lon, nil
	}
	return 0, 0, ErrNoCoordinates
}
