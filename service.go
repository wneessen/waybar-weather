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
	"github.com/nathan-osman/go-sunrise"
	"github.com/wneessen/go-moonphase"
)

const (
	OutputClass = "waybar-weather"
	TimeFormat  = "15:04"
)

type outputData struct {
	Text    string `json:"text"`
	Tooltip string `json:"tooltip"`
	Class   string `json:"class"`
}

type DisplayData struct {
	Latitude           float64
	Longitude          float64
	Elevation          float64
	Address            shared.Address
	UpdateTime         time.Time
	WeatherDateForTime time.Time
	Temperature        float64
	WeatherCode        float64
	WindDirection      float64
	WindSpeed          float64
	IsDaytime          bool
	TempUnit           string
	SunsetTime         time.Time
	SunriseTime        time.Time
	ConditionIcon      string
	Condition          string
	Moonphase          string
	MoonphaseIcon      string
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

	weatherLock  sync.RWMutex
	weatherIsSet bool
	weather      *omgo.Forecast
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
	_, err := s.scheduler.NewJob(gocron.DurationJob(s.config.Intervals.WeatherUpdate),
		gocron.NewTask(s.printWeather),
		gocron.WithContext(ctx),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
		gocron.WithName("weatherdata_output_job"),
	)
	if err != nil {
		return fmt.Errorf("failed to create weather data output job: %w", err)
	}

	_, err = s.scheduler.NewJob(gocron.DurationJob(s.config.Intervals.Output),
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
	if !s.weatherIsSet {
		return
	}

	displayData := new(DisplayData)
	s.fillDisplayData(displayData)

	output := outputData{
		Text: fmt.Sprintf("%s %.1f%s",
			displayData.ConditionIcon,
			displayData.Temperature,
			displayData.TempUnit,
		),
		Tooltip: fmt.Sprintf("Condition: %s\nLocation: %s, %s\nSunrise: %s\nSunset: %s\nMoonphase: %s %s\nWeather information: %s\n",
			displayData.Condition,
			displayData.Address.City, displayData.Address.Country,
			displayData.SunriseTime.Format(TimeFormat),
			displayData.SunsetTime.Format(TimeFormat),
			displayData.Moonphase,
			displayData.MoonphaseIcon,
			displayData.WeatherDateForTime.Format(TimeFormat),
		),
		Class: OutputClass,
	}

	if err := json.NewEncoder(os.Stdout).Encode(output); err != nil {
		s.logger.Error("failed to encode weather data", logError(err))
	}
}

func (s *Service) fillDisplayData(target *DisplayData) {
	s.locationLock.RLock()
	defer s.locationLock.RUnlock()
	s.weatherLock.RLock()
	defer s.weatherLock.RUnlock()

	// We need valid weather data to fill the display data
	if s.weather == nil {
		return
	}

	// Coordinate data
	target.Latitude = s.weather.Latitude
	target.Longitude = s.weather.Longitude
	target.Elevation = s.weather.Elevation
	if s.address != nil {
		target.Address = *s.address
	}

	// Moon phase
	m := moonphase.New(time.Now())
	target.Moonphase = m.PhaseName()
	target.MoonphaseIcon = moonPhases[target.Moonphase]

	// Fill weather data
	now := time.Now()
	switch s.config.WeatherMode {
	case "current":
		target.SunriseTime, target.SunsetTime = sunrise.SunriseSunset(s.weather.Latitude, s.weather.Longitude, now.Year(),
			now.Month(), now.Day())
		target.IsDaytime = false
		if now.After(target.SunriseTime) && now.Before(target.SunsetTime) {
			target.IsDaytime = true
		}

		target.UpdateTime = s.weather.CurrentWeather.Time.Time
		target.Temperature = s.weather.CurrentWeather.Temperature
		target.WeatherCode = s.weather.CurrentWeather.WeatherCode
		target.WindDirection = s.weather.CurrentWeather.WindDirection
		target.WindSpeed = s.weather.CurrentWeather.WindSpeed
		target.TempUnit = s.weather.HourlyUnits["temperature_2m"]
		target.WeatherDateForTime = s.weather.CurrentWeather.Time.Time
		target.ConditionIcon = wmoWeatherIcons[target.WeatherCode][target.IsDaytime]
		target.Condition = WMOWeatherCodes[target.WeatherCode]
	case "forecast":
		fcastHours := time.Duration(s.config.ForecastHours) * time.Hour //nolint:gosec
		fcastTime := now.Add(fcastHours).Truncate(time.Hour)
		idx := -1
		for i, t := range s.weather.HourlyTimes {
			if t.Equal(fcastTime) {
				idx = i
				break
			}
		}
		if idx == -1 {
			break
		}

		target.SunriseTime, target.SunsetTime = sunrise.SunriseSunset(s.weather.Latitude, s.weather.Longitude,
			fcastTime.Year(), fcastTime.Month(), fcastTime.Day())
		target.IsDaytime = false
		if s.weather.HourlyUnits["is_day"] == "1" {
			target.IsDaytime = true
		}

		target.UpdateTime = s.weather.CurrentWeather.Time.Time
		target.Temperature = s.weather.HourlyMetrics["temperature_2m"][idx]
		target.WeatherCode = s.weather.HourlyMetrics["weather_code"][idx]
		target.WindDirection = s.weather.HourlyMetrics["wind_direction_10m"][idx]
		target.WindSpeed = s.weather.HourlyMetrics["wind_speed_10m"][idx]
		target.TempUnit = s.weather.HourlyUnits["temperature_2m"]
		target.WeatherDateForTime = fcastTime
		target.ConditionIcon = wmoWeatherIcons[target.WeatherCode][target.IsDaytime]
		target.Condition = WMOWeatherCodes[target.WeatherCode]
	}
}
