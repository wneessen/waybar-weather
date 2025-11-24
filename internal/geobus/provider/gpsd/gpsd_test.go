// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package gpsd

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/wneessen/waybar-weather/internal/geobus"
	"github.com/wneessen/waybar-weather/internal/gpspoll"
)

const (
	testLat = 40.7185
	testLon = -74.0025
)

func TestNewGeolocationGPSDProvider(t *testing.T) {
	t.Run("new GPSd provider succeeds", func(t *testing.T) {
		provider := NewGeolocationGPSDProvider()
		if provider == nil {
			t.Fatal("expected provider to be non-nil")
		}
	})
}

func TestGeolocationGPSDProvider_Name(t *testing.T) {
	provider := NewGeolocationGPSDProvider()
	if !strings.EqualFold(provider.Name(), name) {
		t.Errorf("expected provider name to be %s, got %s", name, provider.Name())
	}
}

func TestGeolocationGPSDProvider_createResult(t *testing.T) {
	provider := NewGeolocationGPSDProvider()
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

func TestGeolocationGPSDProvider_LookupStream(t *testing.T) {
	t.Run("fetching GPS data fails on first run but then succeeds", func(t *testing.T) {
		runCount := 0
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			provider := NewGeolocationGPSDProvider()
			provider.period = time.Millisecond * 10
			provider.locateFn = func(ctx context.Context) (gpspoll.Fix, error) {
				if runCount == 0 {
					runCount++
					return gpspoll.Fix{}, errors.New("intentionally failing")
				}
				if runCount == 1 {
					runCount++
					return gpspoll.Fix{Lat: 1, Lon: 2, Acc: 3, Mode: 1}, nil
				}
				return gpspoll.Fix{Lat: 1.0, Lon: 2.0, Acc: 3.0, Mode: 2}, nil
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

/*
func TestNewGeolocationGPSDProvider_fetchGPSData(t *testing.T) {
	testRequiresGPSD(t)
}

*/
