// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package presenter

import (
	"fmt"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/vorlif/spreak"

	"github.com/wneessen/waybar-weather/internal/config"
	"github.com/wneessen/waybar-weather/internal/geobus"
	"github.com/wneessen/waybar-weather/internal/geocode"
	"github.com/wneessen/waybar-weather/internal/i18n"
	"github.com/wneessen/waybar-weather/internal/weather"
)

var (
	now  = time.Now()
	addr = geocode.Address{
		AddressFound: true,
		Latitude:     12.345,
		Longitude:    67.890,
		City:         "Test City",
		Country:      "Test Country",
		DisplayName:  "Test City, Test Country",
	}
	sunrise        = time.Date(2026, 1, 18, 7, 1, 2, 0, time.UTC)
	sunset         = time.Date(2026, 1, 18, 17, 39, 41, 0, time.UTC)
	moonphase      = "Waxing Gibbous"
	fcastHour      = weather.NewDayHour(now.Add(time.Hour * 3))
	fcastHourFirst = weather.NewDayHour(now.Add(time.Hour))
	wthr           = weather.Instant{
		InstantTime:         now,
		Temperature:         20.0,
		ApparentTemperature: 25.0,
		WeatherCode:         45,
		WindDirection:       67,
		WindSpeed:           10.0,
		WindGusts:           30.0,
		RelativeHumidity:    87,
		PressureMSL:         1013.2,
		IsDay:               true,
		Units: weather.Units{
			Temperature:   "°C",
			WindSpeed:     "km/h",
			Humidity:      "%",
			Pressure:      "hPa",
			WindDirection: "°",
		},
	}
	wthrAlt = weather.Instant{
		InstantTime:         fcastHour.Time(),
		Temperature:         25.0,
		ApparentTemperature: 30.0,
		WeatherCode:         1,
		WindDirection:       185,
		WindSpeed:           3.0,
		WindGusts:           19.0,
		RelativeHumidity:    43,
		PressureMSL:         1083.4,
		IsDay:               false,
		Units: weather.Units{
			Temperature:   "°F",
			WindSpeed:     "m/h",
			Humidity:      "%",
			Pressure:      "hPa",
			WindDirection: "°",
		},
	}
)

func TestNew(t *testing.T) {
	t.Run("creating a new presenter succeeds", func(t *testing.T) {
		conf, lang := testConfLang(t)
		pres, err := New(conf, lang)
		if err != nil {
			t.Fatalf("failed to create presenter: %s", err)
		}
		if pres == nil {
			t.Fatal("expected presenter to be non-nil")
		}
	})
	t.Run("creating presenter with invalid templates fails", func(t *testing.T) {
		tests := []struct {
			name       string
			templateFn func(conf *config.Config)
		}{
			{"text", func(conf *config.Config) { conf.Templates.Text = "{{invalid" }},
			{"alt_text", func(conf *config.Config) { conf.Templates.AltText = "{{invalid" }},
			{"tooltip", func(conf *config.Config) { conf.Templates.Tooltip = "{{invalid" }},
			{"alt_tooltip", func(conf *config.Config) { conf.Templates.AltTooltip = "{{invalid" }},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				conf, lang := testConfLang(t)
				tt.templateFn(conf)
				_, err := New(conf, lang)
				if err == nil {
					t.Error("expected presenter to fail, but didn't")
				}
				wantErr := "failed to parse"
				if !strings.Contains(err.Error(), wantErr) {
					t.Errorf("expected error to contain %q, got %q", wantErr, err)
				}
			})
		}
	})
	t.Run("creating presenter with template execution errors fails", func(t *testing.T) {
		tests := []struct {
			name       string
			templateFn func(conf *config.Config)
		}{
			{"text", func(conf *config.Config) { conf.Templates.Text = "{{.Data}}" }},
			{"alt_text", func(conf *config.Config) { conf.Templates.AltText = "{{.Data}}" }},
			{"tooltip", func(conf *config.Config) { conf.Templates.Tooltip = "{{.Data}}" }},
			{"alt_tooltip", func(conf *config.Config) { conf.Templates.AltTooltip = "{{.Data}}" }},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				conf, lang := testConfLang(t)
				tt.templateFn(conf)
				_, err := New(conf, lang)
				if err == nil {
					t.Error("expected presenter to fail, but didn't")
				}
				wantErr := "failed to render"
				if !strings.Contains(err.Error(), wantErr) {
					t.Errorf("expected error to contain %q, got %q", wantErr, err)
				}
			})
		}
	})
}

