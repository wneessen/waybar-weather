// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kkyr/fig"
)

const configEnv = "WAYBARWEATHER"

// config represents the application's configuration structure.
type config struct {
	// Allowed values: metric, imperial
	Units    string     `fig:"units" default:"metric"`
	Locale   string     `fig:"locale"`
	LogLevel slog.Level `fig:"loglevel" default:"0"`
	// Allowed values: current, forecast
	WeatherMode string `fig:"weather_mode" default:"current"`
	// Allowed value: 1 to 24
	ForecastHours uint `fig:"forecast_hours" default:"3"`

	Intervals struct {
		WeatherUpdate time.Duration `fig:"weather_update" default:"15m"`
		Output        time.Duration `fig:"output" default:"30s"`
	} `fig:"intervals"`
}

func newConfigFromFile(path, file string) (*config, error) {
	conf := new(config)
	_, err := os.Stat(filepath.Join(path, file))
	if err != nil {
		return conf, fmt.Errorf("failed to read config: %w", err)
	}
	if err = fig.Load(conf, fig.Dirs(path), fig.File(file), fig.UseEnv(configEnv)); err != nil {
		return conf, fmt.Errorf("failed to load config: %w", err)
	}

	return conf, conf.Validate()
}

func newConfig() (*config, error) {
	conf := new(config)
	if err := fig.Load(conf, fig.AllowNoFile(), fig.UseEnv(configEnv)); err != nil {
		return conf, fmt.Errorf("failed to load config: %w", err)
	}

	return conf, conf.Validate()
}

func (c *config) Validate() error {
	if c.Units != "metric" && c.Units != "imperial" {
		return fmt.Errorf("invalid units: %s", c.Units)
	}
	if c.Locale == "" {
		c.Locale = getLocale()
	}
	if c.WeatherMode != "current" && c.WeatherMode != "forecast" {
		return fmt.Errorf("invalid weather mode: %s", c.WeatherMode)
	}
	if c.WeatherMode == "forecast" && c.ForecastHours < 1 || c.ForecastHours > 24 {
		return fmt.Errorf("invalid forcast hours: %d", c.ForecastHours)
	}

	return nil
}

func getLocale() string {
	locale := os.Getenv("LC_MESSAGES")
	if idx := strings.Index(locale, "."); idx != -1 {
		lang := locale[:idx]
		return strings.ReplaceAll(lang, "_", "-")
	}
	return locale
}
