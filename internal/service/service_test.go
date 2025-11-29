// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"
	tt "text/template"
	"time"
	"unsafe"

	"github.com/hectormalot/omgo"

	"github.com/wneessen/waybar-weather/internal/config"
	"github.com/wneessen/waybar-weather/internal/geocode"
	"github.com/wneessen/waybar-weather/internal/http"
	"github.com/wneessen/waybar-weather/internal/i18n"
	"github.com/wneessen/waybar-weather/internal/logger"
	"github.com/wneessen/waybar-weather/internal/template"
)

const (
	weatherDataFile = "../../testdata/weatherdata.json"
)

func TestNew(t *testing.T) {
	t.Run("new service succeeds", func(t *testing.T) {
		_, err := testService(t, false)
		if err != nil {
			t.Fatalf("failed to create service: %s", err)
		}
	})
	t.Run("initializing service with different geocode providers", func(t *testing.T) {
		tests := []struct {
			name     string
			env      []string
			wantName string
			wantFail bool
		}{
			{
				"osm-nominatim",
				[]string{"WAYBARWEATHER_GEOCODER_PROVIDER=nominatim"},
				"osm-nominatim",
				false,
			},
			{
				"opencage without api-key",
				[]string{"WAYBARWEATHER_GEOCODER_PROVIDER=opencage"},
				"opencage",
				true,
			},
			{
				"opencage with api-key",
				[]string{
					"WAYBARWEATHER_GEOCODER_PROVIDER=opencage",
					"WAYBARWEATHER_GEOCODER_APIKEY=abc",
				},
				"opencage",
				false,
			},
			{
				"geocode.earth without api-key",
				[]string{"WAYBARWEATHER_GEOCODER_PROVIDER=geocode-earth"},
				"geocode-earth",
				true,
			},
			{
				"opencage with api-key",
				[]string{
					"WAYBARWEATHER_GEOCODER_PROVIDER=geocode-earth",
					"WAYBARWEATHER_GEOCODER_APIKEY=abc",
				},
				"geocode-earth",
				false,
			},
			{
				"unsupported provider",
				[]string{"WAYBARWEATHER_GEOCODER_PROVIDER=invalid"},
				"",
				true,
			},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				for _, envVars := range tc.env {
					vals := strings.Split(envVars, "=")
					if len(vals) != 2 {
						t.Fatalf("invalid env var %q", envVars)
					}
					t.Setenv(vals[0], vals[1])
				}
				serv, err := testService(t, false)
				if err != nil {
					t.Fatalf("failed to create service: %s", err)
				}
				if serv == nil {
					t.Fatal("expected service to be non-nil")
				}
				provider, err := serv.selectGeocodeProvider(serv.config, serv.logger, serv.t.Language())
				if tc.wantFail && err == nil {
					t.Fatal("expected geocode provider selection to fail")
				}
				if !tc.wantFail && err != nil {
					t.Fatalf("failed to select geocode provider: %s", err)
				}
				if tc.wantFail {
					return
				}
				if provider == nil {
					t.Fatal("expected geocoder to be non-nil")
				}
				name := fmt.Sprintf("geocoder cache using %s", tc.wantName)
				if provider.Name() != name {
					t.Errorf("expected geocoder name to be %q, got %q", name, provider.Name())
				}
			})
		}
	})
	t.Run("OpenWeather client UA should be the same as our default user-agent", func(t *testing.T) {
		serv, err := testService(t, false)
		if err != nil {
			t.Fatalf("failed to create service: %s", err)
		}
		if serv.omclient.UserAgent != http.UserAgent {
			t.Errorf("expected UserAgent to be %q, got %q", http.UserAgent, serv.omclient.UserAgent)
		}
	})
	t.Run("invalid template configuration should fail", func(t *testing.T) {
		t.Setenv("WAYBARWEATHER_TEMPLATES_TEXT", "{{")
		_, err := testService(t, false)
		if err == nil {
			t.Fatal("expected service creation to fail")
		}
		wantErr := "failed to parse template"
		if !strings.Contains(err.Error(), wantErr) {
			t.Errorf("expected error to contain %q, got %q", wantErr, err)
		}
	})
	t.Run("nil logger fails the geobus initialization", func(t *testing.T) {
		_, err := testService(t, true)
		if err == nil {
			t.Fatal("expected service creation to fail")
		}
		wantErr := "logger is required"
		if !strings.Contains(err.Error(), wantErr) {
			t.Errorf("expected error to contain %q, got %q", wantErr, err)
		}
	})
}

