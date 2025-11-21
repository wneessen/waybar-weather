// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package geocode

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

// coordPrecision is the precision used to quantize coordinates (0.01 degrees ≈ 1.1 km)
const coordPrecision = 1e-2

type cacheKey struct {
	Provider string
	LatQ     int32
	LonQ     int32
}

type cacheEntry struct {
	Address Address
	Expiry  time.Time
}

type CachedGeocoder struct {
	coder   Geocoder
	ttlHit  time.Duration
	ttlMiss time.Duration

	mu    sync.RWMutex
	cache map[cacheKey]cacheEntry
}

func NewCachedGeocoder(coder Geocoder, ttlHit, ttlMiss time.Duration) *CachedGeocoder {
	return &CachedGeocoder{
		coder:   coder,
		ttlHit:  ttlHit,
		ttlMiss: ttlMiss,
		cache:   make(map[cacheKey]cacheEntry),
	}
}

func (c *CachedGeocoder) Name() string {
	return "geocoder cache using " + c.coder.Name()
}

func (c *CachedGeocoder) Reverse(ctx context.Context, lat, lon float64) (Address, error) {
	key := newKey(c.coder.Name(), lat, lon)

	c.mu.RLock()
	entry, ok := c.cache[key]
	if ok && time.Now().Before(entry.Expiry) {
		addr := entry.Address
		c.mu.RUnlock()
		fmt.Printf("cache hit for %+v\n", addr)
		return addr, nil
	}
	c.mu.RUnlock()

	addr, err := c.coder.Reverse(ctx, lat, lon)
	if err != nil {
		return addr, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	ttl := c.ttlHit
	if !addr.AddressFound {
		ttl = c.ttlMiss
	}
	c.cache[key] = cacheEntry{
		Address: addr,
		Expiry:  time.Now().Add(ttl),
	}

	return addr, nil
}

func quantizeCoord(v float64) int32 {
	return int32(math.Round(v / coordPrecision))
}

func newKey(provider string, lat, lon float64) cacheKey {
	return cacheKey{
		Provider: provider,
		LatQ:     quantizeCoord(lat),
		LonQ:     quantizeCoord(lon),
	}
}
