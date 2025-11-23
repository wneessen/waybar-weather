// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package geoapi

import (
	"context"
	"errors"
	"log/slog"
	stdhttp "net/http"
	"os"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/wneessen/waybar-weather/internal/geobus"
	"github.com/wneessen/waybar-weather/internal/http"
	"github.com/wneessen/waybar-weather/internal/logger"
	"github.com/wneessen/waybar-weather/internal/testhelper"
)

const (
	testLat = 40.7185
	testLon = -74.0025
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
			{name: "zip", file: "../../../../testdata/geoapi.json", want: geobus.AccuracyZip},
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

func TestGeolocationGeoAPIProvider_createResult(t *testing.T) {
	provider, err := NewGeolocationGeoAPIProvider(http.New(logger.New(slog.LevelInfo)))
	if err != nil {
		t.Fatalf("failed to create GeoAPI provider: %s", err)
	}
	result := provider.createResult("test", geobus.Coordinate{Lat: testLat, Lon: testLon, Acc: geobus.AccuracyCity})
	if result.Lat != testLat {
		t.Errorf("expected latitude to be %f, got %f", testLat, result.Lat)
	}
	if result.Lon != testLon {
		t.Errorf("expected longitude to be %f, got %f", testLon, result.Lon)
	}
	if result.Key != "test" {
		t.Errorf("expected key to be %s, got %s", "test", result.Key)
	}
	if result.AccuracyMeters != geobus.AccuracyCity {
		t.Errorf("expected accuracy to be %d, got %f", geobus.AccuracyCity, result.AccuracyMeters)
	}
	if result.Source != provider.Name() {
		t.Errorf("expected source to be %s, got %s", provider.Name(), result.Source)
	}
	if result.TTL != provider.ttl {
		t.Errorf("expected TTL to be %d, got %d", provider.ttl, result.TTL)
	}
}

func TestGeolocationGeoAPIProvider_LookupStream(t *testing.T) {
	t.Run("lookup stream succeeds", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			rtFn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
				data, err := os.Open("../../../../testdata/geoapi.json")
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
			provider.ttl = time.Millisecond * 10
			provider.period = time.Millisecond * 10

			out := provider.LookupStream(ctx, "test")
			if out == nil {
				t.Fatal("expected stream to be non-nil")
			}

			var results []geobus.Result
			for len(results) < 1 {
				select {
				case r := <-out:
					results = append(results, r)
					cancel()
				default:
					// Block until all goroutines are durably blocked, then advance
					// fake time to the next wakeup (e.g. time.After/ Sleep).
					synctest.Wait()
				}
			}

			synctest.Wait()
			if len(results) != 1 {
				t.Fatalf("expected at least one result, got %d", len(results))
			}
			result := results[0]
			if result.Key != "test" {
				t.Errorf("expected key to be %s, got %s", "test", result.Key)
			}
			if result.Lat != testLat {
				t.Errorf("expected latitude to be %f, got %f", testLat, result.Lat)
			}
			if result.Lon != testLon {
				t.Errorf("expected longitude to be %f, got %f", testLon, result.Lon)
			}
			if result.AccuracyMeters != float64(geobus.AccuracyZip) {
				t.Errorf("expected accuracy to be %d, got %f", geobus.AccuracyZip, result.AccuracyMeters)
			}
			if result.Source != provider.Name() {
				t.Errorf("expected source to be %s, got %s", provider.Name(), result.Source)
			}
		})
	})
	t.Run("lookup stream fails during lookup", func(t *testing.T) {
		runCount := 0
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			provider, err := NewGeolocationGeoAPIProvider(http.New(logger.New(slog.LevelInfo)))
			if err != nil {
				t.Fatalf("failed to create GeoAPI provider: %s", err)
			}
			provider.period = time.Millisecond * 10
			provider.locateFn = func(ctx context.Context) (float64, float64, float64, error) {
				if runCount == 0 {
					runCount++
					return 0, 0, 0, errors.New("intentionally failing")
				}
				return 1.0, 2.0, 3.0, nil
			}

			out := provider.LookupStream(ctx, "test")
			if out == nil {
				t.Fatal("expected stream to be non-nil")
			}

			var result geobus.Result
			select {
			case r := <-out:
				result = r
				cancel()
			case <-ctx.Done():
				t.Fatalf("context done before result: %v", ctx.Err())
			}
			synctest.Wait()

			if result.Lat != 1.0 {
				t.Errorf("expected latitude to be %f, got %f", 1.0, result.Lat)
			}
			if result.Lon != 2.0 {
				t.Errorf("expected longitude to be %f, got %f", 2.0, result.Lon)
			}
			if result.AccuracyMeters != 3.0 {
				t.Errorf("expected accuracy to be %f, got %f", 3.0, result.AccuracyMeters)
			}
		})
	})
}
