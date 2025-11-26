// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package service

import (
	"fmt"
	"strings"
	"testing"

	"github.com/wneessen/waybar-weather/internal/config"
	"github.com/wneessen/waybar-weather/internal/http"
	"github.com/wneessen/waybar-weather/internal/i18n"
	"github.com/wneessen/waybar-weather/internal/logger"
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

func testService(_ *testing.T, nilLogger bool) (*Service, error) {
	conf, err := config.New()
	if err != nil {
		return nil, err
	}

	var log *logger.Logger
	if !nilLogger {
		log = logger.New(conf.LogLevel)
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
