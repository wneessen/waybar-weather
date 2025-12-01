// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package nominatim

import (
	"errors"
	"log/slog"
	stdhttp "net/http"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/text/language"

	"github.com/wneessen/waybar-weather/internal/geobus"
	"github.com/wneessen/waybar-weather/internal/geocode"
	"github.com/wneessen/waybar-weather/internal/http"
	"github.com/wneessen/waybar-weather/internal/logger"
	"github.com/wneessen/waybar-weather/internal/testhelper"
)

const (
	cityExpected      = "Quartier 205, 67, Friedrichstrasse, Friedrichstadt, Mitte, Berlin, 10117, Germany"
	cityFile          = "../../../../testdata/nominatim_berlin.json"
	cityFileBrokenLat = "../../../../testdata/nominatim_berlin_brokenlat.json"
	cityFileBrokenLon = "../../../../testdata/nominatim_berlin_brokenlon.json"
	testHitTTL        = 1 * time.Second
	testMissTTL       = 1 * time.Second

	villageExpected = "Marshfield"
	villageFile     = "../../../../testdata/nominatim_marshfield.json"

	townExpected = "Otley"
	townFile     = "../../../../testdata/nominatim_otley.json"
)

var (
	cityCoords    = geobus.Coordinate{Lat: 52.5129, Lon: 13.3910}
	villageCoords = geobus.Coordinate{Lat: 51.46292, Lon: -2.31850}
	townCoords    = geobus.Coordinate{Lat: 53.90712, Lon: -1.69404}
)

func TestNew(t *testing.T) {
	t.Run("creating a new provider succeeds", func(t *testing.T) {
		coder := testCoder(t)
		if coder == nil {
			t.Fatal("expected a non-nil geocoder")
		}
	})
	t.Run("provider name is correct", func(t *testing.T) {
		coder := testCoder(t)
		if coder.Name() != name {
			t.Errorf("expected provider name to be %q, got %q", name, coder.Name())
		}
	})
}

