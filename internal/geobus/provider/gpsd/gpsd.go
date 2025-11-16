// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package gpsd

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/wneessen/waybar-weather/internal/geobus"

	"github.com/stratoberry/go-gpsd"
)

const (
	host = "localhost"
	port = "2947"
)

type GeolocationGPSDProvider struct {
	name   string
	period time.Duration
	ttl    time.Duration
}

func NewGeolocationGPSDProvider() *GeolocationGPSDProvider {
	return &GeolocationGPSDProvider{
		name:   "gpsd",
		period: time.Second * 30,
		ttl:    time.Minute * 2,
	}
}

func (p *GeolocationGPSDProvider) Name() string {
	return p.name
}

func (p *GeolocationGPSDProvider) LookupStream(ctx context.Context, key string) <-chan geobus.Result {
	out := make(chan geobus.Result)

	go func() {
		defer close(out)
		state := geobus.GeolocationState{}

		for {
			// Exit if the caller is done
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Connect to gpsd
			addr := net.JoinHostPort(host, port)
			session, err := gpsd.Dial(addr)
			if err != nil {
				// gpsd unavailable — log and retry after a delay
				// (Replace with your logger if you have one.)
				fmt.Printf("GeolocationGPSD: failed to connect to gpsd at %q: %s\n", addr, err)

				select {
				case <-ctx.Done():
					return
				case <-time.After(p.period):
					continue
				}
			}

			// Install TPV filter: this gets called for every TPV report
			session.AddFilter("TPV", func(r interface{}) {
				tpv, ok := r.(*gpsd.TPVReport)
				if !ok {
					return
				}

				// Need at least 2D fix
				if tpv.Mode < gpsd.Mode2D {
					return
				}

				lat, lon, alt, acc := geobus.Truncate(tpv.Lat, geobus.TruncPrecision),
					geobus.Truncate(tpv.Lon, geobus.TruncPrecision),
					geobus.Truncate(tpv.Alt, geobus.TruncPrecision),
					geobus.Truncate((tpv.Lat+tpv.Lon)/2, geobus.TruncPrecision)

				// Only emit if values changed or it's the first fix
				if !state.HasChanged(lat, lon, 0, acc) {
					return
				}
				state.Update(lat, lon, alt, acc)

				res := p.createResult(key, lat, lon, acc)

				select {
				case <-ctx.Done():
					// Caller is done; just stop sending.
					return
				case out <- res:
				}
			})

			// Start watching the stream. Watch() returns a channel that closes
			// when the watch ends (e.g. connection lost).
			done := session.Watch()

			select {
			case <-ctx.Done():
				// Context canceled; just return. The process exiting will
				// tear down the gpsd connection; go-gpsd itself has no Close().
				return
			case <-done:
				// gpsd connection ended; reconnect after a short delay
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(p.period):
			}
		}
	}()

	return out
}

// createResult composes and returns a Result using provided geolocation data and metadata.
func (p *GeolocationGPSDProvider) createResult(key string, lat, lon, acc float64) geobus.Result {
	return geobus.Result{
		Key:            key,
		Lat:            lat,
		Lon:            lon,
		AccuracyMeters: acc,
		Source:         p.name,
		At:             time.Now(),
		TTL:            p.ttl,
	}
}
