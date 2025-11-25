// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package geobus

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/wneessen/waybar-weather/internal/logger"
)

func TestGeolocationState_HasChanged(t *testing.T) {
	t.Run("empty state always returns true", func(t *testing.T) {
		state := GeolocationState{}
		if !state.HasChanged(Coordinate{Lat: 1, Lon: 1, Acc: AccuracyZip}) {
			t.Error("expected state to have changed")
		}
	})
	t.Run("same coordinate return false", func(t *testing.T) {
		state := GeolocationState{}
		state.Update(Coordinate{Lat: 1, Lon: 1, Acc: AccuracyZip})
		if state.HasChanged(Coordinate{Lat: 1, Lon: 1, Acc: AccuracyZip}) {
			t.Error("expected state to not have changed")
		}
	})
	t.Run("different coordinate return true", func(t *testing.T) {
		tests := []struct {
			name    string
			lat     float64
			lon     float64
			acc     float64
			changed bool
		}{
			{"lat changes", 2, 1, AccuracyZip, true},
			{"lon changes", 1, 2, AccuracyZip, true},
			// an accuracy change is not considered a significant positional change
			{"acc changes", 1, 1, AccuracyCity, false},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				state := GeolocationState{}
				state.Update(Coordinate{Lat: 1, Lon: 1, Acc: AccuracyZip})
				if state.HasChanged(Coordinate{Lat: tc.lat, Lon: tc.lon, Acc: tc.acc}) != tc.changed {
					t.Error("expected state change to be", tc.changed, "but it wasn't")
				}
			})
		}
	})
}

func TestCoordinate_PosHasSignificantChange(t *testing.T) {
	tests := []struct {
		name    string
		coord   Coordinate
		other   Coordinate
		changed bool
	}{
		{
			name: "same point, no change",
			coord: Coordinate{
				Lat: 50.0,
				Lon: 8.0,
			},
			other: Coordinate{
				Lat: 50.0,
				Lon: 8.0,
			},
			changed: false,
		},
		{
			name: "small move within threshold",
			coord: Coordinate{
				Lat: 50.0,
				Lon: 8.0,
			},
			other: Coordinate{
				Lat: 50.01,
				Lon: 8.01,
			},
			changed: false,
		},
		{
			name: "small move within threshold with negative coordinates",
			coord: Coordinate{
				Lat: -50.0,
				Lon: -8.0,
			},
			other: Coordinate{
				Lat: -50.01,
				Lon: -8.01,
			},
			changed: false,
		},
		{
			name: "just above threshold",
			coord: Coordinate{
				Lat: 50.0,
				Lon: 8.0,
			},
			other: Coordinate{
				Lat: 50.0225,
				Lon: 8.0,
			},
			changed: true,
		},
		{
			name: "far above threshold but negative coordinates",
			coord: Coordinate{
				Lat: -50.0,
				Lon: -8.0,
			},
			other: Coordinate{
				Lat: -50.0,
				Lon: -8.1,
			},
			changed: true,
		},
		{
			name: "large movement, Berlin to Paris",
			coord: Coordinate{
				Lat: 52.52,
				Lon: 13.405,
			},
			other: Coordinate{
				Lat: 48.8566,
				Lon: 2.3522,
			},
			changed: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.coord.PosHasSignificantChange(tc.other) != tc.changed {
				t.Error("expected change to be", tc.changed, "but it wasn't")
			}
		})
	}
}

func TestResult_BetterThan(t *testing.T) {
	tests := []struct {
		name   string
		new    Result
		prev   Result
		better bool
	}{
		{
			name:   "same result, no difference",
			new:    Result{Key: "test", Lat: 1, Lon: 1, AccuracyMeters: 1},
			prev:   Result{Key: "test", Lat: 1, Lon: 1, AccuracyMeters: 1},
			better: false,
		},
		{
			name:   "previous result had no key",
			new:    Result{Key: "test", Lat: 1, Lon: 1, AccuracyMeters: 1},
			prev:   Result{Lat: 1, Lon: 1, AccuracyMeters: 1},
			better: true,
		},
		{
			name:   "previous result is newer",
			new:    Result{Key: "test", At: time.Date(2024, time.January, 1, 16, 56, 0, 0, time.UTC)},
			prev:   Result{Key: "test", At: time.Date(2025, time.January, 1, 16, 56, 0, 0, time.UTC)},
			better: false,
		},
		{
			name:   "new result is more accurate",
			new:    Result{Key: "test", AccuracyMeters: AccuracyZip},
			prev:   Result{Key: "test", AccuracyMeters: AccuracyCity},
			better: true,
		},
		{
			name:   "previsou result is more accurate",
			new:    Result{Key: "test", AccuracyMeters: AccuracyCity},
			prev:   Result{Key: "test", AccuracyMeters: AccuracyZip},
			better: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.new.BetterThan(tc.prev) != tc.better {
				t.Error("expected new to be", tc.better, "but it wasn't")
			}
		})
	}
}