func TestService_Run(t *testing.T) {
	t.Run("start the service and gracefully shut it down", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			afterFuncCalled := false
			context.AfterFunc(ctx, func() {
				afterFuncCalled = true
			})

			serv, err := testService(t, false)
			if err != nil {
				t.Fatalf("failed to create service: %s", err)
			}

			go func() {
				if err = serv.Run(ctx); err != nil {
					t.Errorf("failed to run service: %s", err)
				}
			}()

			cancel()
			synctest.Wait()
			if !afterFuncCalled {
				t.Fatalf("before context is canceled: AfterFunc not called")
			}
		})
	})
}

func TestService_printWeather(t *testing.T) {
	t.Run("print weather to a buffer", func(t *testing.T) {
		t.Setenv("WAYBARWEATHER_TEMPLATES_TEXT", "text")
		t.Setenv("WAYBARWEATHER_TEMPLATES_TOOLTIP", "tooltip")

		serv, err := testService(t, false)
		if err != nil {
			t.Fatalf("failed to create service: %s", err)
		}
		buf := bytes.NewBuffer(nil)
		serv.output = buf
		serv.weatherIsSet = true

		serv.printWeather(t.Context())

		var output outputData
		if err = json.Unmarshal(buf.Bytes(), &output); err != nil {
			t.Fatalf("failed to unmarshal JSON: %s", err)
		}
		if output.Text != "text" {
			t.Errorf("expected Text to be %q, got %q", "text", output.Text)
		}
		if output.Tooltip != "tooltip" {
			t.Errorf("expected Tooltip to be %q, got %q", "tooltip", output.Tooltip)
		}
		if output.Class != OutputClass {
			t.Errorf("expected Class to be %q, got %q", OutputClass, output.Class)
		}
	})
	t.Run("print alt_text to a buffer", func(t *testing.T) {
		t.Setenv("WAYBARWEATHER_TEMPLATES_ALT_TEXT", "alt_text")

		serv, err := testService(t, false)
		if err != nil {
			t.Fatalf("failed to create service: %s", err)
		}
		buf := bytes.NewBuffer(nil)
		serv.output = buf
		serv.weatherIsSet = true
		serv.displayAltText = true

		serv.printWeather(t.Context())

		var output outputData
		if err = json.Unmarshal(buf.Bytes(), &output); err != nil {
			t.Fatalf("failed to unmarshal JSON: %s", err)
		}
		if output.Text != "alt_text" {
			t.Errorf("expected Text to be %q, got %q", "alt_text", output.Text)
		}
	})
	t.Run("print weather returns when weather is not set", func(t *testing.T) {
		serv, err := testService(t, false)
		if err != nil {
			t.Fatalf("failed to create service: %s", err)
		}
		buf := bytes.NewBuffer(nil)
		serv.output = buf
		serv.printWeather(t.Context())
		if buf.Len() != 0 {
			t.Errorf("expected output buffer to be empty, got %q", buf.String())
		}
	})
	t.Run("output is empty on failing writer", func(t *testing.T) {
		serv, err := testService(t, false)
		if err != nil {
			t.Fatalf("failed to create service: %s", err)
		}
		serv.output = &failWriter{}
		serv.weatherIsSet = true
		serv.printWeather(t.Context())
	})
	t.Run("printing weather fails on different template errors", func(t *testing.T) {
		tests := []struct {
			name   string
			confFn func(*config.Config)
			tplFn  func(*template.Templates, *config.Config) error
		}{
			{
				name: "text template",
				confFn: func(c *config.Config) {
					c.Templates.Text = "{{.Data}}"
				},
				tplFn: func(tpls *template.Templates, conf *config.Config) error {
					tpl, err := tt.New("text").Parse(conf.Templates.Text)
					if err != nil {
						return err
					}
					tpls.Text = tpl
					return nil
				},
			},
			{
				name: "tooltip template",
				confFn: func(c *config.Config) {
					c.Templates.Tooltip = "{{.Data}}"
				},
				tplFn: func(tpls *template.Templates, conf *config.Config) error {
					tpl, err := tt.New("tooltip").Parse(conf.Templates.Tooltip)
					if err != nil {
						return err
					}
					tpls.Tooltip = tpl
					return nil
				},
			},
			{
				name: "alt text template",
				confFn: func(c *config.Config) {
					c.Templates.AltText = "{{.Data}}"
				},
				tplFn: func(tpls *template.Templates, conf *config.Config) error {
					tpl, err := tt.New("alt_text").Parse(conf.Templates.AltText)
					if err != nil {
						return err
					}
					tpls.AltText = tpl
					return nil
				},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				serv, err := testService(t, false)
				if err != nil {
					t.Fatalf("failed to create service: %s", err)
				}
				tc.confFn(serv.config)
				if err = tc.tplFn(serv.templates, serv.config); err != nil {
					t.Fatalf("failed to parse override template: %s", err)
				}

				buf := bytes.NewBuffer(nil)
				serv.output = buf
				serv.weatherIsSet = true
				serv.printWeather(t.Context())
				if buf.Len() != 0 {
					t.Errorf("expected output buffer to be empty, got %q", buf.String())
				}
			})
		}
	})
}

