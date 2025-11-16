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

// Accuracy is the default accuracy value for geolocation data. We consider geolocation file data as
// the most accurate data available.
const Accuracy = 5

// GeolocationFileProvider reads geolocation data from a file and emits updates via a stream.
// It periodically reads a specified file, parses its data, and updates geolocation results based on changes.
// Each result includes details about the location, accuracy, confidence, and timestamp of the data.
// Results are subject to a time-to-live (TTL) duration, ensuring outdated data is discarded.
type GeolocationFileProvider struct {
	name   string
	path   string
	period time.Duration
	ttl    time.Duration
}

// NewGeolocationFileProvider initializes a GeolocationFileProvider with a file path and default update
// interval and TTL settings.
func NewGeolocationFileProvider(path string) *GeolocationFileProvider {
	return &GeolocationFileProvider{
		name:   "GeolocationFile",
		path:   path,
		period: 2 * time.Minute,
		ttl:    15 * time.Minute,
	}
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

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			lat, lon, err := p.readFile()
			if err != nil {
				// File missing or malformed — just retry later
				time.Sleep(p.period)
				continue
			}

			// Only emit if values changed or it's the first read
			if state.HasChanged(lat, lon, 0, Accuracy) {
				state.Update(lat, lon, 0, Accuracy)
				r := p.createResult(key, lat, lon)

				select {
				case <-ctx.Done():
					return
				case out <- r:
				}
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(p.period):
			}
		}
	}()
	return out
}

// createResult composes and returns a Result using provided geolocation data and metadata.
func (p *GeolocationFileProvider) createResult(key string, lat, lon float64) geobus.Result {
	return geobus.Result{
		Key:            key,
		Lat:            lat,
		Lon:            lon,
		AccuracyMeters: Accuracy,
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
	coords := strings.Split(strings.TrimSpace(string(data)), ",")
	if len(coords) != 2 {
		return 0, 0, fmt.Errorf("geolocation file %q contains invalid coordinates", p.path)
	}
	lat, err = strconv.ParseFloat(coords[0], 64)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse latitude from geolocation file %q: %w", p.path, err)
	}
	lon, err = strconv.ParseFloat(coords[1], 64)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse longitude from geolocation file %q: %w", p.path, err)
	}
	return lat, lon, nil
}
