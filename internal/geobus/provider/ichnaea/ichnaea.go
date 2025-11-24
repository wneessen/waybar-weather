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
	"sync"
	"time"

	"github.com/wneessen/waybar-weather/internal/geobus"
	"github.com/wneessen/waybar-weather/internal/http"

	"github.com/mdlayher/wifi"
)

const (
	apiEndpoint   = "https://api.beacondb.net/v1/geolocate"
	lookupTimeout = time.Second * 5
	wifiScanTime  = time.Minute * 2
	name          = "ichnaea"
)

type GeolocationICHNAEAProvider struct {
	name     string
	http     *http.Client
	wlan     *wifi.Client
	period   time.Duration
	ttl      time.Duration
	locateFn func(ctx context.Context) (lat, lon, acc float64, err error)

	apLock sync.RWMutex
	aps    []WirelessNetwork
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
	if http == nil {
		return nil, fmt.Errorf("http client is required")
	}
	wlan, err := wifi.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create wifi client: %w", err)
	}

	provider := &GeolocationICHNAEAProvider{
		name:   name,
		http:   http,
		wlan:   wlan,
		period: time.Minute * 5,
		ttl:    time.Hour * 1,
	}
	provider.locateFn = provider.locate
	return provider, nil
}

func (p *GeolocationICHNAEAProvider) Name() string {
	return p.name
}

// LookupStream continuously streams geolocation results from a file, emitting updates when data changes
// or context ends.
func (p *GeolocationICHNAEAProvider) LookupStream(ctx context.Context, key string) <-chan geobus.Result {
	out := make(chan geobus.Result)
	go p.monitorWifiAccessPoints(ctx)
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

			lat, lon, acc, err := p.locateFn(ctx)
			if err != nil {
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

func (p *GeolocationICHNAEAProvider) monitorWifiAccessPoints(ctx context.Context) {
	firstRun := true
	for {
		if !firstRun {
			select {
			case <-ctx.Done():
				return
			case <-time.After(wifiScanTime):
			}
		}
		firstRun = false

		list, err := p.wifiAccessPoints()
		if err != nil {
			continue
		}
		p.apLock.Lock()
		p.aps = list
		p.apLock.Unlock()
	}
}

func (p *GeolocationICHNAEAProvider) wifiAccessPoints() ([]WirelessNetwork, error) {
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
	p.apLock.RLock()
	wifiList := p.aps
	p.apLock.RUnlock()

	type request struct {
		ConsiderIP   bool              `json:"considerIp"`
		Accesspoints []WirelessNetwork `json:"wifiAccessPoints,omitempty"`
	}
	req := request{
		ConsiderIP:   true,
		Accesspoints: wifiList,
	}
	bodyBuffer := bytes.NewBuffer(nil)
	if err = json.NewEncoder(bodyBuffer).Encode(req); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to encode wifi list to JSON: %w", err)
	}

	ctxHttp, cancelHttp := context.WithTimeout(ctx, lookupTimeout)
	defer cancelHttp()
	result := new(APIResult)
	if _, err = p.http.Post(ctxHttp, apiEndpoint, result, bodyBuffer,
		map[string]string{"Content-Type": "application/json"}); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get geolocation data from API: %w", err)
	}

	return geobus.Truncate(result.Location.Latitude, geobus.TruncPrecision),
		geobus.Truncate(result.Location.Longitude, geobus.TruncPrecision),
		geobus.Truncate(result.Accuracy, geobus.TruncPrecision), nil
}
