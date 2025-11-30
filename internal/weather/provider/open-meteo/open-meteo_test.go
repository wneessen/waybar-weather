// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package openmeteo

import (
	"log/slog"
	"testing"
	"time"

	"github.com/wneessen/waybar-weather/internal/geobus"
	"github.com/wneessen/waybar-weather/internal/http"
	"github.com/wneessen/waybar-weather/internal/logger"
	"github.com/wneessen/waybar-weather/internal/weather"
)

const (
	testLat = 50.95099552
	testLon = 6.929531592
)

func TestNew(t *testing.T) {
	var client weather.Provider
	var err error
	client, err = New(http.New(logger.New(slog.LevelInfo)), logger.New(slog.LevelDebug), "metric")
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}
	t.Logf("client: %+v", client.Name())
	data, err := client.GetWeather(t.Context(), geobus.Coordinate{Lat: testLat, Lon: testLon})
	if err != nil {
		t.Fatalf("failed to get weather: %s", err)
	}
	now := weather.NewDayHour(time.Now())
	t.Logf("now: %s / weather: %+v°C", now.Time().String(), data.Current.Temperature)
}
