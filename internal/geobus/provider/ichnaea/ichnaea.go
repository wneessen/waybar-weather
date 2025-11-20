// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package ichnaea

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/wneessen/waybar-weather/internal/geobus"
	"github.com/wneessen/waybar-weather/internal/http"

	"github.com/mdlayher/wifi"
)

const (
	APIEndpoint   = "https://api.beacondb.net/v1/geolocate"
	LookupTimeout = time.Second * 5
)

type GeolocationICHNAEAProvider struct {
	name   string
	http   *http.Client
	wlan   *wifi.Client
	period time.Duration
	ttl    time.Duration
}

type APIResult struct {
	Location struct {
		Latitude  float64 `json:"lat"`
		Longitude float64 `json:"lng"`
	} `json:"location"`
	Accuracy float64 `json:"accuracy"`
}

type WirelessNetwork struct {
	LastSeen       int64  `json:"age"`
	MACAddress     string `json:"macAddress"`
	SignalStrength int32  `json:"signalStrength"`
}

func NewGeolocationICHNAEAProvider(http *http.Client) (*GeolocationICHNAEAProvider, error) {
	wlan, err := wifi.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create wifi client: %w", err)
	}
	return &GeolocationICHNAEAProvider{
		name:   "ichnaea",
		http:   http,
		wlan:   wlan,
		period: 5 * time.Minute,
		ttl:    10 * time.Minute,
	}, nil
}

func (p *GeolocationICHNAEAProvider) Name() string {
	return p.name
}

// LookupStream continuously streams geolocation results from a file, emitting updates when data changes
// or context ends.
func (p *GeolocationICHNAEAProvider) LookupStream(ctx context.Context, key string) <-chan geobus.Result {
	out := make(chan geobus.Result)
	go func() {
		defer close(out)
		state := geobus.GeolocationState{}

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			lat, lon, acc, err := p.locate(ctx)
			if err != nil {
				time.Sleep(p.period)
				continue
			}
			coord := geobus.Coordinate{Lat: lat, Lon: lon, Acc: acc}

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
func (p *GeolocationICHNAEAProvider) createResult(key string, coord geobus.Coordinate) geobus.Result {
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

func (p *GeolocationICHNAEAProvider) wifiList() ([]WirelessNetwork, error) {
	var checkIfaces []*wifi.Interface
	var list []WirelessNetwork

	ifaces, err := p.wlan.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to list interfaces: %w", err)
	}
	for _, iface := range ifaces {
		if iface.Type != wifi.InterfaceTypeStation {
			continue
		}
		checkIfaces = append(checkIfaces, iface)
	}
	if len(checkIfaces) == 0 {
		return nil, nil
	}

	for _, iface := range checkIfaces {
		aps, err := p.wlan.AccessPoints(iface)
		if err != nil {
			continue
		}
		for _, ap := range aps {
			if ap.SSID == "" || ap.SSID[0] == '\x00' || strings.HasSuffix(ap.SSID, "_nomap") {
				continue
			}
			list = append(list, WirelessNetwork{
				SignalStrength: ap.Signal / 100,
				MACAddress:     ap.BSSID.String(),
				LastSeen:       ap.LastSeen.Milliseconds(),
			})
		}
	}

	return list, nil
}

func (p *GeolocationICHNAEAProvider) locate(ctx context.Context) (lat, lon, acc float64, err error) {
	wifiList, err := p.wifiList()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to retrieve wifi list: %w", err)
	}
	if len(wifiList) == 0 {
		return 0, 0, 0, nil
	}

	type request struct {
		ConsiderIP   bool              `json:"considerIp"`
		Accesspoints []WirelessNetwork `json:"wifiAccessPoints"`
	}
	req := request{
		ConsiderIP:   true,
		Accesspoints: wifiList,
	}
	bodyBuffer := bytes.NewBuffer(nil)
	if err = json.NewEncoder(bodyBuffer).Encode(req); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to encode wifi list to JSON: %w", err)
	}

	ctxHttp, cancelHttp := context.WithTimeout(ctx, LookupTimeout)
	defer cancelHttp()
	result := new(APIResult)
	if _, err = p.http.Post(ctxHttp, APIEndpoint, result, bodyBuffer,
		map[string]string{"Content-Provider": "application/json"}); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get geolocation data from API: %w", err)
	}

	return geobus.Truncate(result.Location.Latitude, geobus.TruncPrecision),
		geobus.Truncate(result.Location.Longitude, geobus.TruncPrecision),
		geobus.Truncate(result.Accuracy, geobus.TruncPrecision), nil
}
