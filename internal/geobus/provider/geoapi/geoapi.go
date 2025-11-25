// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package geoapi

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/wneessen/waybar-weather/internal/geobus"
	"github.com/wneessen/waybar-weather/internal/http"
)

const (
	apiEndpoint  = "https://geoapi.info/api/geo"
	lookupTimeou = time.Second * 5
	name         = "geoapi"
)

type GeolocationGeoAPIProvider struct {
	name     string
	http     *http.Client
	period   time.Duration
	ttl      time.Duration
	locateFn func(ctx context.Context) (lat, lon, acc float64, err error)
}

type APIResult struct {
	IP       string `json:"ip"`
	Location struct {
		CountryCode string `json:"country,omitempty"`
		Country     string `json:"countryName,omitempty"`
		Region      string `json:"region,omitempty"`
		City        string `json:"city,omitempty"`
		ZipCode     string `json:"postalCode,omitempty"`
		TimeZone    string `json:"timezone"`
		Coordinates struct {
			Latitude  string `json:"latitude"`
			Longitude string `json:"longitude"`
		} `json:"coordinates"`
	} `json:"location"`
}

func NewGeolocationGeoAPIProvider(http *http.Client) (*GeolocationGeoAPIProvider, error) {
	if http == nil {
		return nil, fmt.Errorf("http client is required")
	}
	provider := &GeolocationGeoAPIProvider{
		name:   name,
		http:   http,
		period: time.Minute * 10,
		ttl:    time.Hour * 2,
	}
	provider.locateFn = provider.locate
	return provider, nil
}

func (p *GeolocationGeoAPIProvider) Name() string {
	return p.name
}

// LookupStream continuously streams geolocation results from a file, emitting updates when data changes
// or context ends.
func (p *GeolocationGeoAPIProvider) LookupStream(ctx context.Context, key string) <-chan geobus.Result {
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
			state.Update(coord)
			r := p.createResult(key, coord)

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
func (p *GeolocationGeoAPIProvider) createResult(key string, coord geobus.Coordinate) geobus.Result {
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

func (p *GeolocationGeoAPIProvider) locate(ctx context.Context) (lat, lon, acc float64, err error) {
	ctxHttp, cancelHttp := context.WithTimeout(ctx, lookupTimeou)
	defer cancelHttp()

	result := new(APIResult)
	if _, err = p.http.Get(ctxHttp, apiEndpoint, result, nil, nil); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get geolocation data from API: %w", err)
	}

	acc = geobus.AccuracyUnknown
	if result.Location.CountryCode != "" {
		acc = geobus.AccuracyCountry
	}
	if result.Location.Region != "" {
		acc = geobus.AccuracyRegion
	}
	if result.Location.City != "" {
		acc = geobus.AccuracyCity
	}
	if result.Location.ZipCode != "" {
		acc = geobus.AccuracyZip
	}

	lat, err = strconv.ParseFloat(result.Location.Coordinates.Latitude, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse latitude from API response: %w", err)
	}
	lon, err = strconv.ParseFloat(result.Location.Coordinates.Longitude, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse longitude from API response: %w", err)
	}

	return geobus.Truncate(lat, geobus.TruncPrecision),
		geobus.Truncate(lon, geobus.TruncPrecision),
		geobus.Truncate(acc, geobus.TruncPrecision), nil
}
