// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	stdhttp "net/http"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"testing/synctest"
	tt "text/template"
	"time"

	"github.com/wneessen/waybar-weather/internal/config"
	"github.com/wneessen/waybar-weather/internal/geobus"
	"github.com/wneessen/waybar-weather/internal/geocode"
	"github.com/wneessen/waybar-weather/internal/http"
	"github.com/wneessen/waybar-weather/internal/i18n"
	"github.com/wneessen/waybar-weather/internal/logger"
	"github.com/wneessen/waybar-weather/internal/presenter"
	"github.com/wneessen/waybar-weather/internal/testhelper"
	"github.com/wneessen/waybar-weather/internal/weather"
	openmeteo "github.com/wneessen/waybar-weather/internal/weather/provider/open-meteo"
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
			serv.jobs = append(serv.jobs, nil)

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
	t.Run("starting service fails due to invalid geocoding provider", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			serv, err := testService(t, false)
			if err != nil {
				t.Fatalf("failed to create service: %s", err)
			}
			serv.config.GeoCoder.Provider = "invalid"
			err = serv.Run(t.Context())
			if err == nil {
				t.Fatal("expected service to fail")
			}
			wantErr := `failed to create geocode provider: unsupported geocoder type: invalid`
			if !strings.Contains(err.Error(), wantErr) {
				t.Errorf("expected error to contain %q, got %q", wantErr, err)
			}
		})
	})
	t.Run("starting service fails due to invalid geobus provider", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			serv, err := testService(t, false)
			if err != nil {
				t.Fatalf("failed to create service: %s", err)
			}
			serv.config.GeoLocation.DisableGeoAPI = true
			serv.config.GeoLocation.DisableGeoIP = true
			serv.config.GeoLocation.DisableGPSD = true
			serv.config.GeoLocation.DisableGeolocationFile = true
			serv.config.GeoLocation.DisableCitynameFile = true
			serv.config.GeoLocation.DisableICHNAEA = true
			err = serv.Run(t.Context())
			if err == nil {
				t.Fatal("expected service to fail")
			}
			wantErr := `failed to create geobus orchestrator: no geolocation providers enabled`
			if !strings.Contains(err.Error(), wantErr) {
				t.Errorf("expected error to contain %q, got %q", wantErr, err)
			}
		})
	})
	t.Run("starting service fails due to invalid weather provider", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			serv, err := testService(t, false)
			if err != nil {
				t.Fatalf("failed to create service: %s", err)
			}
			serv.config.Weather.Provider = "invalid"
			err = serv.Run(t.Context())
			if err == nil {
				t.Fatal("expected service to fail")
			}
			wantErr := `failed to create weather provider: unsupported weather provider: invalid`
			if !strings.Contains(err.Error(), wantErr) {
				t.Errorf("expected error to contain %q, got %q", wantErr, err)
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
		wantClasses := 2
		if len(output.Classes) != wantClasses {
			t.Errorf("expected Classes to have length %d, got %d", wantClasses, len(output.Classes))
		}
		if output.Classes[0] != OutputClass {
			t.Errorf("expected first class to be %q, got %q", OutputClass, output.Classes[0])
		}
		if output.Classes[1] != ColdOutputClass {
			t.Errorf("expected 2nd class to be %q, got %q", ColdOutputClass, output.Classes[1])
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
	t.Run("printing weather fails on template rendering", func(t *testing.T) {
		tests := []struct {
			name    string
			confFn  func(*config.Config)
			tplFn   func(pres *presenter.Presenter, conf *config.Config) error
			wantErr string
		}{
			{
				name: "text template",
				confFn: func(c *config.Config) {
					c.Templates.Text = "{{.AbsolutelyInvalid}}"
				},
				tplFn: func(pres *presenter.Presenter, conf *config.Config) error {
					tpl, err := tt.New("text").Parse(conf.Templates.Text)
					if err != nil {
						return err
					}
					pres.TextTemplate = tpl
					return nil
				},
				wantErr: "text template",
			},
			{
				name: "alternative text template",
				confFn: func(c *config.Config) {
					c.Templates.AltText = "{{.AbsolutelyInvalid}}"
				},
				tplFn: func(pres *presenter.Presenter, conf *config.Config) error {
					tpl, err := tt.New("alt_text").Parse(conf.Templates.AltText)
					if err != nil {
						return err
					}
					pres.AltTextTemplate = tpl
					return nil
				},
				wantErr: "alt text template",
			},
			{
				name: "tooltip template",
				confFn: func(c *config.Config) {
					c.Templates.Tooltip = "{{.AbsolutelyInvalid}}"
				},
				tplFn: func(pres *presenter.Presenter, conf *config.Config) error {
					tpl, err := tt.New("tooltip").Parse(conf.Templates.Tooltip)
					if err != nil {
						return err
					}
					pres.TooltipTemplate = tpl
					return nil
				},
				wantErr: "tooltip template",
			},
			{
				name: "alternative tooltip template",
				confFn: func(c *config.Config) {
					c.Templates.AltTooltip = "{{.AbsolutelyInvalid}}"
				},
				tplFn: func(pres *presenter.Presenter, conf *config.Config) error {
					tpl, err := tt.New("alt_tooltip").Parse(conf.Templates.AltTooltip)
					if err != nil {
						return err
					}
					pres.AltTooltipTemplate = tpl
					return nil
				},
				wantErr: "alt tooltip template",
			},
		}

		for _, tc := range tests {
			serv, err := testService(t, false)
			if err != nil {
				t.Fatalf("failed to create service: %s", err)
			}
			tc.confFn(serv.config)
			if err = tc.tplFn(serv.presenter, serv.config); err != nil {
				t.Fatalf("failed to update presenter template: %s", err)
			}
			serv.weatherIsSet = true

			logBuf := bytes.NewBuffer(nil)
			serv.logger = logger.NewLogger(slog.LevelError, logBuf)

			buf := bytes.NewBuffer(nil)
			serv.output = buf
			serv.printWeather(t.Context())
			wantErr1 := `msg="failed to render weather template" error="failed to render ` + tc.wantErr
			wantErr2 := `can't evaluate field AbsolutelyInvalid in type presenter.TemplateContext`
			if !strings.Contains(logBuf.String(), wantErr1) || !strings.Contains(logBuf.String(), wantErr2) {
				t.Errorf("expected error to contain %q and %q, got %q", wantErr1, wantErr2, logBuf.String())
			}
		}
	})
	t.Run("hot and cold thresholds return correct output classes", func(t *testing.T) {
		tests := []struct {
			name        string
			weatherData *weather.Data
			altMode     bool
			wantClass   string
		}{
			{
				name: "it is hot",
				weatherData: &weather.Data{
					Current:  weather.Instant{Temperature: 25},
					Forecast: make(map[weather.DayHour]weather.Instant),
				},
				altMode:   false,
				wantClass: "hot",
			},
			{
				name: "it is cold",
				weatherData: &weather.Data{
					Current:  weather.Instant{Temperature: -25},
					Forecast: make(map[weather.DayHour]weather.Instant),
				},
				altMode:   false,
				wantClass: "cold",
			},
			{
				name: "it is hot (alt)",
				weatherData: &weather.Data{
					Current:  weather.Instant{Temperature: 25},
					Forecast: make(map[weather.DayHour]weather.Instant),
				},
				altMode:   true,
				wantClass: "hot",
			},
			{
				name: "it is cold (alt)",
				weatherData: &weather.Data{
					Current:  weather.Instant{Temperature: -25},
					Forecast: make(map[weather.DayHour]weather.Instant),
				},
				altMode:   true,
				wantClass: "cold",
			},
		}

		for _, tc := range tests {
			serv, err := testService(t, false)
			if err != nil {
				t.Fatalf("failed to create service: %s", err)
			}
			serv.config.Weather.HotThreshold = 10
			serv.config.Weather.ColdThreshold = -10
			now := time.Now()
			fcastNow := now.Add(time.Hour * time.Duration(serv.config.Weather.ForecastHours))
			tc.weatherData.Current.InstantTime = now
			fcast := tc.weatherData.Current
			fcast.InstantTime = fcastNow
			tc.weatherData.Forecast[weather.NewDayHour(fcastNow)] = fcast
			serv.weatherIsSet = true
			serv.weather = tc.weatherData
			serv.displayAltText = tc.altMode
			buf := bytes.NewBuffer(nil)
			serv.output = buf
			serv.printWeather(t.Context())

			var output outputData
			if err = json.Unmarshal(buf.Bytes(), &output); err != nil {
				t.Fatalf("failed to unmarshal JSON: %s", err)
			}

			found := false
			for _, class := range output.Classes {
				if class == tc.wantClass {
					found = true
				}
			}
			if !found {
				t.Errorf("expected output class to be %q, got %#v", tc.wantClass, output.Classes)
			}
		}
	})
}

func TestService_fetchWeather(t *testing.T) {
	t.Run("fetching weather with mock providers succeeds", func(t *testing.T) {
		serv, err := testService(t, false)
		if err != nil {
			t.Fatalf("failed to create service: %s", err)
		}
		serv.weatherProv = &weatherProv{}
		serv.fetchWeather(t.Context())
		if serv.weather == nil {
			t.Error("expected weather to be set")
		}
		if serv.weather.Current.InstantTime.IsZero() {
			t.Errorf("expected weather instant time to be set, got %s", serv.weather.Current.InstantTime)
		}
		if serv.weather.GeneratedAt.IsZero() {
			t.Errorf("expected weather generated at to be set, got %s", serv.weather.GeneratedAt)
		}
		wantTemp := 20.0
		if serv.weather.Current.Temperature != wantTemp {
			t.Errorf("expected weather temperature to be %f, got %f", wantTemp, serv.weather.Current.Temperature)
		}
	})
	t.Run("fetching weather with mock providers fails", func(t *testing.T) {
		serv, err := testService(t, false)
		if err != nil {
			t.Fatalf("failed to create service: %s", err)
		}
		buf := bytes.NewBuffer(nil)
		serv.logger = logger.NewLogger(slog.LevelError, buf)
		serv.weatherProv = &weatherProv{shouldFail: true}
		serv.fetchWeather(t.Context())
		if serv.weather != nil {
			t.Errorf("expected weather to not bet set, got: %+v", serv.weather)
		}
		wantErr := `msg="failed to fetch weather data" error="intentionally failing" source="mock weather provider"`
		if !strings.Contains(buf.String(), wantErr) {
			t.Errorf("expected error to contain %q, got %q", wantErr, buf.String())
		}
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
				c.GeoLocation.DisableCitynameFile = false
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
				c.GeoLocation.DisableCitynameFile = true
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
				c.GeoLocation.DisableCitynameFile = true
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
				c.GeoLocation.DisableCitynameFile = true
				c.GeoLocation.DisableGPSD = true
				c.GeoLocation.DisableICHNAEA = true
			},
			shouldFail: false,
		},
		{
			name: "only cityname file",
			confFn: func(c *config.Config) {
				c.GeoLocation.DisableGeoAPI = true
				c.GeoLocation.DisableGeoIP = true
				c.GeoLocation.DisableGeolocationFile = true
				c.GeoLocation.DisableCitynameFile = false
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
				c.GeoLocation.DisableCitynameFile = true
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
				c.GeoLocation.DisableCitynameFile = true
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
			serv.geocoder = new(mockGeocoder)

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

func TestService_updateLocation(t *testing.T) {
	t.Run("different coordinates are updated", func(t *testing.T) {
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
				name:      "invalid positive latitude",
				latitude:  91.0,
				longitude: 180.0,
				wantErr:   true,
			},
			{
				name:      "invalid positive longitude",
				latitude:  90.0,
				longitude: 181.0,
				wantErr:   true,
			},
			{
				name:      "invalid positive values",
				latitude:  91.0,
				longitude: 181.0,
				wantErr:   true,
			},
			{
				name:      "invalid negative latitude",
				latitude:  -91.0,
				longitude: 180.0,
				wantErr:   true,
			},
			{
				name:      "invalid negative longitude",
				latitude:  90.0,
				longitude: -181.0,
				wantErr:   true,
			},
			{
				name:      "invalid negative values",
				latitude:  -91.0,
				longitude: -181.0,
				wantErr:   true,
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

		rtFn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
			return &stdhttp.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString("{}")),
				Header:     make(stdhttp.Header),
			}, nil
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				serv, err := testService(t, false)
				if err != nil {
					t.Fatalf("failed to create service: %s", err)
				}
				serv.output = io.Discard
				serv.geocoder = &mockGeocoder{}

				httpclient := http.New(serv.logger)
				httpclient.Transport = testhelper.MockRoundTripper{Fn: rtFn}
				weatherProv, err := openmeteo.New(httpclient, serv.logger, serv.config.Units)
				if err != nil {
					t.Fatalf("failed to create weather provider: %s", err)
				}
				serv.weatherProv = weatherProv
				err = serv.updateLocation(t.Context(), geobus.Coordinate{Lat: tc.latitude, Lon: tc.longitude})

				if tc.wantErr && err == nil {
					t.Error("expected error but got none")
				}
				if !tc.wantErr && err != nil {
					t.Errorf("unexpected error: %s", err)
				}
			})
		}
	})
	t.Run("geocoder fails", func(t *testing.T) {
		serv, err := testService(t, false)
		if err != nil {
			t.Fatalf("failed to create service: %s", err)
		}
		serv.output = io.Discard
		serv.geocoder = &mockGeocoder{shouldFail: true}
		err = serv.updateLocation(t.Context(), geobus.Coordinate{Lat: 44.4375, Lon: 26.125})
		if err == nil {
			t.Error("expected error but got none")
		}
		wantErr := `failed reverse geocode coordinates: intentionally failing`
		if !strings.Contains(err.Error(), wantErr) {
			t.Errorf("expected error to contain %q, got %q", wantErr, err)
		}
	})
}