func TestPresenter_BuildContext(t *testing.T) {
	t.Run("building context succeeds", func(t *testing.T) {
		conf, lang := testConfLang(t)
		pres, err := New(conf, lang)
		if err != nil {
			t.Fatalf("failed to create presenter: %s", err)
		}

		fcasts := make(map[weather.DayHour]weather.Instant)
		fcasts[fcastHour] = wthrAlt
		fcasts[fcastHourFirst] = wthrAlt
		data := &weather.Data{
			GeneratedAt: now,
			Coordinates: geobus.Coordinate{Lat: addr.Latitude, Lon: addr.Longitude},
			Current:     wthr,
			Forecast:    fcasts,
		}
		tplCtx := pres.BuildContext(addr, data, sunrise, sunset, moonphase)
		if tplCtx.UpdateTime.IsZero() {
			t.Error("expected update time to be set")
		}
		if tplCtx.Address.City != addr.City {
			t.Errorf("expected address city to be %q, got %q", addr.City, tplCtx.Address.City)
		}
		if tplCtx.Address.Country != addr.Country {
			t.Errorf("expected address country to be %q, got %q", addr.Country, tplCtx.Address.Country)
		}
		if tplCtx.Current.Temperature != wthr.Temperature {
			t.Errorf("expected current temperature to be %f, got %f", wthr.Temperature,
				tplCtx.Current.Temperature)
		}
		if tplCtx.Current.ApparentTemperature != wthr.ApparentTemperature {
			t.Errorf("expected current apparent temperature to be %f, got %f", wthr.ApparentTemperature,
				tplCtx.Current.ApparentTemperature)
		}
		if tplCtx.Current.WeatherCode != wthr.WeatherCode {
			t.Errorf("expected current weather code to be %d, got %d", wthr.WeatherCode,
				tplCtx.Current.WeatherCode)
		}
		if tplCtx.Current.WindSpeed != wthr.WindSpeed {
			t.Errorf("expected current wind speed to be %f, got %f", wthr.WindSpeed, tplCtx.Current.WindSpeed)
		}
		if tplCtx.Current.WindGusts != wthr.WindGusts {
			t.Errorf("expected current wind gusts to be %f, got %f", wthr.WindGusts, tplCtx.Current.WindGusts)
		}
		if tplCtx.Current.WindDirection != wthr.WindDirection {
			t.Errorf("expected current wind direction to be %f, got %f", wthr.WindDirection,
				tplCtx.Current.WindDirection)
		}
		if tplCtx.Current.RelativeHumidity != wthr.RelativeHumidity {
			t.Errorf("expected current humidity to be %f, got %f", wthr.RelativeHumidity,
				tplCtx.Current.RelativeHumidity)
		}
		if tplCtx.Forecast.Temperature != wthrAlt.Temperature {
			t.Errorf("expected forecast temperature to be %f, got %f", wthrAlt.Temperature,
				tplCtx.Forecast.Temperature)
		}
		if len(tplCtx.Forecasts) != 2 {
			t.Fatalf("expected forecasts to have length 1, got %d", len(tplCtx.Forecasts))
		}
		if tplCtx.Forecasts[0].Temperature != wthrAlt.Temperature {
			t.Errorf("expected forecast temperature to be %f, got %f", wthrAlt.Temperature,
				tplCtx.Forecasts[0].Temperature)
		}
		wantCategory := "fog"
		if tplCtx.Current.Category != wantCategory {
			t.Errorf("expected current weather category to be %q, got %q", wantCategory, tplCtx.Current.Category)
		}
		wantAltCategory := "clear"
		if tplCtx.Forecast.Category != wantAltCategory {
			t.Errorf("expected forecast weather category to be %q, got %q", wantAltCategory,
				tplCtx.Forecast.Category)
		}
		wantMoonIcon := "🌔"
		if tplCtx.MoonPhaseIcon != wantMoonIcon {
			t.Errorf("expected moon phase icon to be %q, got %q", wantMoonIcon, tplCtx.MoonPhaseIcon)
		}
		wantCondition := "Fog"
		if tplCtx.Current.Condition != wantCondition {
			t.Errorf("expected current weather condition to be %q, got %q", wantCondition,
				tplCtx.Current.Condition)
		}
		wantAltCondition := "Mainly clear"
		if tplCtx.Forecast.Condition != wantAltCondition {
			t.Errorf("expected forecast weather condition to be %q, got %q", wantAltCondition,
				tplCtx.Forecast.Condition)
		}
		wantCondIcon := "🌫️"
		if tplCtx.Current.ConditionIcon != wantCondIcon {
			t.Errorf("expected current weather condition icon to be %q, got %q", wantCondIcon,
				tplCtx.Current.ConditionIcon)
		}
		wantAltCondIcon := "🌙"
		if tplCtx.Forecast.ConditionIcon != wantAltCondIcon {
			t.Errorf("expected forecast weather condition icon to be %q, got %q", wantAltCondIcon,
				tplCtx.Forecast.ConditionIcon)
		}
	})
	t.Run("building context with nil weather data returns an empty context", func(t *testing.T) {
		conf, lang := testConfLang(t)
		pres, err := New(conf, lang)
		if err != nil {
			t.Fatalf("failed to create presenter: %s", err)
		}

		tplCtx := pres.BuildContext(addr, nil, sunrise, sunset, moonphase)
		if !tplCtx.UpdateTime.IsZero() {
			t.Errorf("expected update time to be zero, got %s", tplCtx.UpdateTime)
		}
	})
}

