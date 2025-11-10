// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package geobus

import (
	"context"
	"sync"
)

// Orchestrator coordinates the tracking and publication of geolocation results from multiple
// providers through a GeoBus.
type Orchestrator struct {
	Bus       *GeoBus
	Providers []Provider
}

// Track initiates concurrent geolocation tracking for a given key across multiple providers in the Orchestrator.
func (o *Orchestrator) Track(ctx context.Context, key string) {
	var wg sync.WaitGroup
	for _, p := range o.Providers {
		wg.Add(1)
		go func(p Provider) {
			defer wg.Done()
			o.trackProvider(ctx, p, key)
		}(p)
	}
	<-ctx.Done()
	wg.Wait()
}

// trackProvider continuously tracks a Provider for geolocation data, publishing results to
// the GeoBus and implementing backoff.
func (o *Orchestrator) trackProvider(ctx context.Context, p Provider, key string) {
	backoff := initialBackoff
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		lookupChan := o.safeLookup(ctx, p, key)
		if lookupChan == nil {
			if !sleepOrDone(ctx, backoff) {
				return
			}
			backoff = nextBackoff(backoff)
			continue
		}

		for {
			select {
			case <-ctx.Done():
				return
			case r, ok := <-lookupChan:
				if !ok {
					if !sleepOrDone(ctx, backoff) {
						return
					}
					backoff = nextBackoff(backoff)
					break
				}
				o.Bus.Publish(r)
				backoff = initialBackoff
			}
		}
	}
}

// safeLookup safely invokes the LookupStream method on a Provider and recovers from potential panics.
// Returns a read-only channel of Result or nil if the operation fails.
func (o *Orchestrator) safeLookup(ctx context.Context, provider Provider, key string) (ch <-chan Result) {
	defer func() { _ = recover() }()
	return provider.LookupStream(ctx, key)
}
