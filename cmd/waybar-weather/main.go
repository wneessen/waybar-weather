// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

//go:build linux

// Package main implements the waybar-weather service.
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/wneessen/waybar-weather/internal/config"
	"github.com/wneessen/waybar-weather/internal/i18n"
	"github.com/wneessen/waybar-weather/internal/logger"
	"github.com/wneessen/waybar-weather/internal/service"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGKILL,
		syscall.SIGABRT, os.Interrupt)
	defer cancel()

	// Initialize Logger
	log := logger.NewLogger(slog.LevelError)

	// Read config
	confRead := false
	confPath := flag.String("config", "", "path to the config file")
	flag.Parse()

	// Read default config
	conf, err := config.New()
	if err != nil {
		log.Error("failed to load config", logger.Err(err))
		os.Exit(1)
	}

	// If config file was specified, read it
	if *confPath != "" {
		file := filepath.Base(*confPath)
		path := filepath.Dir(*confPath)
		conf, err = config.NewFromFile(path, file)
		if err != nil {
			log.Error("failed to load config from file", logger.Err(err))
			os.Exit(1)
		}
		confRead = true
	}

	// Check if we have a config file in the default location
	if path, file := findConfigFile(); !confRead && (path != "" && file != "") {
		conf, err = config.NewFromFile(path, file)
		if err != nil {
			log.Error("failed to load config from file", logger.Err(err))
			os.Exit(1)
		}
	}

	log = logger.NewLogger(conf.LogLevel)
	t, err := i18n.New(conf.Locale)
	if err != nil {
		log.Error("failed to initialize localizer", logger.Err(err))
		os.Exit(1)
	}

	// Initialize the service
	serv, err := service.New(conf, log, t)
	if err != nil {
		log.Error("failed to initialize waybar-weather service", logger.Err(err))
		os.Exit(1)
	}

	// Start the service loop
	log.Info(t.Get("starting waybar-weather service"), slog.String("version", version),
		slog.String("commit", commit), slog.String("date", date))
	if err = serv.Run(ctx); err != nil {
		log.Error(t.Get("failed to start waybar-weather service"), logger.Err(err))
	}
	log.Info(t.Get("shutting down waybar-weather service"))
}

func findConfigFile() (string, string) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return "", ""
	}
	exts := []string{"toml", "yaml", "yml", "json"}
	for _, ext := range exts {
		path := filepath.Join(homedir, ".config", "waybar-weather", "config."+ext)
		if _, err = os.Stat(path); err == nil {
			return filepath.Dir(path), filepath.Base(path)
		}
	}
	return "", ""
}