func TestService_HandleSignals(t *testing.T) {
	t.Run("USR1 signal is handled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		serv, err := testService(t, false)
		if err != nil {
			t.Fatalf("failed to create service: %s", err)
		}
		sigChan := make(chan os.Signal, 1)
		serv.SignalSrc.Notify(sigChan, syscall.SIGUSR1, syscall.SIGUSR2)
		go func() {
			defer serv.SignalSrc.Stop(sigChan)
			serv.HandleSignals(ctx, sigChan)
		}()

		sigChan <- syscall.SIGUSR1
		time.Sleep(time.Millisecond * 100)
		serv.displayAltLock.RLock()
		defer serv.displayAltLock.RUnlock()
		if !serv.displayAltText {
			t.Errorf("expected alt mode to be enabled, got %t", serv.displayAltText)
		}
		cancel()
	})
	t.Run("USR2 signal is handled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		serv, err := testService(t, false)
		if err != nil {
			t.Fatalf("failed to create service: %s", err)
		}
		buf := &syncBuffer{buf: bytes.NewBuffer(nil)}
		serv.logger = logger.NewLogger(slog.LevelInfo, buf)
		sigChan := make(chan os.Signal, 1)
		serv.SignalSrc.Notify(sigChan, syscall.SIGUSR1, syscall.SIGUSR2)
		go func() {
			defer serv.SignalSrc.Stop(sigChan)
			serv.HandleSignals(ctx, sigChan)
		}()

		sigChan <- syscall.SIGUSR2
		time.Sleep(time.Millisecond * 100)
		wantLog := `msg="currently resolved address" address="" latitude=0 longitude=0`
		if !strings.Contains(buf.String(), wantLog) {
			t.Errorf("expected log to contain %q, got %q", wantLog, buf.String())
		}
		cancel()
		time.Sleep(time.Millisecond * 100)
	})
}

