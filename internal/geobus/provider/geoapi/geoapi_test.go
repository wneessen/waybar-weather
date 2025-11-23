// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package geoapi

import (
	"errors"
	"log/slog"
	stdhttp "net/http"
	"os"
	"strings"
	"testing"

	"github.com/wneessen/waybar-weather/internal/geobus"
	"github.com/wneessen/waybar-weather/internal/http"
	"github.com/wneessen/waybar-weather/internal/logger"
	"github.com/wneessen/waybar-weather/internal/testhelper"
)

const (
	testLat = 40.7185
	testLon = -74.0025
	wantAcc = geobus.AccuracyZip
)

func TestNewGeolocationGeoAPIProvider(t *testing.T) {
	t.Run("new GeoAPI provider succeeds", func(t *testing.T) {
		provider, err := NewGeolocationGeoAPIProvider(http.New(logger.New(slog.LevelInfo)))
		if err != nil {
			t.Fatalf("failed to create GeoAPI provider: %s", err)
		}
		if provider == nil {
			t.Fatal("expected provider to be non-nil")
		}
	})
	t.Run("GeoAPI without http client fails ", func(t *testing.T) {
		provider, err := NewGeolocationGeoAPIProvider(nil)
		if err == nil {
			t.Fatal("expected provider to fail")
		}
		if provider != nil {
			t.Fatal("expected provider to be nil")
		}
	})
}

func TestGeolocationGeoAPIProvider_Name(t *testing.T) {
	provider, err := NewGeolocationGeoAPIProvider(http.New(logger.New(slog.LevelInfo)))
	if err != nil {
		t.Fatalf("failed to create GeoAPI provider: %s", err)
	}
	if !strings.EqualFold(provider.Name(), name) {
		t.Errorf("expected provider name to be %s, got %s", name, provider.Name())
	}
}

func TestNewGeolocationGeoAPIProvider_locate(t *testing.T) {
	t.Run("locate succeeds with different accuracies", func(t *testing.T) {
		tests := []struct {
			name string
			file string
		}{
			{name: "latitude", file: "../../../../testdata/geoapi_brokenlat.json"},
			{name: "longitude", file: "../../../../testdata/geoapi_brokenlon.json"},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				rtFn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
					data, err := os.Open(tc.file)
					if err != nil {
						t.Fatalf("failed to open JSON response file: %s", err)
					}

					return &stdhttp.Response{
						StatusCode: 200,
						Body:       data,
						Header:     make(stdhttp.Header),
					}, nil
				}
				client := http.New(logger.New(slog.LevelInfo))
				client.Transport = testhelper.MockRoundTripper{Fn: rtFn}
				provider, err := NewGeolocationGeoAPIProvider(client)
				if err != nil {
					t.Fatalf("failed to create GeoAPI provider: %s", err)
				}

				if _, _, _, err = provider.locate(t.Context()); err == nil {
					t.Error("expected locate to fail")
				}
			})
		}
	})
	t.Run("locate fails on invalid coordinate parsing", func(t *testing.T) {
		tests := []struct {
			name string
			file string
			want float64
		}{
			{name: "zip", file: "../../../../testdata/geoapi.json", want: wantAcc},
			{name: "city", file: "../../../../testdata/geoapi_nozip.json", want: geobus.AccuracyCity},
			{name: "region", file: "../../../../testdata/geoapi_nocity.json", want: geobus.AccuracyRegion},
			{name: "country", file: "../../../../testdata/geoapi_noregion.json", want: geobus.AccuracyCountry},
			{name: "unknown", file: "../../../../testdata/geoapi_nocountry.json", want: geobus.AccuracyUnknown},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				rtFn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
					data, err := os.Open(tc.file)
					if err != nil {
						t.Fatalf("failed to open JSON response file: %s", err)
					}

					return &stdhttp.Response{
						StatusCode: 200,
						Body:       data,
						Header:     make(stdhttp.Header),
					}, nil
				}
				client := http.New(logger.New(slog.LevelInfo))
				client.Transport = testhelper.MockRoundTripper{Fn: rtFn}
				provider, err := NewGeolocationGeoAPIProvider(client)
				if err != nil {
					t.Fatalf("failed to create GeoAPI provider: %s", err)
				}

				lat, lon, acc, err := provider.locate(t.Context())
				if err != nil {
					t.Fatalf("failed to locate coordinates via GeoAPI: %s", err)
				}
				if lat != testLat {
					t.Errorf("expected latitude to be %f, got %f", testLat, lat)
				}
				if lon != testLon {
					t.Errorf("expected longitude to be %f, got %f", testLon, lon)
				}
				if geobus.Truncate(acc, 1) != geobus.Truncate(tc.want, 1) {
					t.Errorf("expected accuracy to be %f, got %f", geobus.Truncate(tc.want, 1),
						geobus.Truncate(acc, 1))
				}
			})
		}
	})
	t.Run("locate fails on API request", func(t *testing.T) {
		rtFn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
			return nil, errors.New("intentionally failing")
		}
		client := http.New(logger.New(slog.LevelInfo))
		client.Transport = testhelper.MockRoundTripper{Fn: rtFn}
		provider, err := NewGeolocationGeoAPIProvider(client)
		if err != nil {
			t.Fatalf("failed to create GeoAPI provider: %s", err)
		}
		if _, _, _, err = provider.locate(t.Context()); err == nil {
			t.Error("expected locate to fail")
		}
	})
}