func TestService_fillDisplayData(t *testing.T) {
	type currentWeather struct {
		Temperature   float64 `json:"temperature"`
		WeatherCode   float64 `json:"weather_code"`
		WindDirection float64 `json:"wind_direction"`
		WindSpeed     float64 `json:"wind_speed"`
	}
	type forecast struct {
		Latitude       float64              `json:"latitude"`
		Longitude      float64              `json:"longitude"`
		Elevation      float64              `json:"elevation"`
		CurrentWeather currentWeather       `json:"currentWeather"`
		HourlyUnits    map[string]string    `json:"hourly_units"`
		HourlyMetrics  map[string][]float64 `json:"hourlyMetrics"`
		HourlyTimes    []time.Time          `json:"hourlyTimes"`
		DailyUnits     map[string]string    `json:"daily_units"`
		DailyMetrics   map[string][]float64 `json:"dailyMetrics"`
		DailyTimes     []time.Time          `json:"dailyTimes"`
	}
	weatherJSON := new(forecast)
	data, err := os.Open(weatherDataFile)
	if err != nil {
		t.Fatalf("failed to open JSON response file: %s", err)
	}
	defer func() {
		_ = data.Close()
	}()
	if err = json.NewDecoder(data).Decode(weatherJSON); err != nil {
		t.Fatalf("failed to decode JSON response: %s", err)
	}
	weatherData := &omgo.Forecast{
		Latitude:  weatherJSON.Latitude,
		Longitude: weatherJSON.Longitude,
		Elevation: weatherJSON.Elevation,
		CurrentWeather: omgo.CurrentWeather{
			Temperature:   weatherJSON.CurrentWeather.Temperature,
			WeatherCode:   weatherJSON.CurrentWeather.WeatherCode,
			WindDirection: weatherJSON.CurrentWeather.WindDirection,
			WindSpeed:     weatherJSON.CurrentWeather.WindSpeed,
		},
		HourlyUnits:   weatherJSON.HourlyUnits,
		HourlyMetrics: weatherJSON.HourlyMetrics,
		HourlyTimes:   weatherJSON.HourlyTimes,
		DailyUnits:    weatherJSON.DailyUnits,
		DailyMetrics:  weatherJSON.DailyMetrics,
		DailyTimes:    weatherJSON.DailyTimes,
	}

	t.Run("fill display data with weather data", func(t *testing.T) {
		serv, err := testService(t, false)
		if err != nil {
			t.Fatalf("failed to create service: %s", err)
		}
		serv.weather = weatherData

		displaydata := new(template.DisplayData)
		serv.fillDisplayData(displaydata)
		if displaydata.Latitude != 44.4375 {
			t.Errorf("expected Latitude to be %f, got %f", 44.4375, displaydata.Latitude)
		}
		if displaydata.Longitude != 26.125 {
			t.Errorf("expected Longitude to be %f, got %f", 26.125, displaydata.Longitude)
		}
		if displaydata.Elevation != 85 {
			t.Errorf("expected Elevation to be %f, got %f", 85., displaydata.Elevation)
		}
		if displaydata.Address.AddressFound {
			t.Error("expected AddressFound to be false")
		}
		if displaydata.SunsetTime.IsZero() {
			t.Errorf("expected SunsetTime to be set, got %s", displaydata.SunsetTime)
		}
		if displaydata.SunriseTime.IsZero() {
			t.Errorf("expected SunriseTime to be set, got %s", displaydata.SunsetTime)
		}
		if displaydata.Moonphase != "First Quarter" {
			t.Errorf("expected Moonphase to be %q, got %q", "First Quarter", displaydata.Moonphase)
		}
		if displaydata.MoonphaseIcon != MoonPhaseIcon[displaydata.Moonphase] {
			t.Errorf("expected MoonphaseIcon to be %q, got %q", MoonPhaseIcon[displaydata.Moonphase], displaydata.MoonphaseIcon)
		}
		if displaydata.Current.Temperature != 9.1 {
			t.Errorf("expected Current.Temperature to be %f, got %f", 9.1, displaydata.Current.Temperature)
		}
	})
	t.Run("filling a nil target returns nothing", func(t *testing.T) {
		serv, err := testService(t, false)
		if err != nil {
			t.Fatalf("failed to create service: %s", err)
		}
		serv.weather = weatherData
		serv.fillDisplayData(nil)
	})
}

