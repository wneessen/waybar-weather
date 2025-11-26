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
	"strings"
	"testing"
	"testing/synctest"
	tt "text/template"

	"github.com/wneessen/waybar-weather/internal/config"
	"github.com/wneessen/waybar-weather/internal/http"
	"github.com/wneessen/waybar-weather/internal/i18n"
	"github.com/wneessen/waybar-weather/internal/logger"
	"github.com/wneessen/waybar-weather/internal/template"
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
				if tc.wantFail && err == nil {
					t.Fatal("expected service creation to fail")
				}
				if !tc.wantFail && err != nil {
					t.Fatalf("failed to create service: %s", err)
				}
				if tc.wantFail {
					return
				}
				if serv == nil {
					t.Fatal("expected service to be non-nil")
				}
				if serv.geocoder == nil {
					t.Fatal("expected geocoder to be non-nil")
				}
				name := fmt.Sprintf("geocoder cache using %s", tc.wantName)
				if serv.geocoder.Name() != name {
					t.Errorf("expected geocoder name to be %q, got %q", name, serv.geocoder.Name())
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
