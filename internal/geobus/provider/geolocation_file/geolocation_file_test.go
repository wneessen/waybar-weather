// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package geolocation_file

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/wneessen/waybar-weather/internal/geobus"
)

const (
	testFile = "../../../../testdata/geolocation"
	testLat  = 40.7185
	testLon  = -74.0025
)

func TestNewGeolocationFileProvider(t *testing.T) {
	t.Run("new geolocation file provider succeeds", func(t *testing.T) {
		provider := NewGeolocationFileProvider(testFile)
		if provider == nil {
			t.Fatal("expected provider to be non-nil")
		}
	})
}

func TestGeolocationFileProvider_Name(t *testing.T) {
	provider := NewGeolocationFileProvider(testFile)
	if provider == nil {
		t.Fatal("expected provider to be non-nil")
	}
	if !strings.EqualFold(provider.Name(), name) {
		t.Errorf("expected provider name to be %s, got %s", name, provider.Name())
	}
}

func TestNewGeolocationFileProvider_readFile(t *testing.T) {
	t.Run("read file succeeds", func(t *testing.T) {
		provider := NewGeolocationFileProvider(testFile)
		if provider == nil {
			t.Fatal("expected provider to be non-nil")
		}
		lat, lon, err := provider.readFile()
		if err != nil {
			t.Fatalf("failed to read file: %s", err)
		}
		if lat != testLat {
			t.Errorf("expected latitude to be %f, got %f", testLat, lat)
		}
		if lon != testLon {
			t.Errorf("expected longitude to be %f, got %f", testLon, lon)
		}
	})
	t.Run("read of non-existent file fails", func(t *testing.T) {
		provider := NewGeolocationFileProvider("non-existent.txt")
		if provider == nil {
			t.Fatal("expected provider to be non-nil")
		}
		_, _, err := provider.readFile()
		if err == nil {
			t.Error("expected error, but didn't get one")
		}
	})
	t.Run("reading invalid file fails", func(t *testing.T) {
		provider := NewGeolocationFileProvider(testFile + "_nocoord")
		if provider == nil {
			t.Fatal("expected provider to be non-nil")
		}
		_, _, err := provider.readFile()
		if err == nil {
			t.Error("expected error, but didn't get one")
		}
		if !errors.Is(err, ErrNoCoordinates) {
			t.Errorf("expected error to be %s, got %s", ErrNoCoordinates, err)
		}
	})
	t.Run("parsing invalid coordinates fails", func(t *testing.T) {
		tests := []struct {
			name string
			file string
		}{
			{"latitude", testFile + "_brokenlat"},
			{"longitude", testFile + "_brokenlon"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				provider := NewGeolocationFileProvider(tt.file)
				if provider == nil {
					t.Fatal("expected provider to be non-nil")
				}
				_, _, err := provider.readFile()
				if err == nil {
					t.Error("expected error, but didn't get one")
				}
				if !errors.Is(err, ErrNoCoordinates) {
					t.Errorf("expected error to be %s, got %s", ErrNoCoordinates, err)
				}
			})
		}
	})
}

func TestGeolocationFileProvider_createResult(t *testing.T) {
	provider := NewGeolocationFileProvider(testFile)
	if provider == nil {
		t.Fatal("expected provider to be non-nil")
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

func TestGeolocationFileProvider_LookupStream(t *testing.T) {
	t.Run("lookup stream succeeds", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			provider := NewGeolocationFileProvider(testFile)
			if provider == nil {
				t.Fatal("expected provider to be non-nil")
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

			provider := NewGeolocationFileProvider(testFile)
			if provider == nil {
				t.Fatal("expected provider to be non-nil")
			}
			provider.period = time.Millisecond * 10
			provider.locateFn = func() (float64, float64, error) {
				if runCount == 0 {
					runCount++
					return 0, 0, errors.New("intentionally failing")
				}
				return 1.0, 2.0, nil
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
			if result.AccuracyMeters != geobus.AccuracyZip {
				t.Errorf("expected accuracy to be %d, got %f", geobus.AccuracyZip, result.AccuracyMeters)
			}
		})
	})
}