func TestResult_IsExpired(t *testing.T) {
	result := Result{At: time.Now().Add(-time.Hour), TTL: time.Hour}
	if !result.IsExpired() {
		t.Error("expected result to be expired")
	}
	result = Result{At: time.Now().Add(-time.Hour), TTL: time.Hour * 2}
	if result.IsExpired() {
		t.Error("expected result to not be expired")
	}
}

func TestNew(t *testing.T) {
	bus := New(logger.New(slog.LevelInfo))
	if bus == nil {
		t.Fatal("expected bus to be non-nil")
	}
	if bus.logger == nil {
		t.Fatal("expected logger to be non-nil")
	}
	if bus.best == nil {
		t.Fatal("expected best provider to be non-nil")
	}
	if bus.subscribers == nil {
		t.Fatal("expected subscribers to be non-nil")
	}
	if bus.globalSubs == nil {
		t.Fatal("expected global subscribers to be non-nil")
	}
}

func TestGeoBus_NewOrchestrator(t *testing.T) {
	bus := New(logger.New(slog.LevelInfo))
	if bus == nil {
		t.Fatal("expected bus to be non-nil")
	}
	orch := bus.NewOrchestrator([]Provider{&mockProvider{Results: []Result{{Key: "test"}}}})
	if orch == nil {
		t.Fatal("expected orchestrator to be non-nil")
	}
}

/*
func TestGeoBus_Subscribe(t *testing.T) {
	id := "waybar-weather"
	bus := New(logger.New(slog.LevelInfo))
	if bus == nil {
		t.Fatal("expected bus to be non-nil")
	}

	provider := &mockProvider{
		ProviderName: "test-provider",
		Results: []Result{
			{
				Key:    id,
				Lat:    50.0,
				Lon:    8.0,
				Alt:    100,
				Source: "mock",
				At:     time.Now(),
			},
			{
				Key:    id,
				Lat:    51.0,
				Lon:    9.0,
				Alt:    100,
				Source: "mock",
				At:     time.Now(),
			},
		},
		Delay: time.Millisecond * 500,
	}
	orch := bus.NewOrchestrator([]Provider{provider})

		t.Run("subscribe to global updates", func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				var results []Result

				sub, unsub := bus.Subscribe(id, 32)
				defer unsub()
				go orch.Track(ctx, id)

				go func() {
					for len(results) <= 2 {
						select {
						case <-ctx.Done():
							cancel()
							return
						case r, ok := <-sub:
							t.Logf("received geolocation update from %+v", r)
							if !ok {
								return
							}
							results = append(results, r)
						}
					}
				}()
				synctest.Wait()
			})
		})

}
*/

// MockProvider is a test double for the Provider interface.
type mockProvider struct {
	ProviderName string
	Results      []Result
	Delay        time.Duration
}

// Name returns the configured provider name.
func (m *mockProvider) Name() string {
	return "mock-provider"
}

// LookupStream returns a channel that streams the configured Results.
// The channel is closed after all results are sent or when ctx is cancelled.
func (m *mockProvider) LookupStream(ctx context.Context, key string) <-chan Result {
	ch := make(chan Result)

	go func() {
		defer close(ch)

		for _, res := range m.Results {
			select {
			case <-ctx.Done():
				return
			case <-time.After(m.Delay):
				ch <- res
			}
		}
	}()

	return ch
}

func processLocationUpdates(ctx context.Context, ch <-chan Result, t *testing.T) {
}
