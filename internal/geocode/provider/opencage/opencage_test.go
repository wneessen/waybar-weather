// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package opencage

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

	"github.com/wneessen/waybar-weather/internal/geocode"
	"github.com/wneessen/waybar-weather/internal/http"
	"github.com/wneessen/waybar-weather/internal/logger"
)

const (
	cityExpected = "Quartier 205, Friedrichstrasse 67, 10117 Berlin, Germany"
	cityFile     = "../../../../testdata/opencage_berlin.json"
	cityLat      = 52.5129
	cityLon      = 13.3910
	testHitTTL   = 1 * time.Second
	testMissTTL  = 1 * time.Second

	villageExpected = "Marshfield"
	villageFile     = "../../../../testdata/opencage_marshfield.json"
	villageLat      = 51.46292
	villageLon      = -2.31850

	townExpected = "Otley"
	townFile     = "../../../../testdata/opencage_otley.json"
	townLat      = 53.90712
	townLon      = -1.69404
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

func TestOpenCage_Reverse(t *testing.T) {
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
		addr, err := coder.Reverse(t.Context(), cityLat, cityLon)
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
		addr, err := coder.Reverse(t.Context(), cityLat, cityLon)
		if err != nil {
			t.Fatal(err)
		}
		if !addr.AddressFound {
			t.Fatal("expected address to be found")
		}
		if !strings.EqualFold(addr.DisplayName, cityExpected) {
			t.Errorf("expected address to be %q, got %q", cityExpected, addr.DisplayName)
		}
		addr, err = coder.Reverse(t.Context(), cityLat, cityLon)
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
		addr, err := coder.Reverse(t.Context(), townLat, townLon)
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
		addr, err := coder.Reverse(t.Context(), villageLat, villageLon)
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
		_, err := coder.Reverse(t.Context(), cityLat, cityLon)
		if err == nil {
			t.Fatal("expected API request to fail")
		}
	})
	t.Run("API responding with more than one result should fail", func(t *testing.T) {
		response := Response{TotalResults: 2}
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
		_, err := coder.Reverse(t.Context(), cityLat, cityLon)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		wantErr := "unambigous amount of results returned for coordinates"
		if !strings.Contains(err.Error(), wantErr) {
			t.Errorf("expected error to contain %q, got %q", wantErr, err)
		}
	})
}

func TestOpenCage_Reverse_integration(t *testing.T) {
	performIntegrationTests(t)
	t.Run("reverse geocoding succeeds", func(t *testing.T) {
		coder := testCoder(t)
		addr, err := coder.Reverse(t.Context(), cityLat, cityLon)
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

func testCoder(t *testing.T) geocode.Geocoder {
	testHttpClient := http.New(logger.NewLogger(slog.LevelDebug))
	testLang := language.English
	apikey := os.Getenv("OPENCAGE_APIKEY")
	if apikey == "" {
		t.Skip("no opencage API key set, skipping tests")
	}
	return New(testHttpClient, testLang, apikey)
}

func testCoderWithRoundtripFunc(t *testing.T, fn func(req *stdhttp.Request) (*stdhttp.Response, error)) geocode.Geocoder {
	testHttpClient := http.New(logger.NewLogger(slog.LevelDebug))
	testHttpClient.Transport = mockRoundTripper{fn: fn}
	testLang := language.English
	apikey := os.Getenv("OPENCAGE_APIKEY")
	if apikey == "" {
		t.Skip("no opencage API key set, skipping tests")
	}
	return New(testHttpClient, testLang, apikey)
}

func performIntegrationTests(t *testing.T) {
	if val := os.Getenv("PERFORM_INTEGRATION_TEST"); !strings.EqualFold(val, "true") {
		t.Skip("skipping integration test")
	}
}

type mockRoundTripper struct {
	fn func(req *stdhttp.Request) (*stdhttp.Response, error)
}

func (m mockRoundTripper) RoundTrip(req *stdhttp.Request) (*stdhttp.Response, error) {
	return m.fn(req)
}