func TestService_selectProvider(t *testing.T) {
	tests := []struct {
		name       string
		confFn     func(*config.Config)
		shouldFail bool
	}{
		{
			name: "all providers enabled",
			confFn: func(c *config.Config) {
				c.GeoLocation.DisableGeoAPI = false
				c.GeoLocation.DisableGeoIP = false
				c.GeoLocation.DisableGeolocationFile = false
				c.GeoLocation.DisableGPSD = false
				c.GeoLocation.DisableICHNAEA = false
			},
			shouldFail: false,
		},
		{
			name: "only geo api",
			confFn: func(c *config.Config) {
				c.GeoLocation.DisableGeoAPI = false
				c.GeoLocation.DisableGeoIP = true
				c.GeoLocation.DisableGeolocationFile = true
				c.GeoLocation.DisableGPSD = true
				c.GeoLocation.DisableICHNAEA = true
			},
			shouldFail: false,
		},
		{
			name: "only geo ip",
			confFn: func(c *config.Config) {
				c.GeoLocation.DisableGeoAPI = true
				c.GeoLocation.DisableGeoIP = false
				c.GeoLocation.DisableGeolocationFile = true
				c.GeoLocation.DisableGPSD = true
				c.GeoLocation.DisableICHNAEA = true
			},
			shouldFail: false,
		},
		{
			name: "only geolocation file",
			confFn: func(c *config.Config) {
				c.GeoLocation.DisableGeoAPI = true
				c.GeoLocation.DisableGeoIP = true
				c.GeoLocation.DisableGeolocationFile = false
				c.GeoLocation.DisableGPSD = true
				c.GeoLocation.DisableICHNAEA = true
			},
			shouldFail: false,
		},
		{
			name: "only gpsd",
			confFn: func(c *config.Config) {
				c.GeoLocation.DisableGeoAPI = true
				c.GeoLocation.DisableGeoIP = true
				c.GeoLocation.DisableGeolocationFile = true
				c.GeoLocation.DisableGPSD = false
				c.GeoLocation.DisableICHNAEA = true
			},
			shouldFail: false,
		},
		{
			name: "no provider fails",
			confFn: func(c *config.Config) {
				c.GeoLocation.DisableGeoAPI = true
				c.GeoLocation.DisableGeoIP = true
				c.GeoLocation.DisableGeolocationFile = true
				c.GeoLocation.DisableGPSD = true
				c.GeoLocation.DisableICHNAEA = true
			},
			shouldFail: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			serv, err := testService(t, false)
			if err != nil {
				t.Fatalf("failed to create service: %s", err)
			}
			tc.confFn(serv.config)

			_, err = serv.selectGeobusProviders()
			if !tc.shouldFail && err != nil {
				t.Fatalf("failed to select provider: %s", err)
			}
			if tc.shouldFail && err == nil {
				t.Fatal("expected select provider to fail")
			}
		})
	}
}

