// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package geocodeearth

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
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
	cityExpected    = "A.T. Kearney, Berlin, Germany"
	forwardAddress  = "Quartier 205, Friedrichstrasse 67, 10117 Berlin, Germany"
	cityFile        = "../../../../testdata/geocodeearth_berlin.json"
	cityForwardFile = "../../../../testdata/geocodeearth_berlin_forward.json"
	emptyArray      = "../../../../testdata/empty_array.json"
	testHitTTL      = 1 * time.Second
	testMissTTL     = 1 * time.Second

	villageExpected = "Marshfield"
	villageFile     = "../../../../testdata/geocodeearth_marshfield.json"

	townExpected = "Otley"
	townFile     = "../../../../testdata/geocodeearth_otley.json"
)

var (
	cityCoords        = geobus.Coordinate{Lat: 52.5129, Lon: 13.3910}
	cityForwardCoords = geobus.Coordinate{Lat: 52.512274, Lon: 13.390617}
	villageCoords     = geobus.Coordinate{Lat: 51.46292, Lon: -2.31850}
	townCoords        = geobus.Coordinate{Lat: 53.90712, Lon: -1.69404}
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

func TestGeocodeEarth_Reverse(t *testing.T) {
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
	t.Run("API responding with more than one result should fail", func(t *testing.T) {
		response := ReverseResponse{Features: []ReverseFeature{}}
		rtFn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
			buf := bytes.NewBuffer(nil)
			if err := json.NewEncoder(buf).Encode(response); err != nil {
				return nil, err
			}
			return &stdhttp.Response{
				StatusCode: 200,
				Body:       io.NopCloser(buf),
				Header:     make(stdhttp.Header),
			}, nil
		}
		coder := testCoderWithRoundtripFunc(t, rtFn)
		if coder == nil {
			t.Fatal("expected a non-nil geocoder")
		}
		_, err := coder.Reverse(t.Context(), cityCoords)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		wantErr := "no address found for coordinates"
		if !strings.EqualFold(err.Error(), wantErr) {
			t.Errorf("expected error to be %q, got %q", wantErr, err)
		}
	})
	t.Run("API responding with a non-200 reponse", func(t *testing.T) {
		response := ReverseResponse{Features: []ReverseFeature{{Properties: Properties{City: "Berlin"}}}}
		rtFn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
			buf := bytes.NewBuffer(nil)
			if err := json.NewEncoder(buf).Encode(response); err != nil {
				return nil, err
			}
			return &stdhttp.Response{
				StatusCode: 401,
				Body:       io.NopCloser(buf),
				Header:     make(stdhttp.Header),
			}, nil
		}
		coder := testCoderWithRoundtripFunc(t, rtFn)
		if coder == nil {
			t.Fatal("expected a non-nil geocoder")
		}
		_, err := coder.Reverse(t.Context(), cityCoords)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		wantErr := "received non-positive response code from geocode.earth API: 401"
		if !strings.EqualFold(err.Error(), wantErr) {
			t.Errorf("expected error to be %q, got %q", wantErr, err)
		}
	})
}

