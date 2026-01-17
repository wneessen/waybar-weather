// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package ichnaea

import (
	"context"
	"errors"
	"io"
	"log/slog"
	stdhttp "net/http"
	"os"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/mdlayher/wifi"

	"github.com/wneessen/waybar-weather/internal/geobus"
	"github.com/wneessen/waybar-weather/internal/http"
	"github.com/wneessen/waybar-weather/internal/logger"
	"github.com/wneessen/waybar-weather/internal/testhelper"
)

const (
	testFile = "../../../../testdata/beacondb.json"
	testLat  = 40.7185
	testLon  = -74.0025
	testAcc  = 2000
)

func TestNewGeolocationICHNAEAProvider(t *testing.T) {
	testRequiresWiFi(t)
	t.Run("new ICHNAEA provider succeeds", func(t *testing.T) {
		provider, err := NewGeolocationICHNAEAProvider(http.New(logger.New(slog.LevelInfo)))
		if err != nil {
			t.Fatalf("failed to create ICHNAEA provider: %s", err)
		}
		if provider == nil {
			t.Fatal("expected provider to be non-nil")
		}
	})
	t.Run("ICHNAEA without http client fails ", func(t *testing.T) {
		provider, err := NewGeolocationICHNAEAProvider(nil)
		if err == nil {
			t.Fatal("expected provider to fail")
		}
		if provider != nil {
			t.Fatal("expected provider to be nil")
		}
	})
}

func TestGeolocationICHNAEAProvider_Name(t *testing.T) {
	testRequiresWiFi(t)
	provider, err := NewGeolocationICHNAEAProvider(http.New(logger.New(slog.LevelInfo)))
	if err != nil {
		t.Fatalf("failed to create ICHNAEA provider: %s", err)
	}
	if !strings.EqualFold(provider.Name(), name) {
		t.Errorf("expected provider name to be %s, got %s", name, provider.Name())
	}
}

// This test is very flacky, since it depends on the WiFi hardware
func TestNewGeolocationICHNAEAProvider_wifiList(t *testing.T) {
	testRequiresWiFi(t)
	provider, err := NewGeolocationICHNAEAProvider(http.New(logger.New(slog.LevelInfo)))
	if err != nil {
		t.Fatalf("failed to create ICHNAEA provider: %s", err)
	}
	list, err := provider.wifiAccessPoints(t.Context())
	if err != nil {
		t.Fatalf("failed to get WiFi list: %s", err)
	}
	if len(list) == 0 {
		t.Skip("no WiFi access points found, test results are meaningless")
	}
}

func TestGeolocationICHNAEAProvider_locate(t *testing.T) {
	testRequiresWiFi(t)
	t.Run("locate succeeds with different accuracies", func(t *testing.T) {
		rtFn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
			data, err := os.Open(testFile)
			if err != nil {
				t.Fatalf("failed to open JSON response file: %s", err)
			}

			return &stdhttp.Response{
				StatusCode: 200,
				Body:       data,
				Header:     make(stdhttp.Header),
			}, nil
		}
		client := http.New(logger.New(slog.LevelInfo))
		client.Transport = testhelper.MockRoundTripper{Fn: rtFn}
		provider, err := NewGeolocationICHNAEAProvider(client)
		if err != nil {
			t.Fatalf("failed to create ICHNAEA provider: %s", err)
		}

		lat, lon, acc, err := provider.locate(t.Context())
		if err != nil {
			t.Fatalf("failed to locate coordinates via ICHNAEA: %s", err)
		}
		if lat != testLat {
			t.Errorf("expected latitude to be %f, got %f", testLat, lat)
		}
		if lon != testLon {
			t.Errorf("expected longitude to be %f, got %f", testLon, lon)
		}
		if geobus.Truncate(acc, 1) != geobus.Truncate(testAcc, 1) {
			t.Errorf("expected accuracy to be %f, got %f", geobus.Truncate(testAcc, 1),
				geobus.Truncate(acc, 1))
		}
	})
	t.Run("locate fails with broken JSON", func(t *testing.T) {
		rtFn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
			return &stdhttp.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("NOT_JSON")),
				Header:     make(stdhttp.Header),
			}, nil
		}
		client := http.New(logger.New(slog.LevelInfo))
		client.Transport = testhelper.MockRoundTripper{Fn: rtFn}
		provider, err := NewGeolocationICHNAEAProvider(client)
		if err != nil {
			t.Fatalf("failed to create ICHNAEA provider: %s", err)
		}

		_, _, _, err = provider.locate(t.Context())
		if err == nil {
			t.Fatal("expected locate to fail")
		}
	})
}

