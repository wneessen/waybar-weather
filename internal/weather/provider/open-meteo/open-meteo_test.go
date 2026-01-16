// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package openmeteo

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	stdhttp "net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/wneessen/waybar-weather/internal/geobus"
	"github.com/wneessen/waybar-weather/internal/http"
	"github.com/wneessen/waybar-weather/internal/logger"
	"github.com/wneessen/waybar-weather/internal/testhelper"
	"github.com/wneessen/waybar-weather/internal/weather"
)

const (
	testLat          = 44.4375
	testLon          = 26.125
	testDataMetric   = "../../../../testdata/open-meteo.json"
	testDataImperial = "../../../../testdata/open-meteo-fahrenheit.json"
)

func TestNew(t *testing.T) {
	t.Run("creating a new provider succeeds", func(t *testing.T) {
		unit := "metric"
		client := testClient(t, unit, false)
		if client == nil {
			t.Fatal("expected client to be non-nil")
		}
		if client.unit != unit {
			t.Errorf("expected unit to be %q, got %q", unit, client.unit)
		}
		if client.http == nil {
			t.Fatal("expected http client to be non-nil")
		}
		if client.log == nil {
			t.Fatal("expected logger to be non-nil")
		}
	})
	t.Run("creating a provider without http client fails", func(t *testing.T) {
		unit := "metric"
		client, err := New(nil, logger.New(slog.LevelDebug), unit)
		if err == nil {
			t.Fatal("expected client to fail")
		}
		if client != nil {
			t.Fatal("expected client to be nil")
		}
	})
	t.Run("creating a provider without logger fails", func(t *testing.T) {
		unit := "metric"
		log := logger.NewLogger(slog.LevelDebug, io.Discard)
		httpClient := http.New(log)
		client, err := New(httpClient, nil, unit)
		if err == nil {
			t.Fatal("expected client to fail")
		}
		if client != nil {
			t.Fatal("expected client to be nil")
		}
	})
}

func TestOpenMeteo_Name(t *testing.T) {
	client := testClient(t, "metric", false)
	if client.Name() != "open-meteo" {
		t.Errorf("expected provider name to be %q, got %q", "open-meteo", client.Name())
	}
}

