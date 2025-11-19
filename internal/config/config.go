// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/kkyr/fig"
)

const (
	configEnv         = "WAYBARWEATHER"
	DefaultTextTpl    = "{{.Current.ConditionIcon}} {{.Current.Temperature}}{{.TempUnit}}"
	DefaultAltTextTpl = "{{.Forecast.ConditionIcon}} {{.Forecast.Temperature}}{{.TempUnit}}"
	DefaultTooltipTpl = "{{.Address.City}}, {{.Address.Country}}\n" +
		"{{.Current.Condition}}\n" +
		"{{loc \"apparent\"}}: {{.Current.ApparentTemperature}}{{.TempUnit}}\n" +
		"{{loc \"humidity\"}}: {{.Current.Humidity}}%\n" +
		"{{loc \"pressure\"}}: {{.Current.PressureMSL}} {{.PressureUnit}}\n" +
		"{{loc \"moonphase\"}}: {{.MoonphaseIcon}} {{loc .Moonphase}}\n" +
		"\n" +
		`🌅 {{timeFormat .SunriseTime "15:04"}} • 🌇 {{timeFormat .SunsetTime "15:04"}}`
)

// Config represents the application's configuration structure.
type Config struct {
	// Allowed values: metric, imperial
	Units    string     `fig:"units" default:"metric"`
	Locale   string     `fig:"locale"`
	LogLevel slog.Level `fig:"loglevel" default:"0"`

	Weather struct {
		// Allowed value: 1 to 24
		ForecastHours uint `fig:"forecast_hours" default:"3"`
	} `fig:"weather"`

	Intervals struct {
		WeatherUpdate time.Duration `fig:"weather_update" default:"15m"`
		Output        time.Duration `fig:"output" default:"30s"`
	} `fig:"intervals"`

	Templates struct {
		Text    string `fig:"text"`
		AltText string `fig:"alt_text"`
		Tooltip string `fig:"tooltip"`
	} `fig:"templates"`

	GeoLocation struct {
		File                   string `fig:"file"`
		DisableGeoIP           bool   `fig:"disable_geoip"`
		DisableGeoAPI          bool   `fig:"disable_geoapi"`
		DisableGeolocationFile bool   `fig:"disable_geolocation_file"`
		DisableICHNAEA         bool   `fig:"disable_ichnaea"`
		DisableGPSD            bool   `fig:"disable_gpsd"`
	} `fig:"geolocation"`
}

func NewFromFile(path, file string) (*Config, error) {
	conf := new(Config)
	_, err := os.Stat(filepath.Join(path, file))
	if err != nil {
		return conf, fmt.Errorf("failed to read Config: %w", err)
	}
	if err = fig.Load(conf, fig.Dirs(path), fig.File(file), fig.UseEnv(configEnv)); err != nil {
		return conf, fmt.Errorf("failed to load Config: %w", err)
	}

	return conf, conf.Validate()
}

func New() (*Config, error) {
	conf := new(Config)
	if err := fig.Load(conf, fig.AllowNoFile(), fig.UseEnv(configEnv)); err != nil {
		return conf, fmt.Errorf("failed to load Config: %w", err)
	}

	return conf, conf.Validate()
}

func (c *Config) Validate() error {
	if c.Units != "metric" && c.Units != "imperial" {
		return fmt.Errorf("invalid units: %s", c.Units)
	}
	if c.Weather.ForecastHours < 1 || c.Weather.ForecastHours > 24 {
		return fmt.Errorf("invalid forcast hours: %d", c.Weather.ForecastHours)
	}
	if c.Templates.Text == "" {
		c.Templates.Text = DefaultTextTpl
	}
	if c.Templates.AltText == "" {
		c.Templates.AltText = DefaultAltTextTpl
	}
	if c.Templates.Tooltip == "" {
		c.Templates.Tooltip = DefaultTooltipTpl
	}
	if c.GeoLocation.File == "" {
		home, _ := os.UserHomeDir()
		c.GeoLocation.File = filepath.Join(home, ".config", "waybar-weather", "geolocation")
	}

	return nil
}