func TestGeolocationICHNAEAProvider_LookupStream(t *testing.T) {
	testRequiresWiFi(t)
	t.Run("lookup stream succeeds", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			rtFn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
				data, err := os.Open(testFile)
				if err != nil {
					t.Fatalf("failed to open JSON response file: %s", err)
				}

				return &stdhttp.Response{
					StatusCode: 200,
					Body:       data,
					Header:     make(stdhttp.Header),
				}, nil
			}
			client := http.New(logger.New(slog.LevelInfo))
			client.Transport = testhelper.MockRoundTripper{Fn: rtFn}
			provider, err := NewGeolocationICHNAEAProvider(client)
			if err != nil {
				t.Fatalf("failed to create GeoIP provider: %s", err)
			}
			provider.ttl = time.Millisecond * 10
			provider.period = time.Millisecond * 10

			out := provider.LookupStream(ctx, "test")
			if out == nil {
				t.Fatal("expected stream to be non-nil")
			}

			var results []geobus.Result
			for len(results) < 1 {
				select {
				case r := <-out:
					results = append(results, r)
					cancel()
				default:
					// Block until all goroutines are durably blocked, then advance
					// fake time to the next wakeup (e.g. time.After/ Sleep).
					synctest.Wait()
				}
			}

			synctest.Wait()
			if len(results) != 1 {
				t.Fatalf("expected at least one result, got %d", len(results))
			}
			result := results[0]
			if result.Key != "test" {
				t.Errorf("expected key to be %s, got %s", "test", result.Key)
			}
			if result.Lat != testLat {
				t.Errorf("expected latitude to be %f, got %f", testLat, result.Lat)
			}
			if result.Lon != testLon {
				t.Errorf("expected longitude to be %f, got %f", testLon, result.Lon)
			}
			wantAcc := 2000.0
			if result.AccuracyMeters != wantAcc {
				t.Errorf("expected accuracy to be %f, got %f", wantAcc, result.AccuracyMeters)
			}
			if result.Source != provider.Name() {
				t.Errorf("expected source to be %s, got %s", provider.Name(), result.Source)
			}
		})
	})
	t.Run("lookup stream fails during lookup", func(t *testing.T) {
		runCount := 0
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			provider, err := NewGeolocationICHNAEAProvider(http.New(logger.New(slog.LevelInfo)))
			if err != nil {
				t.Fatalf("failed to create GeoIP provider: %s", err)
			}
			provider.period = time.Millisecond * 10
			provider.locateFn = func(ctx context.Context) (float64, float64, float64, error) {
				if runCount == 0 {
					runCount++
					return 0, 0, 0, errors.New("intentionally failing")
				}
				return 1.0, 2.0, 3.0, nil
			}

			out := provider.LookupStream(ctx, "test")
			if out == nil {
				t.Fatal("expected stream to be non-nil")
			}

			var result geobus.Result
			select {
			case r := <-out:
				result = r
				cancel()
			case <-ctx.Done():
				t.Fatalf("context done before result: %v", ctx.Err())
			}
			synctest.Wait()

			if result.Lat != 1.0 {
				t.Errorf("expected latitude to be %f, got %f", 1.0, result.Lat)
			}
			if result.Lon != 2.0 {
				t.Errorf("expected longitude to be %f, got %f", 2.0, result.Lon)
			}
			if result.AccuracyMeters != 3.0 {
				t.Errorf("expected accuracy to be %f, got %f", 3.0, result.AccuracyMeters)
			}
		})
	})
}

func TestGeolocationICHNAEAProvider_createResult(t *testing.T) {
	testRequiresWiFi(t)
	provider, err := NewGeolocationICHNAEAProvider(http.New(logger.New(slog.LevelInfo)))
	if err != nil {
		t.Fatalf("failed to create GeoIP provider: %s", err)
	}
	result := provider.createResult("test", geobus.Coordinate{Lat: testLat, Lon: testLon, Acc: geobus.AccuracyCity})
	if result.Lat != testLat {
		t.Errorf("expected latitude to be %f, got %f", testLat, result.Lat)
	}
	if result.Lon != testLon {
		t.Errorf("expected longitude to be %f, got %f", testLon, result.Lon)
	}
	if result.Key != "test" {
		t.Errorf("expected key to be %s, got %s", "test", result.Key)
	}
	if result.AccuracyMeters != geobus.AccuracyCity {
		t.Errorf("expected accuracy to be %d, got %f", geobus.AccuracyCity, result.AccuracyMeters)
	}
	if result.Source != provider.Name() {
		t.Errorf("expected source to be %s, got %s", provider.Name(), result.Source)
	}
	if result.TTL != provider.ttl {
		t.Errorf("expected TTL to be %d, got %d", provider.ttl, result.TTL)
	}
}

func TestNewGeolocationICHNAEAProvider_monitorWifiAccessPoints(t *testing.T) {
	testRequiresWiFi(t)
	t.Run("monitor WiFi access points succeeds", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			isCancelled := false
			context.AfterFunc(ctx, func() {
				isCancelled = true
			})

			provider, err := NewGeolocationICHNAEAProvider(http.New(logger.New(slog.LevelInfo)))
			if err != nil {
				t.Fatalf("failed to create ICHNAEA provider: %s", err)
			}
			go provider.monitorWifiAccessPoints(ctx)
			synctest.Wait()
			cancel()
			synctest.Wait()
			if !isCancelled {
				t.Fatal("expected monitor to be cancelled")
			}
		})
	})
}

func testRequiresWiFi(t *testing.T) {
	wlan, err := wifi.New()
	if err != nil {
		t.Skip("system has no WiFi support, skipping WiFi related tests")
	}

	checkIfaces := make([]*wifi.Interface, 0)
	ifaces, err := wlan.Interfaces()
	if err != nil {
		t.Skip("no WiFi interfaces found, skipping WiFi related tests")
	}
	for _, iface := range ifaces {
		if iface.Type != wifi.InterfaceTypeStation {
			continue
		}
		checkIfaces = append(checkIfaces, iface)
	}
	if len(checkIfaces) == 0 {
		t.Skip("no WiFi interfaces found, skipping WiFi related tests")
	}
}