func TestOpenMeteo_GetWeather(t *testing.T) {
	t.Run("weather lookup succeeds", func(t *testing.T) {
		unit := "metric"
		client := testClient(t, unit, false)
		fn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
			data, err := os.Open(testDataMetric)
			if err != nil {
				t.Fatalf("failed to open JSON response file: %s", err)
			}

			return &stdhttp.Response{
				StatusCode: 200,
				Body:       data,
				Header:     make(stdhttp.Header),
			}, nil
		}
		client.http.Transport = testhelper.MockRoundTripper{Fn: fn}

		data, err := client.GetWeather(t.Context(), geobus.Coordinate{Lat: testLat, Lon: testLon})
		if err != nil {
			t.Fatalf("weather lookup failed: %s", err)
		}
		if data.GeneratedAt.IsZero() {
			t.Error("expected generated at to be set")
		}
		wantCurrent := weather.Instant{
			InstantTime:         time.Date(2026, 1, 16, 22, 0o0, 0o0, 0o0, time.Local),
			Temperature:         -5.3,
			ApparentTemperature: -9.2,
			WeatherCode:         0,
			WindSpeed:           4.7,
			WindGusts:           12.2,
			WindDirection:       81,
			RelativeHumidity:    72,
			PressureMSL:         1034.7,
		}
		if data.Current.Temperature != wantCurrent.Temperature {
			t.Errorf("expected current temperature to be %f, got %f", wantCurrent.Temperature,
				data.Current.Temperature)
		}
		if data.Current.ApparentTemperature != wantCurrent.ApparentTemperature {
			t.Errorf("expected current apparent temperature to be %f, got %f", wantCurrent.ApparentTemperature,
				data.Current.ApparentTemperature)
		}
		if data.Current.WeatherCode != wantCurrent.WeatherCode {
			t.Errorf("expected current weather code to be %d, got %d", wantCurrent.WeatherCode,
				data.Current.WeatherCode)
		}
		if data.Current.WindSpeed != wantCurrent.WindSpeed {
			t.Errorf("expected current wind speed to be %f, got %f", wantCurrent.WindSpeed,
				data.Current.WindSpeed)
		}
		if data.Current.WindGusts != wantCurrent.WindGusts {
			t.Errorf("expected current wind gusts to be %f, got %f", wantCurrent.WindGusts,
				data.Current.WindGusts)
		}
		if data.Current.WindDirection != wantCurrent.WindDirection {
			t.Errorf("expected current wind direction to be %f, got %f", wantCurrent.WindDirection,
				data.Current.WindDirection)
		}
		if data.Current.RelativeHumidity != wantCurrent.RelativeHumidity {
			t.Errorf("expected current relative humidity to be %f, got %f", wantCurrent.RelativeHumidity,
				data.Current.RelativeHumidity)
		}
		if data.Current.PressureMSL != wantCurrent.PressureMSL {
			t.Errorf("expected current pressure MSL to be %f, got %f", wantCurrent.PressureMSL,
				data.Current.PressureMSL)
		}
		wantFCast := weather.Instant{
			Temperature:         -3.0,
			ApparentTemperature: -6.6,
			WeatherCode:         3,
			WindSpeed:           6.4,
			WindGusts:           16.6,
			WindDirection:       232,
			RelativeHumidity:    91,
			PressureMSL:         1022.2,
		}
		fcastTime := weather.NewDayHour(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))
		fcast := data.Forecast[fcastTime]
		if fcast.Temperature != wantFCast.Temperature {
			t.Errorf("expected forecast temperature to be %f, got %f", wantFCast.Temperature, fcast.Temperature)
		}
		if fcast.ApparentTemperature != wantFCast.ApparentTemperature {
			t.Errorf("expected forecast apparent temperature to be %f, got %f", wantFCast.ApparentTemperature,
				fcast.ApparentTemperature)
		}
		if fcast.WeatherCode != wantFCast.WeatherCode {
			t.Errorf("expected forecast weather code to be %d, got %d", wantFCast.WeatherCode, fcast.WeatherCode)
		}
		if fcast.WindSpeed != wantFCast.WindSpeed {
			t.Errorf("expected forecast wind speed to be %f, got %f", wantFCast.WindSpeed, fcast.WindSpeed)
		}
		if fcast.WindGusts != wantFCast.WindGusts {
			t.Errorf("expected forecast wind gusts to be %f, got %f", wantFCast.WindGusts, fcast.WindGusts)
		}
		if fcast.WindDirection != wantFCast.WindDirection {
			t.Errorf("expected forecast wind direction to be %f, got %f", wantFCast.WindDirection,
				fcast.WindDirection)
		}
		if fcast.RelativeHumidity != wantFCast.RelativeHumidity {
			t.Errorf("expected forecast relative humidity to be %f, got %f", wantFCast.RelativeHumidity,
				fcast.RelativeHumidity)
		}
		if fcast.PressureMSL != wantFCast.PressureMSL {
			t.Errorf("expected forecast pressure MSL to be %f, got %f", wantFCast.PressureMSL, fcast.PressureMSL)
		}
		wantUnits := map[string]string{
			"temperature": "°C",
			"pressure":    "hPa",
			"windspeed":   "km/h",
			"humidity":    "%",
			"winddir":     "°",
		}
		if data.Current.Units.Temperature != wantUnits["temperature"] {
			t.Errorf("expected current temperature units to be %q, got %q", wantUnits["temperature"],
				data.Current.Units.Temperature)
		}
		if data.Current.Units.Pressure != wantUnits["pressure"] {
			t.Errorf("expected current pressure units to be %q, got %q", wantUnits["pressure"],
				data.Current.Units.Pressure)
		}
		if data.Current.Units.WindSpeed != wantUnits["windspeed"] {
			t.Errorf("expected current wind speed units to be %q, got %q", wantUnits["windspeed"],
				data.Current.Units.WindSpeed)
		}
		if data.Current.Units.Humidity != wantUnits["humidity"] {
			t.Errorf("expected current humidity units to be %q, got %q", wantUnits["humidity"],
				data.Current.Units.Humidity)
		}
		if data.Current.Units.WindDirection != wantUnits["winddir"] {
			t.Errorf("expected current wind direction units to be %q, got %q", wantUnits["winddir"],
				data.Current.Units.WindDirection)
		}
	})
	t.Run("weather lookup with imperial unit succeeds", func(t *testing.T) {
		unit := "imperial"
		client := testClient(t, unit, false)
		fn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
			data, err := os.Open(testDataImperial)
			if err != nil {
				t.Fatalf("failed to open JSON response file: %s", err)
			}

			return &stdhttp.Response{
				StatusCode: 200,
				Body:       data,
				Header:     make(stdhttp.Header),
			}, nil
		}
		client.http.Transport = testhelper.MockRoundTripper{Fn: fn}
		data, err := client.GetWeather(t.Context(), geobus.Coordinate{Lat: testLat, Lon: testLon})
		if err != nil {
			t.Fatalf("weather lookup failed: %s", err)
		}
		if data.GeneratedAt.IsZero() {
			t.Error("expected generated at to be set")
		}
		wantCurrent := weather.Instant{
			Temperature:         22.5,
			ApparentTemperature: 15.5,
			WeatherCode:         0,
			WindSpeed:           2.9,
			WindGusts:           7.6,
			WindDirection:       81,
			RelativeHumidity:    72,
			PressureMSL:         1034.7,
		}
		if data.Current.Temperature != wantCurrent.Temperature {
			t.Errorf("expected current temperature to be %f, got %f", wantCurrent.Temperature,
				data.Current.Temperature)
		}
		if data.Current.ApparentTemperature != wantCurrent.ApparentTemperature {
			t.Errorf("expected current apparent temperature to be %f, got %f", wantCurrent.ApparentTemperature,
				data.Current.ApparentTemperature)
		}
		if data.Current.WeatherCode != wantCurrent.WeatherCode {
			t.Errorf("expected current weather code to be %d, got %d", wantCurrent.WeatherCode,
				data.Current.WeatherCode)
		}
		if data.Current.WindSpeed != wantCurrent.WindSpeed {
			t.Errorf("expected current wind speed to be %f, got %f", wantCurrent.WindSpeed,
				data.Current.WindSpeed)
		}
		if data.Current.WindGusts != wantCurrent.WindGusts {
			t.Errorf("expected current wind gusts to be %f, got %f", wantCurrent.WindGusts,
				data.Current.WindGusts)
		}
		if data.Current.WindDirection != wantCurrent.WindDirection {
			t.Errorf("expected current wind direction to be %f, got %f", wantCurrent.WindDirection,
				data.Current.WindDirection)
		}
		if data.Current.RelativeHumidity != wantCurrent.RelativeHumidity {
			t.Errorf("expected current relative humidity to be %f, got %f", wantCurrent.RelativeHumidity,
				data.Current.RelativeHumidity)
		}
		if data.Current.PressureMSL != wantCurrent.PressureMSL {
			t.Errorf("expected current pressure MSL to be %f, got %f", wantCurrent.PressureMSL,
				data.Current.PressureMSL)
		}
		wantUnits := map[string]string{
			"temperature": "°F",
			"pressure":    "hPa",
			"windspeed":   "mp/h",
			"humidity":    "%",
			"winddir":     "°",
		}
		if data.Current.Units.Temperature != wantUnits["temperature"] {
			t.Errorf("expected current temperature units to be %q, got %q", wantUnits["temperature"],
				data.Current.Units.Temperature)
		}
		if data.Current.Units.Pressure != wantUnits["pressure"] {
			t.Errorf("expected current pressure units to be %q, got %q", wantUnits["pressure"],
				data.Current.Units.Pressure)
		}
		if data.Current.Units.WindSpeed != wantUnits["windspeed"] {
			t.Errorf("expected current wind speed units to be %q, got %q", wantUnits["windspeed"],
				data.Current.Units.WindSpeed)
		}
		if data.Current.Units.Humidity != wantUnits["humidity"] {
			t.Errorf("expected current humidity units to be %q, got %q", wantUnits["humidity"],
				data.Current.Units.Humidity)
		}
		if data.Current.Units.WindDirection != wantUnits["winddir"] {
			t.Errorf("expected current wind direction units to be %q, got %q", wantUnits["winddir"],
				data.Current.Units.WindDirection)
		}
	})
	t.Run("http request fails with a 401", func(t *testing.T) {
		client := testClient(t, "", false)
		fn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
			data := bytes.NewBufferString(`{"status": 401, "message": "Unauthorized"}`)
			return &stdhttp.Response{
				StatusCode: 401,
				Body:       io.NopCloser(data),
				Header:     make(stdhttp.Header),
			}, nil
		}
		client.http.Transport = testhelper.MockRoundTripper{Fn: fn}

		_, err := client.GetWeather(t.Context(), geobus.Coordinate{Lat: testLat, Lon: testLon})
		if err == nil {
			t.Error("expected error to be returned")
		}
		wantErr := `Open-Meteo API returned non-positive response code: 401`
		if !strings.Contains(err.Error(), wantErr) {
			t.Errorf("expected error to contain %q, got %q", wantErr, err)
		}
	})
	t.Run("http request fails unmarshalling the JSON", func(t *testing.T) {
		client := testClient(t, "", false)
		fn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
			data := bytes.NewBufferString(`invalid`)
			return &stdhttp.Response{
				StatusCode: 401,
				Body:       io.NopCloser(data),
				Header:     make(stdhttp.Header),
			}, nil
		}
		client.http.Transport = testhelper.MockRoundTripper{Fn: fn}

		_, err := client.GetWeather(t.Context(), geobus.Coordinate{Lat: testLat, Lon: testLon})
		if err == nil {
			t.Error("expected error to be returned")
		}
		wantErr := `failed to decode JSON: invalid character 'i'`
		if !strings.Contains(err.Error(), wantErr) {
			t.Errorf("expected error to contain %q, got %q", wantErr, err)
		}
	})
}

