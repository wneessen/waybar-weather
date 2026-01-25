// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.de
//
// SPDX-License-Identifier: MIT

package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kkyr/fig"
)

const (
	configEnv         = "WAYBARWEATHER"
	DefaultTextTpl    = "{{.Current.ConditionIcon}} {{hum .Current.Temperature}}{{.Current.Units.Temperature}}"
	DefaultAltTextTpl = "{{.Forecast.ConditionIcon}} {{hum .Forecast.Temperature}}{{.Forecast.Units.Temperature}}"
	DefaultTooltipTpl = "{{.Address.City}}, {{.Address.Country}}\n" +
		"{{.Current.Condition}}\n" +
		"{{loc \"apparent\"}}: {{hum .Current.ApparentTemperature}}{{.Current.Units.Temperature}}\n" +
		"{{loc \"humidity\"}}: {{.Current.RelativeHumidity}}%\n" +
		"{{loc \"pressure\"}}: {{hum .Current.PressureMSL}} {{.Current.Units.Pressure}}\n" +
		"{{loc \"wind\"}}: {{hum .Current.WindSpeed}} → {{hum .Current.WindGusts}} {{.Current.Units.WindSpeed}} ({{windDir .Current.WindDirection}})\n" +
		"\n" +
		`🌅 {{localizedTime .SunriseTime}} • 🌇 {{localizedTime .SunsetTime}}`
	DefaultAltTooltipTpl = "{{.Address.City}}, {{.Address.Country}}\n" +
		"{{.Forecast.Condition}}\n" +
		"{{loc \"apparent\"}}: {{hum .Forecast.ApparentTemperature}}{{.Forecast.Units.Temperature}}\n" +
		"{{loc \"humidity\"}}: {{.Forecast.RelativeHumidity}}%\n" +
		"{{loc \"pressure\"}}: {{hum .Forecast.PressureMSL}} {{.Forecast.Units.Pressure}}\n" +
		"{{loc \"wind\"}}: {{hum .Forecast.WindSpeed}} → {{hum .Forecast.WindGusts}} {{.Forecast.Units.WindSpeed}} ({{windDir .Forecast.WindDirection}})\n" +
		"\n" +
		`🌅 {{localizedTime .SunriseTime}} • 🌇 {{localizedTime .SunsetTime}}`
)

// Config represents the application's configuration structure.
type Config struct {
	// Allowed values: metric, imperial
	Units    string     `fig:"units" default:"metric"`
	Locale   string     `fig:"locale"`
	LogLevel slog.Level `fig:"loglevel" default:"0"`

	Weather struct {
		Provider string `fig:"provider" default:"open-meteo"`

		// Allowed value: 1 to 24
		ForecastHours uint `fig:"forecast_hours" default:"3"`

		// Cold and hot class thresholds (Defaults are based on °C)
		// Defaults are based on suggestions for dangerous driving conditions and uncomfortable heat.
		ColdThreshold float64 `fig:"cold_threshold" default:"2"`
		HotThreshold  float64 `fig:"hot_threshold" default:"30"`
	} `fig:"weather"`

	Intervals struct {
		WeatherUpdate time.Duration `fig:"weather_update" default:"15m"`
		Output        time.Duration `fig:"output" default:"30s"`
	} `fig:"intervals"`

	Templates struct {
		Text       string `fig:"text"`
		AltText    string `fig:"alt_text"`
		Tooltip    string `fig:"tooltip"`
		AltTooltip string `fig:"alt_tooltip"`
		UseCSSIcon bool   `fig:"use_css_icon"`
	} `fig:"templates"`

	GeoLocation struct {
		GeoLocationFile        string `fig:"geolocation_file"`
		CitynameFile           string `fig:"cityname_file"`
		DisableGeoIP           bool   `fig:"disable_geoip"`
		DisableGeoAPI          bool   `fig:"disable_geoapi"`
		DisableGeolocationFile bool   `fig:"disable_geolocation_file"`
		DisableCitynameFile    bool   `fig:"disable_cityname_file"`
		DisableICHNAEA         bool   `fig:"disable_ichnaea"`
		DisableGPSD            bool   `fig:"disable_gpsd"`
	} `fig:"geolocation"`

	GeoCoder struct {
		Provider string `fig:"provider" default:"nominatim"`
		APIKey   string `fig:"apikey"`
	} `fig:"geocoder"`
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
	if c.Templates.AltTooltip == "" {
		c.Templates.AltTooltip = DefaultAltTooltipTpl
	}
	if c.GeoLocation.GeoLocationFile == "" {
		home, _ := os.UserHomeDir()
		c.GeoLocation.GeoLocationFile = filepath.Join(home, ".config", "waybar-weather", "geolocation")
	}
	if c.GeoLocation.CitynameFile == "" {
		home, _ := os.UserHomeDir()
		c.GeoLocation.CitynameFile = filepath.Join(home, ".config", "waybar-weather", "cityname")
	}
	if c.Templates.UseCSSIcon {
		if strings.EqualFold(c.Templates.Text, DefaultTextTpl) {
			c.Templates.Text = ` {{hum .Current.Temperature}}{{.Current.Units.Temperature}}`
		}
		if strings.EqualFold(c.Templates.AltText, DefaultAltTextTpl) {
			c.Templates.Text = ` {{hum .Forecast.Temperature}}{{.Forecast.Units.Temperature}}`
		}
	}

	return nil
}
