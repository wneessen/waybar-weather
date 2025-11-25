// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package geobus

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/wneessen/waybar-weather/internal/logger"
)

const (
	accuracyEpsilon = 1e-6
	initialBackoff  = time.Second
	maxBackoff      = 30 * time.Second
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

// GeoBus coordinates the publishing and subscribing of geolocation results between providers and consumers.
type GeoBus struct {
	mu          sync.RWMutex
	logger      *logger.Logger
	best        map[string]Result
	subscribers map[string]map[chan Result]struct{}
	globalSubs  map[chan Result]struct{}
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

// BetterThan compares two Result objects to determine if the current instance is better than the provided one.
// Returns true if the current Result is more accurate, more confident, or more recent than the other.
// Considers accuracy, confidence level, and timestamp for the comparison with small tolerances for precision.
func (r Result) BetterThan(prev Result) bool {
	if prev.Key == "" {
		return true
	}
	if r.At.Before(prev.At) {
		return false
	}
	if r.AccuracyMeters < prev.AccuracyMeters-accuracyEpsilon {
		return true
	}
	if prev.AccuracyMeters < r.AccuracyMeters-accuracyEpsilon {
		return false
	}
	return false
}

// IsExpired checks if the Result has exceeded its time-to-live (TTL) based on the current time and the timestamp.
func (r Result) IsExpired() bool {
	return r.TTL > 0 && time.Since(r.At) > r.TTL
}

// New initializes and returns a new instance of GeoBus to handle geolocation result coordination.
func New(logger *logger.Logger) *GeoBus {
	return &GeoBus{
		logger:      logger,
		best:        make(map[string]Result),
		subscribers: make(map[string]map[chan Result]struct{}),
		globalSubs:  make(map[chan Result]struct{}),
	}
}

func (b *GeoBus) NewOrchestrator(provider []Provider) *Orchestrator {
	return &Orchestrator{
		Bus:       b,
		Providers: provider,
	}
}

// Subscribe adds a subscriber for updates associated with the given key and buffer size, returning a result
// channel and an unsubscribe function.
func (b *GeoBus) Subscribe(key string, size int) (<-chan Result, func()) {
	resultChan := make(chan Result, size)
	b.mu.Lock()
	if _, ok := b.subscribers[key]; !ok {
		b.subscribers[key] = make(map[chan Result]struct{})
	}

	b.subscribers[key][resultChan] = struct{}{}
	if best, ok := b.best[key]; ok && !best.IsExpired() {
		resultChan <- best
	}
	b.mu.Unlock()

	unsub := func() {
		b.mu.Lock()
		if subs, ok := b.subscribers[key]; ok {
			delete(subs, resultChan)
			if len(subs) == 0 {
				delete(b.subscribers, key)
			}
		}
		b.mu.Unlock()
		close(resultChan)
	}

	return resultChan, unsub
}

func (b *GeoBus) SubscribeAll(buffer int) (<-chan Result, func()) {
	ch := make(chan Result, buffer)
	b.mu.Lock()
	b.globalSubs[ch] = struct{}{}
	for _, v := range b.best {
		if !v.IsExpired() {
			ch <- v
		}
	}
	b.mu.Unlock()
	unsub := func() {
		b.mu.Lock()
		delete(b.globalSubs, ch)
		b.mu.Unlock()
		close(ch)
	}
	return ch, unsub
}

func (b *GeoBus) Publish(r Result) {
	if r.AccuracyMeters == 0 {
		return
	}
	if r.At.IsZero() {
		r.At = time.Now()
	}

	b.mu.Lock()
	prev, have := b.best[r.Key]
	prevCoord := Coordinate{Lat: prev.Lat, Lon: prev.Lon, Acc: prev.AccuracyMeters}
	newCoord := Coordinate{Lat: r.Lat, Lon: r.Lon, Acc: r.AccuracyMeters}

	// Update/broadcast the result if it's better than the previous one, expired or if the coordinate has
	// changed significantly
	if !have || prev.IsExpired() || r.BetterThan(prev) && newCoord.PosHasSignificantChange(prevCoord) {
		b.best[r.Key] = r
		b.broadcastResult(r)
	}

	// Update TTL if the source has not changed
	if have && prev.Source == r.Source {
		updated := b.best[r.Key]
		updated.At = r.At
		b.best[r.Key] = updated
	}
	b.mu.Unlock()
}

func (b *GeoBus) broadcastResult(r Result) {
	if subs, ok := b.subscribers[r.Key]; ok {
		for ch := range subs {
			select {
			case ch <- r:
			default:
			}
		}
	}
	for ch := range b.globalSubs {
		select {
		case ch <- r:
		default:
		}
	}
}

func (b *GeoBus) Best(key string) (Result, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	r, ok := b.best[key]
	return r, ok && !r.IsExpired()
}

func sleepOrDone(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func nextBackoff(d time.Duration) time.Duration {
	if d *= 2; d > maxBackoff {
		return maxBackoff
	}
	return d
}

func Truncate(x float64, precision int) float64 {
	p := math.Pow(10, float64(precision))
	return math.Trunc(x*p) / p
}
