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

	"github.com/kkyr/fig"
)

const configEnv = "WAYBARWEATHER"

// config represents the application's configuration structure.
type config struct {
	Units    string     `fig:"units" default:"metric"`
	Locale   string     `fig:"locale"`
	LogLevel slog.Level `fig:"loglevel" default:"0"`
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