func testService(_ *testing.T, nilLogger bool) (*Service, error) {
	conf, err := config.New()
	if err != nil {
		return nil, err
	}

	var log *logger.Logger
	if !nilLogger {
		log = logger.NewLogger(conf.LogLevel, io.Discard)
	}

	lang, err := i18n.New(conf.Locale)
	if err != nil {
		return nil, err
	}
	serv, err := New(conf, log, lang)
	if err != nil {
		return nil, err
	}

	return serv, nil
}

type failWriter struct{}

func (f failWriter) Write([]byte) (int, error) { return 0, fmt.Errorf("failed to write") }

type mockGeocoder struct{}

func (m *mockGeocoder) Name() string {
	return "mock geocoder"
}

func (m *mockGeocoder) Reverse(
	_ context.Context, lat, lon float64,
) (geocode.Address, error) {
	return geocode.Address{
		AddressFound: true,
		Latitude:     lat,
		Longitude:    lon,
		DisplayName:  fmt.Sprintf("Test Location %.6f,%.6f", lat, lon),
	}, nil
}

func TestService_updateLocation(t *testing.T) {
	tests := []struct {
		name      string
		latitude  float64
		longitude float64
		wantErr   bool
	}{
		{
			name:      "positive lat positive lon",
			latitude:  44.4375,
			longitude: 26.125,
			wantErr:   false,
		},
		{
			name:      "negative lat positive lon",
			latitude:  -33.8688,
			longitude: 151.2093,
			wantErr:   false,
		},
		{
			name:      "positive lat negative lon",
			latitude:  40.7128,
			longitude: -74.0060,
			wantErr:   false,
		},
		{
			name:      "negative lat negative lon",
			latitude:  -22.9068,
			longitude: -43.1729,
			wantErr:   false,
		},
		{
			name:      "zero lat zero lon",
			latitude:  0.0,
			longitude: 0.0,
			wantErr:   false,
		},
		{
			name:      "extreme north east",
			latitude:  90.0,
			longitude: 180.0,
			wantErr:   false,
		},
		{
			name:      "extreme south west",
			latitude:  -90.0,
			longitude: -180.0,
			wantErr:   false,
		},
		{
			name:      "equator prime meridian",
			latitude:  0.0,
			longitude: 0.0,
			wantErr:   false,
		},
		{
			name:      "small positive values",
			latitude:  0.000001,
			longitude: 0.000001,
			wantErr:   false,
		},
		{
			name:      "small negative values",
			latitude:  -0.000001,
			longitude: -0.000001,
			wantErr:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			serv, err := testService(t, false)
			if err != nil {
				t.Fatalf("failed to create service: %s", err)
			}

			// Set mock geocoder using unsafe to access unexported field
			rv := reflect.ValueOf(serv).Elem()
			geocoderField := rv.FieldByName("geocoder")
			if !geocoderField.IsValid() {
				t.Fatal("geocoder field not found")
			}
			// Use unsafe to set unexported interface field
			geocoderFieldPtr := unsafe.Pointer(geocoderField.UnsafeAddr())
			mockGeocoder := &mockGeocoder{}
			*(*geocode.Geocoder)(geocoderFieldPtr) = mockGeocoder

			ctx := context.Background()
			err = serv.updateLocation(ctx, tc.latitude, tc.longitude)

			if tc.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}
