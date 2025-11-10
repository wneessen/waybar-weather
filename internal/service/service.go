// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	nominatim "github.com/doppiogancio/go-nominatim"
	"github.com/doppiogancio/go-nominatim/shared"
	"github.com/go-co-op/gocron/v2"
	"github.com/hectormalot/omgo"
	"github.com/nathan-osman/go-sunrise"
	"github.com/wneessen/go-moonphase"

	"app/internal/config"
	"app/internal/geobus"
	"app/internal/geobus/provider/geoapi"
	"app/internal/geobus/provider/geoip"
	"app/internal/geobus/provider/geolocation_file"
	"app/internal/geobus/provider/ichnaea"
	"app/internal/http"
	"app/internal/logger"
	"app/internal/template"
)

const (
	OutputClass = "waybar-weather"
	DesktopID   = "waybar-weather"
)

type outputData struct {
	Text    string `json:"text"`
	Tooltip string `json:"tooltip"`
	Class   string `json:"class"`
}

type Service struct {
	config       *config.Config
	geobus       *geobus.GeoBus
	logger       *logger.Logger
	omclient     omgo.Client
	orchestrator *geobus.Orchestrator
	scheduler    gocron.Scheduler
	templates    *template.Templates

	locationLock sync.RWMutex
	address      *shared.Address
	location     omgo.Location

	weatherLock  sync.RWMutex
	weatherIsSet bool
	weather      *omgo.Forecast
}

func New(conf *config.Config, log *logger.Logger) (*Service, error) {
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduler: %w", err)
	}

	omclient, err := omgo.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Open-Meteo client: %w", err)
	}

	tpls, err := template.NewTemplate(conf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	service := &Service{
		config:    conf,
		geobus:    geobus.New(log),
		logger:    log,
		omclient:  omclient,
		scheduler: scheduler,
		templates: tpls,
	}
	return service, nil
}

func (s *Service) Run(ctx context.Context) error {
	// Start scheduled jobs
	if err := s.createScheduledJob(ctx, s.config.Intervals.Output, s.printWeather,
		"weatherdata_output_job"); err != nil {
		return err
	}
	if err := s.createScheduledJob(ctx, s.config.Intervals.WeatherUpdate, s.fetchWeather,
		"weather_update_job"); err != nil {
		return err
	}
	s.scheduler.Start()

	// Create the orchestrator
	s.orchestrator = s.createOrchestrator()

	// Subscribe to geolocation updates from the geobus
	sub, unsub := s.geobus.Subscribe(DesktopID, 32)
	go s.processLocationUpdates(ctx, sub)
	go s.orchestrator.Track(ctx, DesktopID)

	// Wait for the context to cancel
	<-ctx.Done()
	if unsub != nil {
		unsub()
	}
	return s.scheduler.Shutdown()
}

func (s *Service) createOrchestrator() *geobus.Orchestrator {
	httpClient := http.New(s.logger)
	var provider []geobus.Provider

	if !s.config.GeoLocation.DisableGeolocationFile {
		provider = append(provider, geolocation_file.NewGeolocationFileProvider(s.config.GeoLocation.File))
	}

	if !s.config.GeoLocation.DisableGeoIP {
		provider = append(provider, geoip.NewGeolocationGeoIPProvider(httpClient))
	}

	if !s.config.GeoLocation.DisableGeoAPI {
		provider = append(provider, geoapi.NewGeolocationGeoAPIProvider(httpClient))
	}

	if !s.config.GeoLocation.DisableICHNAEA {
		mls, err := ichnaea.NewGeolocationICHNAEAProvider(httpClient)
		if err != nil {
			s.logger.Error("failed to create ICHNAEA provider", logger.Err(err))
		} else {
			provider = append(provider, mls)
		}
	}

	return s.geobus.NewOrchestrator(provider)
}