func TestResBool_UnmarshalJSON(t *testing.T) {
	t.Run("true/false are correctly unmarshalled", func(t *testing.T) {
		tests := []struct {
			name string
			json []byte
			want bool
		}{
			{"true", []byte(`{"value":1}`), true},
			{"false", []byte(`{"value":0}`), false},
		}

		for _, tc := range tests {
			type data struct {
				Value resBool `json:"value"`
			}
			var output data
			if err := json.Unmarshal(tc.json, &output); err != nil {
				t.Fatalf("failed to unmarshal JSON: %s", err)
			}
			if tc.want != output.Value.bool {
				t.Errorf("expected value to be %t, got %t", tc.want, output.Value.bool)
			}
		}
	})
}

func TestResTime_UnmarshalJSON(t *testing.T) {
	t.Run("unmarshalling diferent times succeeds", func(t *testing.T) {
		tests := []struct {
			name  string
			json  []byte
			want  time.Time
			fails bool
		}{
			{
				"2006-01-02T15:04",
				[]byte(`{"value":"2006-01-02T15:04"}`),
				time.Date(2006, 1, 2, 15, 4, 0, 0, time.UTC),
				false,
			},
			{
				"2006-01-02T15:04:00 (extra text fails)",
				[]byte(`{"value":"2006-01-02T15:04:00"}`),
				time.Time{},
				true,
			},
			{
				"nil",
				[]byte(`{"value":null}`),
				time.Time{},
				true,
			},
		}

		for _, tc := range tests {
			type data struct {
				Value resTime `json:"value"`
			}
			var output data
			if err := json.Unmarshal(tc.json, &output); err != nil && !tc.fails {
				t.Fatalf("failed to unmarshal JSON: %s", err)
			}
			if tc.fails {
				continue
			}
			if !output.Value.Time.Equal(tc.want) {
				t.Errorf("expected value to be %s, got %s", tc.want, output.Value.Time)
			}
		}
	})
}

func testClient(t *testing.T, unit string, nilLogger bool) *OpenMeteo {
	var output io.Writer = os.Stdout
	if nilLogger {
		output = io.Discard
	}
	if unit == "" {
		unit = "metric"
	}
	log := logger.NewLogger(slog.LevelDebug, output)
	httpClient := http.New(log)
	client, err := New(httpClient, log, unit)
	if err != nil {
		t.Fatalf("failed to create open-meteo client: %s", err)
	}
	return client
}
