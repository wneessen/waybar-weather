// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package geocode

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/wneessen/waybar-weather/internal/geobus"
)

// coordPrecision is the precision used to quantize coordinates (0.01 degrees ≈ 1.1 km)
const coordPrecision = 1e-2

type reverseKey struct {
	Provider string
	LatQ     int32
	LonQ     int32
}

type reverseCacheEntry struct {
	Address Address
	Expiry  time.Time
}

type searchCacheEntry struct {
	Coords geobus.Coordinate
	Expiry time.Time
}

type CachedGeocoder struct {
	coder   Geocoder
	ttlHit  time.Duration
	ttlMiss time.Duration

	mu           sync.RWMutex
	reverseCache map[reverseKey]reverseCacheEntry
	searchCache  map[string]searchCacheEntry
}

func NewCachedGeocoder(coder Geocoder, ttlHit, ttlMiss time.Duration) *CachedGeocoder {
	return &CachedGeocoder{
		coder:        coder,
		ttlHit:       ttlHit,
		ttlMiss:      ttlMiss,
		reverseCache: make(map[reverseKey]reverseCacheEntry),
	}
}

func (c *CachedGeocoder) Name() string {
	return "geocoder cache using " + c.coder.Name()
}

func (c *CachedGeocoder) Reverse(ctx context.Context, coords geobus.Coordinate) (Address, error) {
	key := newKey(c.coder.Name(), coords.Lat, coords.Lon)

	c.mu.RLock()
	entry, ok := c.reverseCache[key]
	if ok && time.Now().Before(entry.Expiry) {
		addr := entry.Address
		c.mu.RUnlock()
		addr.CacheHit = true
		return addr, nil
	}
	c.mu.RUnlock()

	addr, err := c.coder.Reverse(ctx, coords)
	if err != nil {
		return addr, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	ttl := c.ttlHit
	if !addr.AddressFound {
		ttl = c.ttlMiss
	}
	c.reverseCache[key] = reverseCacheEntry{
		Address: addr,
		Expiry:  time.Now().Add(ttl),
	}

	return addr, nil
}

func (c *CachedGeocoder) Search(ctx context.Context, key string) (geobus.Coordinate, error) {
	c.mu.RLock()
	entry, ok := c.searchCache[key]
	if ok && time.Now().Before(entry.Expiry) {
		coords := entry.Coords
		c.mu.RUnlock()
		coords.CacheHit = true
		return coords, nil
	}
	c.mu.RUnlock()

	coords, err := c.coder.Search(ctx, key)
	if err != nil {
		return coords, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	ttl := c.ttlHit
	if !coords.Found {
		ttl = c.ttlMiss
	}
	c.searchCache[key] = searchCacheEntry{
		Coords: coords,
		Expiry: time.Now().Add(ttl),
	}

	return coords, nil
}

func quantizeCoord(val float64) int32 {
	return int32(math.Round(val / coordPrecision))
}

func newKey(provider string, lat, lon float64) reverseKey {
	return reverseKey{
		Provider: provider,
		LatQ:     quantizeCoord(lat),
		LonQ:     quantizeCoord(lon),
	}
}
