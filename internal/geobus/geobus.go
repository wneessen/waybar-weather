// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package geobus

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/wneessen/waybar-weather/internal/logger"
)

const (
	accuracyEpsilon = 1e-6
)

const (
	AccuracyCountry = 300000
	AccuracyRegion  = 100000
	AccuracyCity    = 15000
	AccuracyZip     = 3000
	AccuracyUnknown = 1000000
	TruncPrecision  = 4
)

// Provider defines an interface for geolocation service providers.
// It supports retrieving streamed results for a given key.
type Provider interface {
	Name() string
	LookupStream(ctx context.Context, key string) <-chan Result
}

// GeoBus coordinates the publishing and subscribing of geolocation
// results between providers and consumers.
type GeoBus struct {
	mu          sync.RWMutex
	best        map[string]Result
	subscribers map[string]map[chan Result]struct{}
	log         *logger.Logger
}

// Result represents a geolocation result with associated metadata.
type Result struct {
	Key            string
	Lat, Lon       float64
	Alt            float64
	AccuracyMeters float64
	Source         string
	At             time.Time
	TTL            time.Duration
}

// New initializes and returns a new instance of GeoBus to handle
// geolocation result coordination.
func New(log *logger.Logger) (*GeoBus, error) {
	if log == nil {
		return nil, fmt.Errorf("logger is required")
	}
	return &GeoBus{
		best:        make(map[string]Result),
		subscribers: make(map[string]map[chan Result]struct{}),
		log:         log,
	}, nil
}

// Subscribe adds a subscriber for updates associated with the given key and
// buffer size, returning a result channel and an unsubscribe function.
func (b *GeoBus) Subscribe(key string, size int) (<-chan Result, func()) {
	ch := make(chan Result, size)

	b.mu.Lock()
	if _, ok := b.subscribers[key]; !ok {
		b.subscribers[key] = make(map[chan Result]struct{})
	}
	b.subscribers[key][ch] = struct{}{}

	// Immediately send the current best if we have it and it’s not expired.
	if best, ok := b.best[key]; ok && !best.IsExpired() {
		ch <- best
	}
	b.mu.Unlock()

	unsub := func() {
		b.mu.Lock()
		if subs, ok := b.subscribers[key]; ok {
			delete(subs, ch)
			if len(subs) == 0 {
				delete(b.subscribers, key)
			}
		}
		b.mu.Unlock()
		close(ch)
	}

	b.log.Debug("subscribed to geobus updates", slog.String("key", key))
	return ch, unsub
}

// Publish updates the best result for a key and notifies subscribers
func (b *GeoBus) Publish(r Result) {
	// Ignore zero-accuracy results; they’re meaningless.
	if r.AccuracyMeters <= 0 {
		return
	}
	// Ensure At is set.
	if r.At.IsZero() {
		r.At = time.Now()
	}

	newCoord := Coordinate{
		Lat: r.Lat,
		Lon: r.Lon,
		Acc: r.AccuracyMeters,
	}

	b.mu.Lock()
	shouldUpdate := false

	prev, have := b.best[r.Key]
	prevCoord := Coordinate{
		Lat: prev.Lat,
		Lon: prev.Lon,
		Acc: prev.AccuracyMeters,
	}

	// If the result is not expired or better and the position has changed significantly, update it.
	if !have || prev.IsExpired() || r.BetterThan(prev) && newCoord.PosHasSignificantChange(prevCoord) {
		shouldUpdate = true
	}

	b.log.Debug("received publish request", slog.Float64("latitude", r.Lat),
		slog.Float64("longitude", r.Lon), slog.Float64("accuracy", r.AccuracyMeters),
		slog.String("source", r.Source), slog.Bool("will_update", shouldUpdate),
	)
	if !shouldUpdate {
		b.mu.Unlock()
		return
	}

	b.best[r.Key] = r
	subs := b.subscribers[r.Key]
	b.mu.Unlock()

	// Non-blocking broadcast; slow subscribers just drop updates.
	for ch := range subs {
		select {
		case ch <- r:
		default:
		}
	}
}

// BetterThan compares two Result objects to determine if the current instance
// is better than the provided one.
func (r Result) BetterThan(prev Result) bool {
	if prev.Key == "" {
		return true
	}

	// Reject out-of-order results.
	if r.At.Before(prev.At) {
		return false
	}

	// More accurate?
	if r.AccuracyMeters < prev.AccuracyMeters-accuracyEpsilon {
		return true
	}
	if prev.AccuracyMeters < r.AccuracyMeters-accuracyEpsilon {
		return false
	}

	// Same-ish accuracy; we treat them as "not better".
	return false
}

// IsExpired checks if the Result has exceeded its time-to-live (TTL)
// based on the current time and the timestamp.
func (r Result) IsExpired() bool {
	return r.TTL > 0 && time.Since(r.At) > r.TTL
}

// Truncate truncates a float to a fixed decimal precision.
func Truncate(x float64, precision int) float64 {
	p := math.Pow(10, float64(precision))
	return math.Trunc(x*p) / p
}

// TrackProviders starts one goroutine per provider that streams results into the bus.
// It returns immediately; goroutines exit when ctx is cancelled or the provider channel closes.
func TrackProviders(ctx context.Context, bus *GeoBus, key string, providers ...Provider) {
	for _, p := range providers {
		go func() {
			ch := p.LookupStream(ctx, key)
			for {
				select {
				case <-ctx.Done():
					return
				case r, ok := <-ch:
					if !ok {
						return
					}
					bus.Publish(r)
				}
			}
		}()
	}
}
