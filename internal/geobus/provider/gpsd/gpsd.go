// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package gpsd

import (
	"context"
	"time"

	"github.com/wneessen/waybar-weather/internal/geobus"
	"github.com/wneessen/waybar-weather/internal/gpspoll"
)

const (
	host = "localhost"
	port = "2947"
	name = "gpsd"
)

type GeolocationGPSDProvider struct {
	name     string
	period   time.Duration
	ttl      time.Duration
	client   *gpspoll.Client
	locateFn func(ctx context.Context) (gpspoll.Fix, error)
}

func NewGeolocationGPSDProvider() *GeolocationGPSDProvider {
	provider := &GeolocationGPSDProvider{
		name:   name,
		period: time.Second * 3,
		ttl:    time.Minute * 2,
		client: gpspoll.NewClient(host, port),
	}
	provider.locateFn = provider.client.Poll

	return provider
}

func (p *GeolocationGPSDProvider) Name() string {
	return p.name
}

func (p *GeolocationGPSDProvider) LookupStream(ctx context.Context, key string) <-chan geobus.Result {
	out := make(chan geobus.Result)

	go func() {
		defer close(out)
		state := geobus.GeolocationState{}
		firstRun := true

		for {
			if !firstRun {
				select {
				case <-ctx.Done():
					return
				case <-time.After(p.period):
				}
			}
			firstRun = false

			fix, err := p.locateFn(ctx)
			if err != nil {
				continue
			}
			if !fix.Has2DFix() {
				continue
			}
			coord := geobus.Coordinate{Lat: fix.Lat, Lon: fix.Lon, Alt: fix.Alt, Acc: fix.Acc}

			// Only emit if values changed or it's the first read
			if state.HasChanged(coord) {
				state.Update(coord)
				r := p.createResult(key, coord)

				select {
				case <-ctx.Done():
					return
				case out <- r:
				}
			}
		}
	}()
	return out
}

// createResult composes and returns a Result using provided geolocation data and metadata.
func (p *GeolocationGPSDProvider) createResult(key string, coord geobus.Coordinate) geobus.Result {
	return geobus.Result{
		Key:            key,
		Lat:            coord.Lat,
		Lon:            coord.Lon,
		AccuracyMeters: coord.Acc,
		Source:         p.name,
		At:             time.Now(),
		TTL:            p.ttl,
	}
}
