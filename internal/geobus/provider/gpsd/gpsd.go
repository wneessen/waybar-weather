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
	name = "gpsd"
)

type GeolocationGPSDProvider struct {
	name   string
	period time.Duration
	ttl    time.Duration
}

func NewGeolocationGPSDProvider() *GeolocationGPSDProvider {
	return &GeolocationGPSDProvider{
		name:   name,
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

			coords := p.fetchGPSData(ctx)
			select {
			case <-ctx.Done():
				return
			case coord := <-coords:
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

func (p *GeolocationGPSDProvider) fetchGPSData(ctx context.Context) <-chan geobus.Coordinate {
	coords := make(chan geobus.Coordinate)

	go func() {
		addr := net.JoinHostPort(host, port)
		session, err := gpsd.Dial(addr)
		if err != nil {
			close(coords)
			return
		}
		defer func() {
			_ = session.Close()
		}()

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

			coord := geobus.Coordinate{Lat: lat, Lon: lon, Alt: alt, Acc: acc}
			fmt.Printf("Received coord from GPSd: %+v\n", coord)

			select {
			case <-ctx.Done():
				return
			case coords <- coord:
			}
		})

		done := session.Watch()

		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		}
	}()

	return coords
}