func TestPresenter_Render(t *testing.T) {
	t.Run("rendering succeeds", func(t *testing.T) {
		conf, lang := testConfLang(t)
		pres, err := New(conf, lang)
		if err != nil {
			t.Fatalf("failed to create presenter: %s", err)
		}

		fcasts := make(map[weather.DayHour]weather.Instant)
		fcasts[fcastHour] = wthrAlt
		fcasts[fcastHourFirst] = wthrAlt
		data := &weather.Data{
			GeneratedAt: now,
			Coordinates: geobus.Coordinate{Lat: addr.Latitude, Lon: addr.Longitude},
			Current:     wthr,
			Forecast:    fcasts,
		}
		tplCtx := pres.BuildContext(addr, data, sunrise, sunset, moonphase)
		outMap, err := pres.Render(tplCtx)
		if err != nil {
			t.Fatalf("failed to render: %s", err)
		}
		if len(outMap) != 4 {
			t.Errorf("expected output map to have length 4, got %d", len(outMap))
		}
		wantAltText := "🌙 25.0°F"
		wantText := "🌫️ 20.0°C"
		wantAltTooltip := `Test City, Test Country
Mainly clear
Feels like: 30.0°F
Humidity: 43%
Pressure: 1,083.4 hPa
Wind: 3.0 → 19.0 m/h (S)

🌅 7:01 a.m. • 🌇 5:39 p.m.`
		wantTooltip := `Test City, Test Country
Fog
Feels like: 25.0°C
Humidity: 87%
Pressure: 1,013.2 hPa
Wind: 10.0 → 30.0 km/h (NE)

🌅 7:01 a.m. • 🌇 5:39 p.m.`
		if outMap["text"] != wantText {
			t.Errorf("expected text output to be %q, got %q", wantText, outMap["text"])
		}
		if outMap["alt_text"] != wantAltText {
			t.Errorf("expected alt_text output to be %q, got %q", wantAltText, outMap["alt_text"])
		}
		if outMap["alt_tooltip"] != wantAltTooltip {
			t.Errorf("expected alt_tooltip output to be %q, got %q", wantAltTooltip, outMap["alt_tooltip"])
		}
		if outMap["tooltip"] != wantTooltip {
			t.Errorf("expected tooltip output to be %q, got %q", wantTooltip, outMap["tooltip"])
		}
	})
	t.Run("rendering with invalid templates fails", func(t *testing.T) {
		tests := []struct {
			name       string
			templateFn func(*Presenter) error
		}{
			{
				"text",
				func(pres *Presenter) error {
					tpltext := "{{.Data}}"
					tpl, err := template.New("text").Parse(tpltext)
					if err != nil {
						return fmt.Errorf("failed to parse template: %w", err)
					}
					pres.TextTemplate = tpl
					return nil
				},
			},
			{
				"alt_text",
				func(pres *Presenter) error {
					tpltext := "{{.Data}}"
					tpl, err := template.New("alt_text").Parse(tpltext)
					if err != nil {
						return fmt.Errorf("failed to parse template: %w", err)
					}
					pres.AltTextTemplate = tpl
					return nil
				},
			},
			{
				"tooltip",
				func(pres *Presenter) error {
					tpltext := "{{.Data}}"
					tpl, err := template.New("tooltip").Parse(tpltext)
					if err != nil {
						return fmt.Errorf("failed to parse template: %w", err)
					}
					pres.TooltipTemplate = tpl
					return nil
				},
			},
			{
				"alt_tooltip",
				func(pres *Presenter) error {
					tpltext := "{{.Data}}"
					tpl, err := template.New("alt_tooltip").Parse(tpltext)
					if err != nil {
						return fmt.Errorf("failed to parse template: %w", err)
					}
					pres.AltTooltipTemplate = tpl
					return nil
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				conf, lang := testConfLang(t)
				pres, err := New(conf, lang)
				if err != nil {
					t.Fatalf("failed to create presenter: %s", err)
				}
				if err = tt.templateFn(pres); err != nil {
					t.Fatalf("failed to set template: %s", err)
				}
				data := &weather.Data{
					GeneratedAt: now,
					Coordinates: geobus.Coordinate{Lat: addr.Latitude, Lon: addr.Longitude},
					Current:     wthr,
				}
				tplCtx := pres.BuildContext(addr, data, sunrise, sunset, moonphase)
				_, err = pres.Render(tplCtx)
				if err == nil {
					t.Error("expected rendering to fail, but didn't")
				}
			})
		}
	})
}