type (
	weatherProv  struct{ shouldFail bool }
	failWriter   struct{}
	mockGeocoder struct{ shouldFail bool }
	syncBuffer   struct {
		mu  sync.Mutex
		buf *bytes.Buffer
	}
)

func (f failWriter) Write([]byte) (int, error) { return 0, fmt.Errorf("failed to write") }

func (m *mockGeocoder) Name() string {
	return "mock geocoder"
}

func (m *mockGeocoder) Reverse(_ context.Context, coords geobus.Coordinate) (geocode.Address, error) {
	if m.shouldFail {
		return geocode.Address{}, errors.New("intentionally failing")
	}
	return geocode.Address{
		AddressFound: true,
		Latitude:     coords.Lat,
		Longitude:    coords.Lon,
		DisplayName:  fmt.Sprintf("Test Location %.6f,%.6f", coords.Lat, coords.Lon),
	}, nil
}

func (c *mockGeocoder) Search(_ context.Context, _ string) (geobus.Coordinate, error) {
	return geobus.Coordinate{}, errors.New("not implemented")
}

func (w *weatherProv) Name() string {
	return "mock weather provider"
}

func (w *weatherProv) GetWeather(_ context.Context, coords geobus.Coordinate) (*weather.Data, error) {
	if w.shouldFail {
		return nil, errors.New("intentionally failing")
	}
	return &weather.Data{
		GeneratedAt: time.Now(),
		Coordinates: coords,
		Current: weather.Instant{
			InstantTime: time.Now(),
			Temperature: 20.0,
		},
	}, nil
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}