func TestGeocodeEarth_Search(t *testing.T) {
	t.Run("forward geocoding succeeds", func(t *testing.T) {
		rtFn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
			data, err := os.Open(cityForwardFile)
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
		coords, err := coder.Search(t.Context(), cityExpected)
		if err != nil {
			t.Fatal(err)
		}
		if !coords.Found {
			t.Fatal("expected address to be found")
		}
		if coords.Lat != cityForwardCoords.Lat {
			t.Errorf("expected latitude to be %f, got %f", cityForwardCoords.Lat, coords.Lat)
		}
		if coords.Lon != cityForwardCoords.Lon {
			t.Errorf("expected longitude to be %f, got %f", cityForwardCoords.Lon, coords.Lon)
		}
	})
	t.Run("forward geocoding fails", func(t *testing.T) {
		rtFn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
			return nil, errors.New("intentionally failing")
		}

		coder := testCoderWithRoundtripFunc(t, rtFn)
		_, err := coder.Search(t.Context(), cityExpected)
		if err == nil {
			t.Fatal("expected API request to fail")
		}
	})
	t.Run("API responding with a non-200 reponse", func(t *testing.T) {
		response := SearchResponse{Features: []SearchFeature{{Geometry: Geometry{
			Coordinates: []float64{cityForwardCoords.Lat, cityForwardCoords.Lon},
		}}}}
		rtFn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
			buf := bytes.NewBuffer(nil)
			if err := json.NewEncoder(buf).Encode(response); err != nil {
				return nil, err
			}
			return &stdhttp.Response{
				StatusCode: 401,
				Body:       io.NopCloser(buf),
				Header:     make(stdhttp.Header),
			}, nil
		}
		coder := testCoderWithRoundtripFunc(t, rtFn)
		if coder == nil {
			t.Fatal("expected a non-nil geocoder")
		}
		_, err := coder.Search(t.Context(), cityExpected)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		wantErr := "received non-positive response code from geocode.earth API: 401"
		if !strings.EqualFold(err.Error(), wantErr) {
			t.Errorf("expected error to be %q, got %q", wantErr, err)
		}
	})
	t.Run("forward geocoding returning empty array fails", func(t *testing.T) {
		rtFn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
			data, err := os.Open(emptyArray)
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
		_, err := coder.Search(t.Context(), cityExpected)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
	t.Run("API responding with a only one coordinate in JSON", func(t *testing.T) {
		response := SearchResponse{Features: []SearchFeature{{Geometry: Geometry{
			Coordinates: []float64{cityForwardCoords.Lon},
		}}}}
		rtFn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
			buf := bytes.NewBuffer(nil)
			if err := json.NewEncoder(buf).Encode(response); err != nil {
				return nil, err
			}
			return &stdhttp.Response{
				StatusCode: 200,
				Body:       io.NopCloser(buf),
				Header:     make(stdhttp.Header),
			}, nil
		}
		coder := testCoderWithRoundtripFunc(t, rtFn)
		if coder == nil {
			t.Fatal("expected a non-nil geocoder")
		}
		_, err := coder.Search(t.Context(), cityExpected)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		wantErr := "unexpected 2 coordinates in response"
		if !strings.Contains(err.Error(), wantErr) {
			t.Errorf("expected error to be %q, got %q", wantErr, err)
		}
	})
	t.Run("API responding with no 'Feature' in JSON", func(t *testing.T) {
		response := SearchResponse{Type: "Invalid"}
		rtFn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
			buf := bytes.NewBuffer(nil)
			if err := json.NewEncoder(buf).Encode(response); err != nil {
				return nil, err
			}
			return &stdhttp.Response{
				StatusCode: 200,
				Body:       io.NopCloser(buf),
				Header:     make(stdhttp.Header),
			}, nil
		}
		coder := testCoderWithRoundtripFunc(t, rtFn)
		if coder == nil {
			t.Fatal("expected a non-nil geocoder")
		}
		_, err := coder.Search(t.Context(), cityExpected)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		wantErr := "no coordinates found for address"
		if !strings.Contains(err.Error(), wantErr) {
			t.Errorf("expected error to be %q, got %q", wantErr, err)
		}
	})
}

func TestGeocodeEarth_integration(t *testing.T) {
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
	t.Run("forward geocoding succeeds", func(t *testing.T) {
		coder := testCoder(t)
		coords, err := coder.Search(t.Context(), forwardAddress)
		if err != nil {
			t.Fatal(err)
		}
		if !coords.Found {
			t.Fatal("expected coordinates to be found")
		}
		if coords.Lat != cityForwardCoords.Lat {
			t.Errorf("expected latitude to be %f, got %f", cityForwardCoords.Lat, coords.Lat)
		}
		if coords.Lon != cityForwardCoords.Lon {
			t.Errorf("expected longitude to be %f, got %f", cityForwardCoords.Lon, coords.Lon)
		}
	})
}

func testCoder(t *testing.T) geocode.Geocoder {
	testHttpClient := http.New(logger.New(slog.LevelDebug))
	testLang := language.English
	apikey := os.Getenv("GEOCODEEARTH_APIKEY")
	if apikey == "" {
		t.Skip("no geocode.earth API key set, skipping tests")
	}
	return New(testHttpClient, testLang, apikey)
}

func testCoderWithRoundtripFunc(t *testing.T, fn func(req *stdhttp.Request) (*stdhttp.Response, error)) geocode.Geocoder {
	testHttpClient := http.New(logger.New(slog.LevelDebug))
	testHttpClient.Transport = testhelper.MockRoundTripper{Fn: fn}
	testLang := language.English
	apikey := os.Getenv("GEOCODEEARTH_APIKEY")
	if apikey == "" {
		t.Skip("no geocode.earth API key set, skipping tests")
	}
	return New(testHttpClient, testLang, apikey)
}