func TestPresenter_weatherCategory(t *testing.T) {
	tests := []struct {
		name string
		code int
		want string
	}{
		{"clear", 0, "clear"},
		{"cloudy", 2, "cloudy"},
		{"fog", 45, "fog"},
		{"rain", 51, "rain"},
		{"rain-56", 56, "rain"},
		{"rain-66", 56, "rain"},
		{"rain-80", 56, "rain"},
		{"snow", 71, "snow"},
		{"thunderstorm", 95, "thunderstorm"},
		{"empty", 100, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := weatherCategory(tt.code); got != tt.want {
				t.Errorf("failed to get weather category: got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPresenter_degToString(t *testing.T) {
	tests := []struct {
		name string
		deg  float64
		want string
	}{
		{"0 -> North", 0, "N"},
		{"22.4 -> North", 22.4, "N"},
		{"22.5 -> North-East", 22.5, "NE"},
		{"67.4 -> North-East", 67.4, "NE"},
		{"67.5 -> East", 67.5, "E"},
		{"112.4 -> East", 112.4, "E"},
		{"112.5 -> South-East", 112.5, "SE"},
		{"157.4 -> South-East", 157.4, "SE"},
		{"157.5 -> South", 157.5, "S"},
		{"202.4 -> South", 202.4, "S"},
		{"202.5 -> South-West", 202.5, "SW"},
		{"247.4 -> South-West", 247.4, "SW"},
		{"247.5 -> West", 247.5, "W"},
		{"292.4 -> West", 292.4, "W"},
		{"292.5 -> North-West", 292.5, "NW"},
		{"337.4 -> North-West", 337.4, "NW"},
		{"337.5 -> North", 337.5, "N"},
		{"359.9 -> North", 359.9, "N"},
		{"360.0 -> North", 360.0, "N"},
	}

	pres := new(Presenter)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pres.degToString(tt.deg)
			if got != tt.want {
				t.Errorf("failed to get direction: got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestPresenter_loc(t *testing.T) {
	t.Run("localized value is found", func(t *testing.T) {
		conf, lang := testConfLang(t)
		pres, err := New(conf, lang)
		if err != nil {
			t.Fatalf("failed to create presenter: %s", err)
		}
		want := "Temperature"
		if got := pres.loc("temp"); got != want {
			t.Errorf("failed to get localized value: got %s, want %s", got, want)
		}
	})
	t.Run("localized german value is found", func(t *testing.T) {
		conf, err := config.New()
		if err != nil {
			t.Fatalf("failed to create config: %s", err)
		}
		lang, err := i18n.New("de-DE")
		if err != nil {
			t.Fatalf("failed to create i18n provider: %s", err)
		}
		pres, err := New(conf, lang)
		if err != nil {
			t.Fatalf("failed to create presenter: %s", err)
		}
		want := "Gefühlt"
		if got := pres.loc("apparent"); got != want {
			t.Errorf("failed to get localized value: got %s, want %s", got, want)
		}
	})
	t.Run("localized value is not found", func(t *testing.T) {
		conf, lang := testConfLang(t)
		pres, err := New(conf, lang)
		if err != nil {
			t.Fatalf("failed to create presenter: %s", err)
		}
		want := "foobar"
		if got := pres.loc("foobar"); got != want {
			t.Errorf("failed to get localized value: got %s, want %s", got, want)
		}
	})
}

func TestPresenter_timeFormat(t *testing.T) {
	t.Run("RFC3339 format is used", func(t *testing.T) {
		pres := new(Presenter)
		if got := pres.timeFormat(now, time.RFC3339); got != now.Format(time.RFC3339) {
			t.Errorf("failed to get time format: got %s, want %s", got, now.Format(time.RFC3339))
		}
	})
}

func TestPresenter_floatFormat(t *testing.T) {
	tests := []struct {
		name string
		val  float64
		prec int
		want string
	}{
		{"0.0", 0.0, 0, "0"},
		{"0.4", 0.4, 1, "0.4"},
		{"0.6", 0.6, 1, "0.6"},
		{"0.1234", 0.1234, 4, "0.1234"},
		{"0.123", 0.1234, 3, "0.123"},
		{"0.12", 0.1234, 2, "0.12"},
		{"0.1", 0.1234, 1, "0.1"},
		{"0", 0.1234, 0, "0"},
	}

	pres := new(Presenter)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pres.floatFormat(tt.val, tt.prec); got != tt.want {
				t.Errorf("failed to get float format: got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestPresenter_windDirIcon(t *testing.T) {
	tests := []struct {
		name string
		val  string
		want string
	}{
		{"North", "N", "↑"},
		{"North-East", "NE", "↗"},
		{"East", "E", "→"},
		{"South-East", "SE", "↘"},
		{"South", "S", "↓"},
		{"South-West", "SW", "↙"},
		{"West", "W", "←"},
		{"North-West", "NW", "↖"},
		{"North", "n", "↑"},
		{"North-East", "ne", "↗"},
		{"East", "e", "→"},
		{"South-East", "se", "↘"},
		{"South", "s", "↓"},
		{"South-West", "sw", "↙"},
		{"West", "w", "←"},
		{"North-West", "nw", "↖"},
		{"Unknown", "Unknown", ""},
	}

	pres := new(Presenter)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pres.windDirIcon(tt.val); got != tt.want {
				t.Errorf("failed to get wind direction icon: got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestPresenter_forecastByOffset(t *testing.T) {
	t.Run("forecast is found", func(t *testing.T) {
		conf, lang := testConfLang(t)
		pres, err := New(conf, lang)
		if err != nil {
			t.Fatalf("failed to create presenter: %s", err)
		}
		fcasts := make(map[weather.DayHour]weather.Instant)
		for i := -23; i < 25; i++ {
			fcast := wthr
			offset := time.Hour * time.Duration(i)
			fcast.InstantTime = now.Add(offset).Truncate(time.Hour)
			hour := weather.NewDayHour(fcast.InstantTime)
			fcasts[hour] = fcast
		}
		data := &weather.Data{
			GeneratedAt: now,
			Coordinates: geobus.Coordinate{Lat: addr.Latitude, Lon: addr.Longitude},
			Current:     wthr,
			Forecast:    fcasts,
		}
		tplCtx := pres.BuildContext(addr, data, sunrise, sunset, moonphase)

		got := pres.forecastByOffset(tplCtx, 3)
		if got.Temperature != wthr.Temperature {
			t.Errorf("failed to get forecast by offset: got %f, want %f", got.Temperature,
				wthr.Temperature)
		}
	})
	t.Run("forecast is not found", func(t *testing.T) {
		conf, lang := testConfLang(t)
		pres, err := New(conf, lang)
		if err != nil {
			t.Fatalf("failed to create presenter: %s", err)
		}
		fcasts := make(map[weather.DayHour]weather.Instant)
		data := &weather.Data{
			GeneratedAt: now,
			Coordinates: geobus.Coordinate{Lat: addr.Latitude, Lon: addr.Longitude},
			Current:     wthr,
			Forecast:    fcasts,
		}
		tplCtx := pres.BuildContext(addr, data, sunrise, sunset, moonphase)

		got := pres.forecastByOffset(tplCtx, 3)
		if got.Temperature != 0 {
			t.Errorf("failed to get forecast by offset: got %f, want 0", got.Temperature)
		}
	})
}

func testConfLang(t *testing.T) (*config.Config, *spreak.Localizer) {
	t.Helper()
	conf, err := config.New()
	if err != nil {
		t.Fatalf("failed to create config: %s", err)
	}
	lang, err := i18n.New(conf.Locale)
	if err != nil {
		t.Fatalf("failed to create i18n provider: %s", err)
	}
	return conf, lang
}
