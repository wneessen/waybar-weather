// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package cityname_file

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/wneessen/waybar-weather/internal/geobus"
	"github.com/wneessen/waybar-weather/internal/geocode"
)

const (
	name     = "cityname_file"
	ttlTime  = time.Hour * 12
	pollTime = time.Minute * 5
)

var ErrNoCoordinates = fmt.Errorf("no valid city name found in cityname file")

// CitynameFileProvider reads city data from a file and emits updates via a stream.
// It periodically reads a specified file, parses its data, and updates geolocation results based on changes.
// Each result includes details about the location, accuracy, confidence, and timestamp of the data.
// Results are subject to a time-to-live (TTL) duration, ensuring outdated data is discarded.
type CitynameFileProvider struct {
	name     string
	path     string
	period   time.Duration
	ttl      time.Duration
	coder    geocode.Geocoder
	locateFn func() (geobus.Coordinate, error)
}

// NewCitynameFileProvider initializes a CitynameFileProvider with a file path and default update
// interval and TTL settings.
func NewCitynameFileProvider(path string, coder geocode.Geocoder) (*CitynameFileProvider, error) {
	if coder == nil {
		return nil, errors.New("geocoder is required")
	}
	provider := &CitynameFileProvider{
		coder:  coder,
		name:   name,
		path:   path,
		period: pollTime,
		ttl:    ttlTime,
	}
	provider.locateFn = provider.readFile
	return provider, nil
}

// Name returns the name of the CitynameFileProvider instance.
func (p *CitynameFileProvider) Name() string {
	return p.name
}

// LookupStream continuously streams geolocation results from a file, emitting updates when data changes
// or context ends.
func (p *CitynameFileProvider) LookupStream(ctx context.Context, key string) <-chan geobus.Result {
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

			coords, err := p.locateFn()
			if err != nil {
				continue
			}
			coords.Acc = geobus.AccuracyCity
			state.Update(coords)
			r := p.createResult(key, coords)

			select {
			case <-ctx.Done():
				return
			case out <- r:
			}
		}
	}()
	return out
}

// createResult composes and returns a Result using provided geolocation data and metadata.
func (p *CitynameFileProvider) createResult(key string, coord geobus.Coordinate) geobus.Result {
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
func (p *CitynameFileProvider) readFile() (coords geobus.Coordinate, err error) {
	data, err := os.ReadFile(p.path)
	if err != nil {
		return coords, fmt.Errorf("failed to read cityname file %q: %w", p.path, err)
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			continue
		}

		coords, err = p.coder.Search(context.Background(), line)
		if err != nil {
			continue
		}
		return coords, nil
	}
	return coords, ErrNoCoordinates
}
