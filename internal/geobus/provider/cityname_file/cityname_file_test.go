// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package cityname_file

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/wneessen/waybar-weather/internal/geobus"
	"github.com/wneessen/waybar-weather/internal/geocode"
)

const (
	testFile = "../../../../testdata/cityname"
	testLat  = 40.7185
	testLon  = -74.0025
)

func TestNewCitynameFileProvider(t *testing.T) {
	t.Run("new cityname file provider succeeds", func(t *testing.T) {
		provider := testProvider(t, testFile)
		if provider == nil {
			t.Fatal("expected provider to be non-nil")
		}
	})
	t.Run("new cityname file provider without geocoder fails", func(t *testing.T) {
		provider, err := NewCitynameFileProvider(testFile, nil)
		if err == nil {
			t.Fatal("expected provider to fail")
		}
		if provider != nil {
			t.Fatal("expected provider to be nil")
		}
	})
}

func TestCitynameFileProvider_Name(t *testing.T) {
	provider := testProvider(t, testFile)
	if provider == nil {
		t.Fatal("expected provider to be non-nil")
	}
	if !strings.EqualFold(provider.Name(), name) {
		t.Errorf("expected provider name to be %s, got %s", name, provider.Name())
	}
}

func TestNewCitynameFileProvider_readFile(t *testing.T) {
	t.Run("read file succeeds", func(t *testing.T) {
		provider := testProvider(t, testFile)
		if provider == nil {
			t.Fatal("expected provider to be non-nil")
		}
		coords, err := provider.readFile()
		if err != nil {
			t.Fatalf("failed to read file: %s", err)
		}
		if coords.Lat != testLat {
			t.Errorf("expected latitude to be %f, got %f", testLat, coords.Lat)
		}
		if coords.Lon != testLon {
			t.Errorf("expected longitude to be %f, got %f", testLon, coords.Lon)
		}
	})
	t.Run("read of non-existent file fails", func(t *testing.T) {
		provider := testProvider(t, "non-existent.txt")
		if provider == nil {
			t.Fatal("expected provider to be non-nil")
		}
		_, err := provider.readFile()
		if err == nil {
			t.Error("expected error, but didn't get one")
		}
	})
	t.Run("empty file fails", func(t *testing.T) {
		provider := testProvider(t, testFile+"_empty")
		if provider == nil {
			t.Fatal("expected provider to be non-nil")
		}
		_, err := provider.readFile()
		if err == nil {
			t.Error("expected error, but didn't get one")
		}
	})
	t.Run("geocoder lookup fails", func(t *testing.T) {
		provider := testProvider(t, testFile+"_fails")
		if provider == nil {
			t.Fatal("expected provider to be non-nil")
		}
		_, err := provider.readFile()
		if err == nil {
			t.Error("expected error, but didn't get one")
		}
	})
}

func TestCitynameFileProvider_createResult(t *testing.T) {
	provider := testProvider(t, testFile)
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

func TestCitynameFileProvider_LookupStream(t *testing.T) {
	t.Run("lookup stream succeeds", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			coder := new(mockCoder)
			provider, err := NewCitynameFileProvider(testFile, coder)
			if err != nil {
				t.Fatalf("failed to create cityname file provider: %s", err)
			}
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
			if result.AccuracyMeters != float64(geobus.AccuracyCity) {
				t.Errorf("expected accuracy to be %d, got %f", geobus.AccuracyCity, result.AccuracyMeters)
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

			coder := new(mockCoder)
			provider, err := NewCitynameFileProvider(testFile, coder)
			if err != nil {
				t.Fatalf("failed to create cityname file provider: %s", err)
			}
			if provider == nil {
				t.Fatal("expected provider to be non-nil")
			}
			provider.period = time.Millisecond * 10
			provider.locateFn = func() (geobus.Coordinate, error) {
				if runCount == 0 {
					runCount++
					return geobus.Coordinate{}, errors.New("intentionally failing")
				}
				return geobus.Coordinate{Lat: 1.0, Lon: 2.0}, nil
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
			if result.AccuracyMeters != geobus.AccuracyCity {
				t.Errorf("expected accuracy to be %d, got %f", geobus.AccuracyCity, result.AccuracyMeters)
			}
		})
	})
}

func testProvider(t *testing.T, file string) *CitynameFileProvider {
	t.Helper()
	coder := new(mockCoder)
	provider, err := NewCitynameFileProvider(file, coder)
	if err != nil {
		t.Fatalf("failed to create cityname file provider: %s", err)
	}
	return provider
}

type mockCoder struct{}

func (m *mockCoder) Name() string { return "mock" }
func (m *mockCoder) Reverse(_ context.Context, _ geobus.Coordinate) (geocode.Address, error) {
	return geocode.Address{}, errors.New("not implemented")
}

func (m *mockCoder) Search(_ context.Context, addr string) (geobus.Coordinate, error) {
	if addr == "Invalid, United Nations" {
		return geobus.Coordinate{}, errors.New("intentionally failing")
	}
	return geobus.Coordinate{Lat: testLat, Lon: testLon}, nil
}
