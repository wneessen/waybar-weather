// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/doppiogancio/go-nominatim/shared"
	"github.com/go-co-op/gocron/v2"
	"github.com/hectormalot/omgo"
	"github.com/maltegrosse/go-geoclue2"
)

const (
	OutputClass = "waybar-weather"
)

var (
	WeatherUpdateInterval = time.Minute * 5
	OutputInterval        = time.Second * 30
)

type outputData struct {
	Text    string `json:"text"`
	Tooltip string `json:"tooltip"`
	Class   string `json:"class"`
}

type Service struct {
	config    *config
	geoclient geoclue2.GeoclueClient
	logger    *logger
	omclient  omgo.Client
	scheduler gocron.Scheduler

	locationLock sync.RWMutex
	address      *shared.Address
	location     omgo.Location
	isDayTime    bool
	sunriseTime  time.Time
	sunsetTime   time.Time

	weatherLock  sync.RWMutex
	weatherIsSet bool
	weather      omgo.CurrentWeather
}

func New(config *config, log *logger) (*Service, error) {
	geoclient, err := RegisterGeoClue()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to register geoclue client: %s\n", err)
		os.Exit(1)
	}

	scheduler, err := gocron.NewScheduler()
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduler: %w", err)
	}

	omclient, err := omgo.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Open-Meteo client: %w", err)
	}

	return &Service{
		config:    config,
		logger:    log,
		geoclient: geoclient,
		omclient:  omclient,
		scheduler: scheduler,
	}, nil
}

func (s *Service) Run(ctx context.Context) error {
	// Start scheduled jobs
	_, err := s.scheduler.NewJob(gocron.DurationJob(WeatherUpdateInterval),
		gocron.NewTask(s.printWeather),
		gocron.WithContext(ctx),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
		gocron.WithName("weatherdata_output_job"),
	)
	if err != nil {
		return fmt.Errorf("failed to create weather data output job: %w", err)
	}

	_, err = s.scheduler.NewJob(gocron.DurationJob(OutputInterval),
		gocron.NewTask(s.fetchWeather),
		gocron.WithContext(ctx),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
		gocron.WithName("weather_update_job"),
	)
	if err != nil {
		return fmt.Errorf("failed to create weather update job: %w", err)
	}
	s.scheduler.Start()

	// Initial geolocation lookup
	if err = s.geoclient.Start(); err != nil {
		return fmt.Errorf("failed to start geoclue client: %w", err)
	}
	latitude, longitude, err := s.geoLocation()
	if err != nil {
		s.logger.Error("failed to get initial geo location", logError(err))
	}
	if err = s.updateLocation(latitude, longitude); err != nil {
		s.logger.Error("failed to update service geo location", logError(err))
	}

	// Subscribe to location updates
	go s.subscribeLocationUpdates()

	// Wait for the context to cancel
	<-ctx.Done()
	return s.scheduler.Shutdown()
}

func (s *Service) printWeather(context.Context) {
	s.locationLock.RLock()
	defer s.locationLock.RUnlock()
	s.weatherLock.RLock()
	defer s.weatherLock.RUnlock()

	if s.address == nil || !s.weatherIsSet {
		return
	}

	dayOrNight := "day"
	if !s.isDayTime {
		dayOrNight = "night"
	}

	tempUnit := "°C"
	if s.config.Units == "imperial" {
		tempUnit = "°F"
	}

	output := outputData{
		Text: fmt.Sprintf("%s %.1f%s",
			WMOWeatherIcons[s.weather.WeatherCode][dayOrNight],
			s.weather.Temperature,
			tempUnit,
		),
		Tooltip: fmt.Sprintf("Condition: %s\nLocation: %s, %s\nSunrise: %s\nSunset: %s\nLast update: %s",
			WMOWeatherCodes[s.weather.WeatherCode],
			s.address.City, s.address.Country,
			s.sunriseTime.Format("2006-01-02 15:04"),
			s.sunsetTime.Format("2006-01-02 15:04"),
			s.weather.Time.Format("2006-01-02 15:04"),
		),
		Class: OutputClass,
	}

	if err := json.NewEncoder(os.Stdout).Encode(output); err != nil {
		s.logger.Error("failed to encode weather data", logError(err))
	}
}
