// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package geocode

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

const (
	testHitTTL  = 200 * time.Millisecond
	testMissTTL = 200 * time.Millisecond
	testLat     = 52.5129
	testLon     = 13.3910
)

var testAddress = Address{
	Altitude:     35,
	DisplayName:  "Quartier 205, Friedrichstraße 67, 10117 Berlin, Germany",
	Country:      "Germany",
	State:        "Berlin",
	Municipality: "Berlin",
	CityDistrict: "Mitte",
	Postcode:     "10117",
	City:         "Berlin",
	Street:       "Friedrichstraße",
	HouseNumber:  "67",
}

type (
	mockCache struct{}
)

func (c *mockCache) Name() string { return "mock" }

func (c *mockCache) Reverse(_ context.Context, lat, lon float64) (Address, error) {
	addr := testAddress
	addr.Latitude = lat
	addr.Longitude = lon
	if lat == testLat && lon == testLon {
		addr.AddressFound = true
	}
	if lat == 1 && lon == -1 {
		return addr, errors.New("lookup intentionally failed")
	}
	return addr, nil
}

func TestNewCachedGeocoder(t *testing.T) {
	t.Run("a new geocoder should be returned", func(t *testing.T) {
		coder := NewCachedGeocoder(&mockCache{}, testHitTTL, testMissTTL)
		if coder == nil {
			t.Fatal("expected a non-nil geocoder")
		}
		if coder.Name() != "geocoder cache using mock" {
			t.Errorf("expected geocoder name to be 'geocode cacher using mock', got %q", coder.Name())
		}
	})
}

func TestCachedGeocoder_Reverse(t *testing.T) {
	coder := NewCachedGeocoder(&mockCache{}, testHitTTL, testMissTTL)
	t.Run("a cached address should be returned", func(t *testing.T) {
		addr, err := coder.Reverse(t.Context(), testLat, testLon)
		if err != nil {
			t.Fatal(err)
		}
		if !addr.AddressFound {
			t.Fatal("expected address to be found")
		}
		if addr.CacheHit {
			t.Fatal("expected cache miss")
		}
		if !strings.EqualFold(addr.DisplayName, testAddress.DisplayName) {
			t.Errorf("expected address to be %q, got %q", testAddress.DisplayName, addr.DisplayName)
		}
		if addr.Latitude != testLat {
			t.Errorf("expected latitude to be %f, got %f", testLat, addr.Latitude)
		}
		if addr.Longitude != testLon {
			t.Errorf("expected longitude to be %f, got %f", testLon, addr.Longitude)
		}
	})
	t.Run("fetching results twice should hit the cache", func(t *testing.T) {
		addr, err := coder.Reverse(t.Context(), testLat, testLon)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.EqualFold(addr.DisplayName, testAddress.DisplayName) {
			t.Errorf("expected address to be %q, got %q", testAddress.DisplayName, addr.DisplayName)
		}
		addr, err = coder.Reverse(t.Context(), testLat, testLon)
		if err != nil {
			t.Fatal(err)
		}
		if !addr.CacheHit {
			t.Error("expected cached result")
		}
		if !strings.EqualFold(addr.DisplayName, testAddress.DisplayName) {
			t.Errorf("expected address to be %q, got %q", testAddress.DisplayName, addr.DisplayName)
		}
	})
	t.Run("fetching a very close address should still hit the cache", func(t *testing.T) {
		addr, err := coder.Reverse(t.Context(), testLat, testLon)
		if err != nil {
			t.Fatal(err)
		}
		addr, err = coder.Reverse(t.Context(), testLat+0.002, testLon-0.002)
		if err != nil {
			t.Fatal(err)
		}
		if !addr.CacheHit {
			t.Error("expected cached result")
		}
		if !strings.EqualFold(addr.DisplayName, testAddress.DisplayName) {
			t.Errorf("expected address to be %q, got %q", testAddress.DisplayName, addr.DisplayName)
		}
	})
	t.Run("fetching a very close address but negative coordinates should still hit the cache", func(t *testing.T) {
		addr, err := coder.Reverse(t.Context(), testLat, testLon)
		if err != nil {
			t.Fatal(err)
		}
		addr, err = coder.Reverse(t.Context(), testLat-0.004, testLon+0.003)
		if err != nil {
			t.Fatal(err)
		}
		if !addr.CacheHit {
			t.Error("expected cached result")
		}
		if !strings.EqualFold(addr.DisplayName, testAddress.DisplayName) {
			t.Errorf("expected address to be %q, got %q", testAddress.DisplayName, addr.DisplayName)
		}
	})
	t.Run("fetching an unknow address causes a cache miss", func(t *testing.T) {
		addr, err := coder.Reverse(t.Context(), 2, -2)
		if err != nil {
			t.Fatal(err)
		}
		if addr.AddressFound {
			t.Fatal("expected address to be not found")
		}
		if addr.CacheHit {
			t.Error("expected cache miss")
		}
	})
	t.Run("fetching failes during lookup should return an error", func(t *testing.T) {
		_, err := coder.Reverse(t.Context(), 1, -1)
		if err == nil {
			t.Fatal("expected an error")
		}
	})
	t.Run("cache should not trigger on expired TTL", func(t *testing.T) {
		addr, err := coder.Reverse(t.Context(), testLat, testLon)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.EqualFold(addr.DisplayName, testAddress.DisplayName) {
			t.Errorf("expected address to be %q, got %q", testAddress.DisplayName, addr.DisplayName)
		}
		time.Sleep(testHitTTL * 2)
		addr, err = coder.Reverse(t.Context(), testLat, testLon)
		if err != nil {
			t.Fatal(err)
		}
		if addr.CacheHit {
			t.Error("expected cache miss")
		}
		if !strings.EqualFold(addr.DisplayName, testAddress.DisplayName) {
			t.Errorf("expected address to be %q, got %q", testAddress.DisplayName, addr.DisplayName)
		}
	})
	t.Run("cache should hit on non-expired TTL", func(t *testing.T) {
		addr, err := coder.Reverse(t.Context(), testLat, testLon)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.EqualFold(addr.DisplayName, testAddress.DisplayName) {
			t.Errorf("expected address to be %q, got %q", testAddress.DisplayName, addr.DisplayName)
		}
		time.Sleep(testHitTTL - 5*time.Millisecond)
		addr, err = coder.Reverse(t.Context(), testLat, testLon)
		if err != nil {
			t.Fatal(err)
		}
		if !addr.CacheHit {
			t.Error("expected cache hit")
		}
		if !strings.EqualFold(addr.DisplayName, testAddress.DisplayName) {
			t.Errorf("expected address to be %q, got %q", testAddress.DisplayName, addr.DisplayName)
		}
	})
}
