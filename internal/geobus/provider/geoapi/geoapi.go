// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package geoapi

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"app/internal/geobus"
	"app/internal/http"
)

const (
	APIEndpoint   = "https://geoapi.info/api/geo"
	LookupTimeout = time.Second * 5
)

const (
	AccuracyCountry = 300000
	AccuracyRegion  = 100000
	AccuracyCity    = 15000
	AccuracyZip     = 3000
	AccuarcyUnknown = 1000000

	ConvidenceCountry = 0.2
	ConvidenceRegion  = 0.5
	ConvidenceCity    = 0.7
	ConvidenceZip     = 0.85
	ConvidenceUnknown = 0.1
)

type GeolocationGeoAPIProvider struct {
	name   string
	http   *http.Client
	period time.Duration
	ttl    time.Duration
}

type APIResult struct {
	IP       string `json:"ip"`
	Location struct {
		CountryCode string `json:"country,omitempty"`
		Country     string `json:"countryName,omitempty"`
		Region      string `json:"region_name,omitempty"`
		City        string `json:"city,omitempty"`
		ZipCode     string `json:"postalCode,omitempty"`
		TimeZone    string `json:"timezone"`
		Coordinates struct {
			Latitude  string `json:"latitude"`
			Longitude string `json:"longitude"`
		} `json:"coordinates"`
	} `json:"location"`
}

func NewGeolocationGeoAPIProvider(http *http.Client) *GeolocationGeoAPIProvider {
	return &GeolocationGeoAPIProvider{
		name:   "geoapi",
		http:   http,
		period: 10 * time.Minute,
		ttl:    20 * time.Minute,
	}
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

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			lat, lon, acc, con, err := p.locate(ctx)
			if err != nil {
				time.Sleep(p.period)
				continue
			}

			// Only emit if values changed or it's the first read
			if state.HasChanged(lat, lon, 0, acc) {
				state.Update(lat, lon, 0, acc)
				r := p.createResult(key, lat, lon, acc, con)

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
func (p *GeolocationGeoAPIProvider) createResult(key string, lat, lon, acc, con float64) geobus.Result {
	return geobus.Result{
		Key:            key,
		Lat:            lat,
		Lon:            lon,
		AccuracyMeters: acc,
		Confidence:     con,
		Source:         p.name,
		At:             time.Now(),
		TTL:            p.ttl,
	}
}

func (p *GeolocationGeoAPIProvider) locate(ctx context.Context) (lat, lon, acc, con float64, err error) {
	ctxHttp, cancelHttp := context.WithTimeout(ctx, LookupTimeout)
	defer cancelHttp()

	result := new(APIResult)
	if _, err = p.http.Get(ctxHttp, APIEndpoint, result, nil); err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to get geolocation data from API: %w", err)
	}

	acc = AccuarcyUnknown
	con = ConvidenceUnknown
	if result.Location.CountryCode != "" {
		acc = AccuracyCountry
		con = ConvidenceCountry
	}
	if result.Location.Region != "" {
		acc = AccuracyRegion
		con = ConvidenceRegion
	}
	if result.Location.City != "" {
		acc = AccuracyCity
		con = ConvidenceCity
	}
	if result.Location.ZipCode != "" {
		acc = AccuracyZip
		con = ConvidenceZip
	}

	lat, err = strconv.ParseFloat(result.Location.Coordinates.Latitude, 64)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to parse latitude from API response: %w", err)
	}
	lon, err = strconv.ParseFloat(result.Location.Coordinates.Longitude, 64)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to parse longitude from API response: %w", err)
	}

	return lat, lon, acc, con, nil
}
