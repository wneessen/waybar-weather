// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package template

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/wneessen/waybar-weather/internal/config"
	"github.com/wneessen/waybar-weather/internal/i18n"
)

const defaultLang = "en"

func TestNewTemplate(t *testing.T) {
	t.Run("new template succeeds", func(t *testing.T) {
		conf, err := config.New()
		if err != nil {
			t.Fatalf("failed to create config: %s", err)
		}
		loc, err := i18n.New(defaultLang)
		if err != nil {
			t.Fatalf("failed to create localizer: %s", err)
		}
		tpl, err := New(conf, loc)
		if err != nil {
			t.Fatalf("failed to create template: %s", err)
		}
		if tpl == nil {
			t.Fatal("expected template to be non-nil")
		}
	})
	t.Run("rendering template succeeds", func(t *testing.T) {
		conf, err := config.New()
		if err != nil {
			t.Fatalf("failed to create config: %s", err)
		}
		loc, err := i18n.New(defaultLang)
		if err != nil {
			t.Fatalf("failed to create localizer: %s", err)
		}
		conf.Templates.Text = "{{ .Data }}"
		tpl, err := New(conf, loc)
		if err != nil {
			t.Fatalf("failed to create template: %s", err)
		}

		expect := "test"
		data := map[string]string{
			"Data": expect,
		}
		buf := bytes.NewBuffer(nil)
		if err = tpl.Text.Execute(buf, data); err != nil {
			t.Errorf("failed to render template: %s", err)
		}
		if buf.String() != "test" {
			t.Errorf("expected rendered template to be %q, got %q", expect, buf.String())
		}
	})

	tests := []struct {
		name      string
		configure func(*config.Config)
	}{
		{
			name: "parsing text template fails",
			configure: func(c *config.Config) {
				c.Templates.Text = "{{ .Data }"
			},
		},
		{
			name: "parsing tooltip template fails",
			configure: func(c *config.Config) {
				c.Templates.Tooltip = "{{ .Data }"
			},
		},
		{
			name: "parsing alt text template fails",
			configure: func(c *config.Config) {
				c.Templates.AltText = "{{ .Data }"
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conf, err := config.New()
			if err != nil {
				t.Fatalf("failed to create config: %s", err)
			}
			loc, err := i18n.New(defaultLang)
			if err != nil {
				t.Fatalf("failed to create localizer: %s", err)
			}

			tc.configure(conf)
			_, err = New(conf, loc)
			if err == nil {
				t.Fatal("expected template parsing to fail, but didn't")
			}
		})
	}

	t.Run("localizer function translates correctly", func(t *testing.T) {
		has := "humidity"
		want := "Luftfeuchtigkeit"
		lang := "de"

		conf, err := config.New()
		if err != nil {
			t.Fatalf("failed to create config: %s", err)
		}
		loc, err := i18n.New(lang)
		if err != nil {
			t.Fatalf("failed to create localizer: %s", err)
		}

		conf.Templates.Text = "{{loc .Data}}"
		tpl, err := New(conf, loc)
		if err != nil {
			t.Fatalf("failed to create template: %s", err)
		}

		data := map[string]string{
			"Data": has,
		}
		buf := bytes.NewBuffer(nil)
		if err = tpl.Text.Execute(buf, data); err != nil {
			t.Errorf("failed to render template: %s", err)
		}
		if buf.String() != want {
			t.Errorf("expected rendered template to be %q, got %q", want, buf.String())
		}
	})
	t.Run("localizer function returns original value on unsupported translation", func(t *testing.T) {
		has := "invalid-unknown"
		lang := "de"

		conf, err := config.New()
		if err != nil {
			t.Fatalf("failed to create config: %s", err)
		}
		loc, err := i18n.New(lang)
		if err != nil {
			t.Fatalf("failed to create localizer: %s", err)
		}

		conf.Templates.Text = "{{loc .Data}}"
		tpl, err := New(conf, loc)
		if err != nil {
			t.Fatalf("failed to create template: %s", err)
		}

		data := map[string]string{
			"Data": has,
		}
		buf := bytes.NewBuffer(nil)
		if err = tpl.Text.Execute(buf, data); err != nil {
			t.Errorf("failed to render template: %s", err)
		}
		if buf.String() != has {
			t.Errorf("expected rendered template to be %q, got %q", has, buf.String())
		}
	})
	t.Run("localized time function returns correct format", func(t *testing.T) {
		conf, err := config.New()
		if err != nil {
			t.Fatalf("failed to create config: %s", err)
		}
		conf.Templates.Text = "{{localizedTime .Data}}"
		wantTime := time.Date(2025, time.January, 1, 16, 56, 0, 0, time.UTC)

		tests := []struct {
			name string
			lang string
			want string
		}{
			{"english 12h", "en", "4:56 p.m."},
			{"german 24h", "de", "16:56"},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				loc, err := i18n.New(tc.lang)
				if err != nil {
					t.Fatalf("failed to create localizer: %s", err)
				}
				tpl, err := New(conf, loc)
				if err != nil {
					t.Fatalf("failed to create template: %s", err)
				}

				data := map[string]time.Time{
					"Data": wantTime,
				}
				buf := bytes.NewBuffer(nil)
				if err = tpl.Text.Execute(buf, data); err != nil {
					t.Errorf("failed to render template: %s", err)
				}
				if !strings.EqualFold(buf.String(), tc.want) {
					t.Errorf("expected rendered template to be %q, got %q", tc.want, buf.String())
				}
			})
		}
	})
}
