// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package opencage

import (
	"context"
	"log/slog"
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
	testLat     = 52.5129
	testLon     = 13.3910
	testHitTTL  = 1 * time.Second
	testMissTTL = 1 * time.Second

	villageExpected = "Marshfield"
	villageLat      = 51.46292
	villageLon      = -2.31850

	townExpected = "Otley"
	townLat      = 53.90712
	townLon      = -1.69404
)

var testAddress = geocode.Address{
	Altitude:     35,
	DisplayName:  "Quartier 205, Friedrichstrasse 67, 10117 Berlin, Germany",
	Country:      "Germany",
	State:        "Berlin",
	Municipality: "Berlin",
	CityDistrict: "Mitte",
	Postcode:     "10117",
	City:         "Berlin",
	Street:       "Friedrichstraße",
	HouseNumber:  "67",
}

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

func TestReverse(t *testing.T) {
	performOnlineTest(t)
	t.Run("reverse geocoding succeeds", func(t *testing.T) {
		coder := testCoder(t)
		addr, err := coder.Reverse(t.Context(), testLat, testLon)
		if err != nil {
			t.Fatal(err)
		}
		if !addr.AddressFound {
			t.Fatal("expected address to be found")
		}
		if !strings.EqualFold(addr.DisplayName, testAddress.DisplayName) {
			t.Errorf("expected address to be %q, got %q", testAddress.DisplayName, addr.DisplayName)
		}
	})
	t.Run("reverse geocoding times out", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
		defer cancel()
		coder := testCoder(t)
		_, err := coder.Reverse(ctx, testLat, testLon)
		if err == nil {
			t.Fatal("expected API request to time out")
		}
	})
	t.Run("looking up a town should set the city correctly", func(t *testing.T) {
		coder := testCoder(t)
		addr, err := coder.Reverse(t.Context(), townLat, townLon)
		if err != nil {
			t.Fatal(err)
		}
		if !addr.AddressFound {
			t.Fatal("expected address to be found")
		}
		if addr.City != townExpected {
			t.Errorf("expected city to be %q, got %q", townExpected, addr.City)
		}
	})
	t.Run("looking up a village should set the city correctly", func(t *testing.T) {
		coder := testCoder(t)
		addr, err := coder.Reverse(t.Context(), villageLat, villageLon)
		if err != nil {
			t.Fatal(err)
		}
		if !addr.AddressFound {
			t.Fatal("expected address to be found")
		}
		if addr.City != villageExpected {
			t.Errorf("expected city to be %q, got %q", villageExpected, addr.City)
		}
	})
	t.Run("cached geocoder with opencage", func(t *testing.T) {
		coder := geocode.NewCachedGeocoder(testCoder(t), testHitTTL, testMissTTL)
		if coder == nil {
			t.Fatal("expected a non-nil geocoder")
		}
		addr, err := coder.Reverse(t.Context(), testLat, testLon)
		if err != nil {
			t.Fatalf("cached reverse geocoding via osm-nominatim failed: %s", err)
		}
		if !addr.AddressFound {
			t.Error("expected address to be found")
		}
		if addr.City != testAddress.City {
			t.Errorf("expected city to be %q, got %q", testAddress.City, addr.City)
		}
		addr, err = coder.Reverse(t.Context(), testLat, testLon)
		if err != nil {
			t.Fatalf("cached reverse geocoding via osm-nominatim failed: %s", err)
		}
		if !addr.CacheHit {
			t.Error("expected cache hit from cached osm-nominatim")
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

func performOnlineTest(t *testing.T) {
	if val := os.Getenv("PERFORM_ONLINE_TEST"); !strings.EqualFold(val, "true") {
		t.Skip("skipping online test")
	}
}
