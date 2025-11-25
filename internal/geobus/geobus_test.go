// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package geobus

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"testing"
	"time"

	"github.com/wneessen/waybar-weather/internal/logger"
)

const (
	subID = "test"
)

func TestGeolocationState_Update(t *testing.T) {
	state := GeolocationState{}
	state.Update(Coordinate{Lat: 50.0, Lon: 8.0})
	if state.last.Lat != 50.0 || state.last.Lon != 8.0 {
		t.Error("expected last coordinate to be updated")
	}
	if !state.haveLast {
		t.Error("expected haveLast to be true")
	}
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
		{
			name: "same place but significantly better accuracy",
			coord: Coordinate{
				Lat: 52.52,
				Lon: 13.405,
				Acc: AccuracyCity,
			},
			other: Coordinate{
				Lat: 52.52,
				Lon: 13.405,
				Acc: AccuracyCountry,
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
			name:   "previous result is more accurate",
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
	bus, err := New(logger.New(slog.LevelInfo))
	if err != nil {
		t.Fatalf("failed to create bus: %s", err)
	}
	if bus == nil {
		t.Fatal("expected bus to be non-nil")
	}
	if bus.best == nil {
		t.Fatal("expected best provider to be non-nil")
	}
	if bus.subscribers == nil {
		t.Fatal("expected subscribers to be non-nil")
	}
}

func TestGeoBus_Publish(t *testing.T) {
	t.Run("a siggnificant change publishes a result", func(t *testing.T) {
		bus, err := New(logger.New(slog.LevelInfo))
		if err != nil {
			t.Fatalf("failed to create bus: %s", err)
		}
		ch, unsub := bus.Subscribe(subID, 1)
		defer unsub()

		bus.Publish(Result{
			Key:            subID,
			Lat:            50.0,
			Lon:            8.0,
			AccuracyMeters: 20,
			At:             time.Now(),
			Source:         "mock-provider",
		})
		<-ch
		bus.Publish(Result{
			Key:            subID,
			Lat:            55.0001,
			Lon:            9.0001,
			AccuracyMeters: 20,
			At:             time.Now(),
			Source:         "mock-provider",
		})
		select {
		case <-ch:
			t.Fatalf("did not expect update for insignificant movement")
		case <-time.After(50 * time.Millisecond):
		}
	})
	t.Run("do not publish results without accuracy", func(t *testing.T) {
		bus, err := New(logger.New(slog.LevelInfo))
		if err != nil {
			t.Fatalf("failed to create bus: %s", err)
		}
		ch, unsub := bus.Subscribe(subID, 1)
		defer unsub()

		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()

		bus.Publish(Result{
			Key:            subID,
			Lat:            50.0,
			Lon:            8.0,
			AccuracyMeters: 0,
			At:             time.Now(),
			Source:         "mock-provider",
			TTL:            time.Millisecond * 500,
		})

		for {
			select {
			case <-ctx.Done():
				return
			case <-ch:
				t.Fatalf("did not expect update for insignificant movement")
			}
		}
	})
	t.Run("no At time sets it to 'now'", func(t *testing.T) {
		bus, err := New(logger.New(slog.LevelInfo))
		if err != nil {
			t.Fatalf("failed to create bus: %s", err)
		}
		ch, unsub := bus.Subscribe(subID, 1)
		defer unsub()

		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()

		bus.Publish(Result{
			Key:            subID,
			Lat:            50.0,
			Lon:            8.0,
			AccuracyMeters: 1,
			Source:         "mock-provider",
			TTL:            time.Millisecond * 500,
		})

		var result *Result
		for result == nil {
			select {
			case <-ctx.Done():
				return
			case r := <-ch:
				result = &r
			}
		}
		if result == nil {
			t.Fatal("expected result to be non-nil")
		}
		if result.At.IsZero() {
			t.Fatal("expected At time to be set")
		}
	})
}

func TestTrackProviders(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	bus, err := New(logger.New(slog.LevelInfo))
	if err != nil {
		t.Fatalf("failed to create bus: %s", err)
	}
	fp := &fakeProvider{name: "test", ch: make(chan Result, 1)}
	TrackProviders(ctx, bus, "k", fp)

	sub, unsub := bus.Subscribe(subID, 1)
	defer unsub()

	r := Result{
		Key:            subID,
		Lat:            1,
		Lon:            2,
		AccuracyMeters: 10,
		At:             time.Now(),
		TTL:            time.Millisecond * 500,
	}
	fp.ch <- r

	got := <-sub
	if got.Lat != r.Lat || got.Lon != r.Lon {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestTruncate(t *testing.T) {
	in := "123.456789"
	for i := 5; i >= 1; i-- {
		t.Run(fmt.Sprintf("truncate float down to precision: %d", i), func(t *testing.T) {
			val := in[:4+i]
			num, err := strconv.ParseFloat(val, 64)
			if err != nil {
				t.Fatalf("failed to parse float: %s", err)
			}

			want := Truncate(num, i)
			if want != num {
				t.Errorf("expected %f, got %f", num, want)
			}
		})
	}
}

type fakeProvider struct {
	name string
	ch   chan Result
}

func (f *fakeProvider) Name() string { return f.name }

func (f *fakeProvider) LookupStream(ctx context.Context, key string) <-chan Result {
	return f.ch
}
