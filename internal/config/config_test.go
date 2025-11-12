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
		expectLocale                = "en-US"
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
	t.Run("correct locale should be set", func(t *testing.T) {
		t.Setenv("LC_MESSAGES", "en_US.UTF-8")
		conf, err := New()
		if err != nil {
			t.Errorf("failed to load config: %s", err)
		}
		if conf.Locale != expectLocale {
			t.Errorf("expected locale to be: %s, got %s", expectLocale, conf.Locale)
		}
	})
	t.Run("invalid locale should be ignored", func(t *testing.T) {
		t.Setenv("LC_MESSAGES", "invalid")
		conf, err := New()
		if err != nil {
			t.Errorf("failed to load config: %s", err)
		}
		t.Logf("locale: %s", conf.Locale)
	})
}
