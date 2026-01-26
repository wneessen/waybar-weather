// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package config

import (
	"log/slog"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	const (
		expectDefaultUnits          = "metric"
		expectLogLevel              = slog.LevelInfo
		expectWeatherForecastHours  = 3
		expectIntervalWeatherUpdate = time.Minute * 15
		expectIntervalOutput        = time.Second * 30
	)
	t.Run("new config with all defaults set", func(t *testing.T) {
		conf, err := New()
		if err != nil {
			t.Errorf("failed to load config: %s", err)
		}
		if conf.Units != expectDefaultUnits {
			t.Errorf("expected units to be: %s, got %s", expectDefaultUnits, conf.Units)
		}
		if conf.LogLevel != expectLogLevel {
			t.Errorf("expected log level to be: %s, got %s", expectLogLevel, conf.LogLevel)
		}
		if conf.Weather.ForecastHours != expectWeatherForecastHours {
			t.Errorf("expected weather forecast hours to be: %d, got %d", expectWeatherForecastHours,
				conf.Weather.ForecastHours)
		}
		if conf.Intervals.WeatherUpdate != expectIntervalWeatherUpdate {
			t.Errorf("expected weather update interval to be: %s, got %s", expectIntervalWeatherUpdate,
				conf.Intervals.WeatherUpdate)
		}
		if conf.Intervals.Output != expectIntervalOutput {
			t.Errorf("expected output interval to be: %s, got %s", expectIntervalOutput, conf.Intervals.Output)
		}
	})
	t.Run("config in CSS icon mode should change the template texts", func(t *testing.T) {
		t.Setenv("WAYBARWEATHER_TEMPLATES_USE_CSS_ICON", "true")
		t.Setenv("WAYBARWEATHER_TEMPLATES_TEXT", "")
		t.Setenv("WAYBARWEATHER_TEMPLATES_ALT_TEXT", "")
		conf, err := New()
		if err != nil {
			t.Fatalf("failed to load config: %s", err)
		}
		if !conf.Templates.UseCSSIcon {
			t.Error("expected CSS icon mode to be enabled")
		}
		wantText := ` {{hum .Current.Temperature}}{{.Current.Units.Temperature}}`
		wantAltText := ` {{hum .Forecast.Temperature}}{{.Forecast.Units.Temperature}}`
		if conf.Templates.Text != wantText {
			t.Errorf("failed to set text template in CSS icon mode: got: %q, want: %q", conf.Templates.Text,
				wantText)
		}
		if conf.Templates.AltText != wantAltText {
			t.Errorf("failed to set alternative text template in CSS icon mode: got: %q, want: %q",
				conf.Templates.AltText, wantAltText)
		}
	})
	t.Run("new config with invalid values from env", func(t *testing.T) {
		t.Setenv("WAYBARWEATHER_LOGLEVEL", "invalid")
		_, err := New()
		if err == nil {
			t.Error("expected config to fail, but didn't")
		}
	})
	t.Run("config validate forecast hours", func(t *testing.T) {
		t.Setenv("WAYBARWEATHER_WEATHER_FORECAST_HOURS", "-1")
		_, err := New()
		if err == nil {
			t.Error("expected config to fail, but didn't")
		}
		t.Setenv("WAYBARWEATHER_WEATHER_FORECAST_HOURS", "25")
		_, err = New()
		if err == nil {
			t.Error("expected config to fail, but didn't")
		}
	})
	t.Run("config validate units", func(t *testing.T) {
		t.Setenv("WAYBARWEATHER_UNITS", "invalid")
		_, err := New()
		if err == nil {
			t.Error("expected config to fail, but didn't")
		}
	})
}

func TestNewFromFile(t *testing.T) {
	const (
		expectDefaultUnits          = "metric"
		expectLogLevel              = slog.LevelInfo
		expectWeatherForecastHours  = 3
		expectIntervalWeatherUpdate = time.Minute * 15
		expectIntervalOutput        = time.Second * 30
	)
	t.Run("reading config from valid file succeeds", func(t *testing.T) {
		conf, err := NewFromFile("../../etc", "config.toml")
		if err != nil {
			t.Fatalf("failed to load config: %s", err)
		}
		if conf.Units != expectDefaultUnits {
			t.Errorf("expected units to be: %s, got %s", expectDefaultUnits, conf.Units)
		}
		if conf.LogLevel != expectLogLevel {
			t.Errorf("expected log level to be: %s, got %s", expectLogLevel, conf.LogLevel)
		}
		if conf.Weather.ForecastHours != expectWeatherForecastHours {
			t.Errorf("expected weather forecast hours to be: %d, got %d", expectWeatherForecastHours,
				conf.Weather.ForecastHours)
		}
		if conf.Intervals.WeatherUpdate != expectIntervalWeatherUpdate {
			t.Errorf("expected weather update interval to be: %s, got %s", expectIntervalWeatherUpdate,
				conf.Intervals.WeatherUpdate)
		}
		if conf.Intervals.Output != expectIntervalOutput {
			t.Errorf("expected output interval to be: %s, got %s", expectIntervalOutput, conf.Intervals.Output)
		}
	})
	t.Run("reading config from non-existent file fails", func(t *testing.T) {
		_, err := NewFromFile("../../etc", "non-existent.toml")
		if err == nil {
			t.Error("expected config to fail, but didn't")
		}
	})
	t.Run("reading invalid config file fails", func(t *testing.T) {
		_, err := NewFromFile("../../testdata", "invalid.toml")
		if err == nil {
			t.Error("expected config to fail, but didn't")
		}
	})
}