func TestNominatim_Reverse(t *testing.T) {
	t.Run("reverse geocoding succeeds", func(t *testing.T) {
		rtFn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
			data, err := os.Open(cityFile)
			if err != nil {
				t.Fatalf("failed to open JSON response file: %s", err)
			}

			return &stdhttp.Response{
				StatusCode: 200,
				Body:       data,
				Header:     make(stdhttp.Header),
			}, nil
		}

		coder := testCoderWithRoundtripFunc(t, rtFn)
		addr, err := coder.Reverse(t.Context(), cityCoords)
		if err != nil {
			t.Fatal(err)
		}
		if !addr.AddressFound {
			t.Fatal("expected address to be found")
		}
		if !strings.EqualFold(addr.DisplayName, cityExpected) {
			t.Errorf("expected address to be %q, got %q", cityExpected, addr.DisplayName)
		}
	})
	t.Run("reverse cached geocoding succeeds", func(t *testing.T) {
		rtFn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
			data, err := os.Open(cityFile)
			if err != nil {
				t.Fatalf("failed to open JSON response file: %s", err)
			}

			return &stdhttp.Response{
				StatusCode: 200,
				Body:       data,
				Header:     make(stdhttp.Header),
			}, nil
		}

		coder := geocode.NewCachedGeocoder(testCoderWithRoundtripFunc(t, rtFn), testHitTTL, testMissTTL)
		addr, err := coder.Reverse(t.Context(), cityCoords)
		if err != nil {
			t.Fatal(err)
		}
		if !addr.AddressFound {
			t.Fatal("expected address to be found")
		}
		if !strings.EqualFold(addr.DisplayName, cityExpected) {
			t.Errorf("expected address to be %q, got %q", cityExpected, addr.DisplayName)
		}
		addr, err = coder.Reverse(t.Context(), cityCoords)
		if err != nil {
			t.Fatal(err)
		}
		if !addr.CacheHit {
			t.Error("expected cache hit")
		}
	})
	t.Run("reverse geocoding with town set should return the correct city", func(t *testing.T) {
		rtFn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
			data, err := os.Open(townFile)
			if err != nil {
				t.Fatalf("failed to open JSON response file: %s", err)
			}

			return &stdhttp.Response{
				StatusCode: 200,
				Body:       data,
				Header:     make(stdhttp.Header),
			}, nil
		}

		coder := testCoderWithRoundtripFunc(t, rtFn)
		addr, err := coder.Reverse(t.Context(), townCoords)
		if err != nil {
			t.Fatal(err)
		}
		if !addr.AddressFound {
			t.Fatal("expected address to be found")
		}
		if !strings.EqualFold(addr.City, townExpected) {
			t.Errorf("expected city to be %q, got %q", townExpected, addr.DisplayName)
		}
	})
	t.Run("reverse geocoding with village set should return the correct city", func(t *testing.T) {
		rtFn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
			data, err := os.Open(villageFile)
			if err != nil {
				t.Fatalf("failed to open JSON response file: %s", err)
			}

			return &stdhttp.Response{
				StatusCode: 200,
				Body:       data,
				Header:     make(stdhttp.Header),
			}, nil
		}

		coder := testCoderWithRoundtripFunc(t, rtFn)
		addr, err := coder.Reverse(t.Context(), villageCoords)
		if err != nil {
			t.Fatal(err)
		}
		if !addr.AddressFound {
			t.Fatal("expected address to be found")
		}
		if !strings.EqualFold(addr.City, villageExpected) {
			t.Errorf("expected city to be %q, got %q", villageExpected, addr.DisplayName)
		}
	})
	t.Run("reverse geocoding fails", func(t *testing.T) {
		rtFn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
			return nil, errors.New("intentionally failing")
		}

		coder := testCoderWithRoundtripFunc(t, rtFn)
		_, err := coder.Reverse(t.Context(), cityCoords)
		if err == nil {
			t.Fatal("expected API request to fail")
		}
	})
	t.Run("reverse geocoding fails on NaN latitude response", func(t *testing.T) {
		rtFn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
			data, err := os.Open(cityFileBrokenLat)
			if err != nil {
				t.Fatalf("failed to open JSON response file: %s", err)
			}

			return &stdhttp.Response{
				StatusCode: 200,
				Body:       data,
				Header:     make(stdhttp.Header),
			}, nil
		}

		coder := testCoderWithRoundtripFunc(t, rtFn)
		_, err := coder.Reverse(t.Context(), villageCoords)
		if err == nil {
			t.Fatal("expected API request to fail")
		}
		if !strings.Contains(err.Error(), "failed to parse latitude") {
			t.Errorf("expected error to contain 'failed to parse latitude', got %s", err)
		}
	})
	t.Run("reverse geocoding fails on NaN longitude response", func(t *testing.T) {
		rtFn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
			data, err := os.Open(cityFileBrokenLon)
			if err != nil {
				t.Fatalf("failed to open JSON response file: %s", err)
			}

			return &stdhttp.Response{
				StatusCode: 200,
				Body:       data,
				Header:     make(stdhttp.Header),
			}, nil
		}

		coder := testCoderWithRoundtripFunc(t, rtFn)
		_, err := coder.Reverse(t.Context(), villageCoords)
		if err == nil {
			t.Fatal("expected API request to fail")
		}
		if !strings.Contains(err.Error(), "failed to parse longitude") {
			t.Errorf("expected error to contain 'failed to parse longitude', got %s", err)
		}
	})
}

func TestNominatim_Reverse_integration(t *testing.T) {
	testhelper.PerformIntegrationTests(t)
	t.Run("reverse geocoding succeeds", func(t *testing.T) {
		coder := testCoder(t)
		addr, err := coder.Reverse(t.Context(), cityCoords)
		if err != nil {
			t.Fatal(err)
		}
		if !addr.AddressFound {
			t.Fatal("expected address to be found")
		}
		if !strings.EqualFold(addr.DisplayName, cityExpected) {
			t.Errorf("expected address to be %q, got %q", cityExpected, addr.DisplayName)
		}
	})
}

func testCoder(_ *testing.T) geocode.Geocoder {
	testHttpClient := http.New(logger.New(slog.LevelDebug))
	testLang := language.English
	return New(testHttpClient, testLang)
}

func testCoderWithRoundtripFunc(_ *testing.T, fn func(req *stdhttp.Request) (*stdhttp.Response, error)) geocode.Geocoder {
	testHttpClient := http.New(logger.New(slog.LevelDebug))
	testHttpClient.Transport = testhelper.MockRoundTripper{Fn: fn}
	testLang := language.English
	return New(testHttpClient, testLang)
}
