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
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGKILL,
		syscall.SIGABRT, os.Interrupt)
	defer cancel()

	// Initialize logger
	log := newLogger(slog.LevelError)

	// Read config
	confPath := flag.String("config", "", "path to the config file")
	flag.Parse()
	conf, err := newConfig()
	if err != nil {
		log.Error("failed to load config", logError(err))
		os.Exit(1)
	}
	if *confPath != "" {
		file := filepath.Base(*confPath)
		path := filepath.Dir(*confPath)
		conf, err = newConfigFromFile(path, file)
		if err != nil {
			log.Error("failed to load config from file", logError(err))
			os.Exit(1)
		}
	}
	log = newLogger(conf.LogLevel)

	// We need a running geoclue agent
	isRunning, err := geoClueAgentIsRunning(ctx)
	if err != nil {
		log.Error("failed to check if geoclue agent is running", logError(err))
		os.Exit(1)
	}
	if !isRunning {
		log.Error("required geoclue agent is not running, shutting down")
		os.Exit(1)
	}

	// Initialize the service
	service, err := New(conf, log)
	if err != nil {
		log.Error("failed to initialize waybar-weather service", logError(err))
		os.Exit(1)
	}

	// Start the service loop
	log.Info("starting waybar-weather service")
	if err = service.Run(ctx); err != nil {
		log.Error("failed to start waybar-weather service", logError(err))
	}
	log.Info("shutting down waybar-weather service")
}
