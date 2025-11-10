// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package geoip

import (
	"context"
	"fmt"
	"time"

	"app/internal/geobus"
	"app/internal/http"
)

const (
	APIEndpoint   = "https://reallyfreegeoip.org/json/"
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

type GeolocationGeoIPProvider struct {
	name   string
	http   *http.Client
	period time.Duration
	ttl    time.Duration
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

func NewGeolocationGeoIPProvider(http *http.Client) *GeolocationGeoIPProvider {
	return &GeolocationGeoIPProvider{
		name:   "geoip",
		http:   http,
		period: 30 * time.Minute,
		ttl:    60 * time.Minute,
	}
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
func (p *GeolocationGeoIPProvider) createResult(key string, lat, lon, acc, con float64) geobus.Result {
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

func (p *GeolocationGeoIPProvider) locate(ctx context.Context) (lat, lon, acc, con float64, err error) {
	ctxHttp, cancelHttp := context.WithTimeout(ctx, LookupTimeout)
	defer cancelHttp()

	result := new(APIResult)
	if _, err = p.http.Get(ctxHttp, APIEndpoint, result, nil); err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to get geolocation data from API: %w", err)
	}

	acc = AccuarcyUnknown
	con = ConvidenceUnknown
	if result.CountryCode != "" {
		acc = AccuracyCountry
		con = ConvidenceCountry
	}
	if result.RegionCode != "" {
		acc = AccuracyRegion
		con = ConvidenceRegion
	}
	if result.City != "" {
		acc = AccuracyCity
		con = ConvidenceCity
	}
	if result.ZipCode != "" {
		acc = AccuracyZip
		con = ConvidenceZip
	}

	return result.Latitude, result.Longitude, acc, con, nil
}