func (s *Service) createScheduledJob(ctx context.Context, interval time.Duration, task func(context.Context),
	jobName string,
) error {
	_, err := s.scheduler.NewJob(
		gocron.DurationJob(interval),
		gocron.NewTask(task),
		gocron.WithContext(ctx),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
		gocron.WithName(jobName),
	)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", jobName, err)
	}
	return nil
}

// printWeather outputs the current weather data to stdout if available and renders it using predefined templates.
func (s *Service) printWeather(context.Context) {
	if !s.weatherIsSet {
		return
	}

	displayData := new(template.DisplayData)
	s.fillDisplayData(displayData)

	textBuf := bytes.NewBuffer(nil)
	if err := s.templates.Text.Execute(textBuf, displayData); err != nil {
		s.logger.Error("failed to render text template", logger.Err(err))
		return
	}
	tooltipBuf := bytes.NewBuffer(nil)
	if err := s.templates.Tooltip.Execute(tooltipBuf, displayData); err != nil {
		s.logger.Error("failed to render tooltip template", logger.Err(err))
		return
	}

	output := outputData{
		Text:    textBuf.String(),
		Tooltip: tooltipBuf.String(),
		Class:   OutputClass,
	}

	if err := json.NewEncoder(os.Stdout).Encode(output); err != nil {
		s.logger.Error("failed to encode weather data", logger.Err(err))
	}
}

// fillDisplayData populates the provided DisplayData object with details based on current or
// forecasted weather information. It locks relevant data structures to ensure safe concurrent
// access and conditionally fills fields based on the mode.
func (s *Service) fillDisplayData(target *template.DisplayData) {
	s.locationLock.RLock()
	defer s.locationLock.RUnlock()
	s.weatherLock.RLock()
	defer s.weatherLock.RUnlock()

	// We need valid weather data to fill the display data
	if s.weather == nil {
		s.logger.Debug("no weather data available yet, geo location might not have returned a location yet")
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
	target.MoonphaseIcon = MoonPhases[target.Moonphase]

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
		target.ConditionIcon = WMOWeatherIcons[target.WeatherCode][target.IsDaytime]
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
		target.ConditionIcon = WMOWeatherIcons[target.WeatherCode][target.IsDaytime]
		target.Condition = WMOWeatherCodes[target.WeatherCode]
	}
}

// updateLocation updates the service's location and address based on provided latitude and longitude.
// It locks the location for thread-safe updates and retrieves the address information using reverse geocoding.
// If valid coordinates are not provided, the update is skipped. The method also triggers all scheduled jobs.
func (s *Service) updateLocation(ctx context.Context, latitude, longitude float64) error {
	if latitude <= 0 || longitude <= 0 {
		s.logger.Debug("coordinates empty, skipping service geo location update")
		return nil
	}

	address, err := nominatim.ReverseGeocode(latitude, longitude, s.config.Locale)
	if err != nil {
		return fmt.Errorf("failed reverse geocode coordinates: %w", err)
	}
	location, err := omgo.NewLocation(latitude, longitude)
	if err != nil {
		return fmt.Errorf("failed create Open-Meteo location from coordinates: %w", err)
	}

	s.locationLock.Lock()
	s.address = address
	s.location = location
	s.locationLock.Unlock()
	s.logger.Debug("address successfully resolved", slog.Any("address", s.address.DisplayName),
		slog.Any("location", s.location))

	s.fetchWeather(ctx)
	s.printWeather(ctx)

	return nil
}

// processLocationUpdates subscribes to geolocation updates, processes location data, and updates the
// service state accordingly.
func (s *Service) processLocationUpdates(ctx context.Context, sub <-chan geobus.Result) {
	for {
		select {
		case <-ctx.Done():
			return
		case r, ok := <-sub:
			if !ok {
				return
			}
			s.logger.Debug("received geolocation update",
				slog.Float64("lat", r.Lat), slog.Float64("lon", r.Lon), slog.String("source", r.Source))
			if err := s.updateLocation(ctx, r.Lat, r.Lon); err != nil {
				s.logger.Error("failed to apply geo update", logger.Err(err), slog.String("source", r.Source))
			}
		}
	}
}
