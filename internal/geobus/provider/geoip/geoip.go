// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package geoip

import (
	"context"
	"fmt"
	"time"

	"github.com/wneessen/waybar-weather/internal/geobus"
	"github.com/wneessen/waybar-weather/internal/http"
)

const (
	APIEndpoint   = "https://reallyfreegeoip.org/json/"
	LookupTimeout = time.Second * 10
	name          = "geoip"
)

type GeolocationGeoIPProvider struct {
	name     string
	http     *http.Client
	period   time.Duration
	ttl      time.Duration
	locateFn func(ctx context.Context) (lat, lon, acc float64, err error)
}

type APIResult struct {
	IP          string  `json:"ip"`
	CountryCode string  `json:"country_code"`
	Country     string  `json:"country_name"`
	RegionCode  string  `json:"region_code,omitempty"`
	Region      string  `json:"region_name,omitempty"`
	City        string  `json:"city,omitempty"`
	ZipCode     string  `json:"zip_code,omitempty"`
	TimeZone    string  `json:"time_zone"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	MetroCode   int     `json:"metro_code"`
}

func NewGeolocationGeoIPProvider(http *http.Client) (*GeolocationGeoIPProvider, error) {
	if http == nil {
		return nil, fmt.Errorf("http client is required")
	}
	provider := &GeolocationGeoIPProvider{
		name:   name,
		http:   http,
		period: 30 * time.Minute,
		ttl:    60 * time.Minute,
	}
	provider.locateFn = provider.locate
	return provider, nil
}

func (p *GeolocationGeoIPProvider) Name() string {
	return p.name
}

// LookupStream continuously streams geolocation results from a file, emitting updates when data changes
// or context ends.
func (p *GeolocationGeoIPProvider) LookupStream(ctx context.Context, key string) <-chan geobus.Result {
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

			lat, lon, acc, err := p.locateFn(ctx)
			if err != nil {
				continue
			}
			coord := geobus.Coordinate{Lat: lat, Lon: lon, Acc: acc}

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
func (p *GeolocationGeoIPProvider) createResult(key string, coord geobus.Coordinate) geobus.Result {
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

func (p *GeolocationGeoIPProvider) locate(ctx context.Context) (lat, lon, acc float64, err error) {
	ctxHttp, cancelHttp := context.WithTimeout(ctx, LookupTimeout)
	defer cancelHttp()

	result := new(APIResult)
	if _, err = p.http.Get(ctxHttp, APIEndpoint, result, nil, nil); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get geolocation data from API: %w", err)
	}

	acc = geobus.AccuracyUnknown
	if result.CountryCode != "" {
		acc = geobus.AccuracyCountry
	}
	if result.RegionCode != "" {
		acc = geobus.AccuracyRegion
	}
	if result.City != "" {
		acc = geobus.AccuracyCity
	}
	if result.ZipCode != "" {
		acc = geobus.AccuracyZip
	}

	return geobus.Truncate(result.Latitude, geobus.TruncPrecision),
		geobus.Truncate(result.Longitude, geobus.TruncPrecision),
		geobus.Truncate(acc, geobus.TruncPrecision), nil
}
