// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/vorlif/spreak"

	"github.com/wneessen/waybar-weather/internal/config"
	"github.com/wneessen/waybar-weather/internal/geobus"
	"github.com/wneessen/waybar-weather/internal/geocode"
	"github.com/wneessen/waybar-weather/internal/job"
	"github.com/wneessen/waybar-weather/internal/logger"
	"github.com/wneessen/waybar-weather/internal/presenter"
	"github.com/wneessen/waybar-weather/internal/weather"
)

const (
	OutputClass  = "waybar-weather"
	SubID        = "location-update"
	cacheHitTTL  = 1 * time.Hour
	cacheMissTTL = 10 * time.Minute
)

type outputData struct {
	Text    string `json:"text"`
	Tooltip string `json:"tooltip"`
	Class   string `json:"class"`
}

type Service struct {
	SignalSrc signalSource

	config      *config.Config
	geobus      *geobus.GeoBus
	logger      *logger.Logger
	geocoder    geocode.Geocoder
	weatherProv weather.Provider
	output      io.Writer
	jobs        []*job.Job
	presenter   *presenter.Presenter
	t           *spreak.Localizer

	locationLock  sync.RWMutex
	address       geocode.Address
	locationIsSet bool
	location      geobus.Coordinate

	weatherLock  sync.RWMutex
	weatherIsSet bool
	weather      *weather.Data

	displayAltLock sync.RWMutex
	displayAltText bool
}

func New(conf *config.Config, log *logger.Logger, t *spreak.Localizer) (*Service, error) {
	pres, err := presenter.New(conf, t)
	if err != nil {
		return nil, fmt.Errorf("failed to create presenter: %w", err)
	}

	bus, err := geobus.New(log)
	if err != nil {
		return nil, fmt.Errorf("failed to create geobus: %w", err)
	}

	service := &Service{
		SignalSrc: stdLibSignalSource{},

		config:         conf,
		geobus:         bus,
		logger:         log,
		output:         os.Stdout,
		presenter:      pres,
		t:              t,
		displayAltText: false,
	}

	// Schedule jobs
	outputJob := job.New(service.config.Intervals.Output, service.printWeather)
	// weatherUpdateJob := job.New(service.config.Intervals.WeatherUpdate, service.fetchWeather)
	service.jobs = append(service.jobs, outputJob)

	return service, nil
}

func (s *Service) Run(ctx context.Context) (err error) {
	// Start scheduled jobs as go routines
	for _, j := range s.jobs {
		if j == nil {
			continue
		}
		go j.Start(ctx)
	}

	// Select the geocode provider for the address lookup
	geocodeProvider, err := s.selectGeocodeProvider(s.config, s.logger, s.t.Language())
	if err != nil {
		return fmt.Errorf("failed to create geocode provider: %w", err)
	}
	s.geocoder = geocodeProvider

	// Select the geobus providers and track them in the geobus
	geobusProvider, err := s.selectGeobusProviders()
	if err != nil {
		return fmt.Errorf("failed to create geobus orchestrator: %w", err)
	}
	geobus.TrackProviders(ctx, s.geobus, SubID, geobusProvider...)

	// Select the weather provider
	weatherProv, err := s.selectWeatherProvider()
	if err != nil {
		return fmt.Errorf("failed to create weather provider: %w", err)
	}
	s.weatherProv = weatherProv

	// Subscribe to geolocation updates from the geobus
	sub, unsub := s.geobus.Subscribe(SubID, 1)
	go s.processLocationUpdates(ctx, sub)

	// Detect sleep/wake events and update the weather
	go s.monitorSleepResume(ctx)

	// Wait for the context to cancel
	<-ctx.Done()
	if unsub != nil {
		unsub()
	}
	return nil
}

func (s *Service) fetchWeather(ctx context.Context) {
	s.weatherLock.Lock()
	defer s.weatherLock.Unlock()

	data, err := s.weatherProv.GetWeather(ctx, s.location)
	if err != nil {
		s.logger.Error("failed to fetch weather data", logger.Err(err),
			slog.String("source", s.weatherProv.Name()))
	}
	s.weather = data
	s.weatherIsSet = true

	s.logger.Debug("weather data fetched successfully")
}

// printWeather outputs the current weather data to stdout if available and renders it using predefined templates.
func (s *Service) printWeather(context.Context) {
	if !s.weatherIsSet {
		return
	}

	s.displayAltLock.RLock()
	displayAltText := s.displayAltText
	s.displayAltLock.RUnlock()

	tplCtx := s.presenter.BuildContext(s.address, s.weather, time.Now(), time.Now(), "", "")
	text, alttext, tooltip, err := s.presenter.Render(tplCtx)
	if err != nil {
		s.logger.Error("failed to render weather template", logger.Err(err))
	}

	displayText := text
	if displayAltText {
		displayText = alttext
	}

	output := outputData{
		Text:    displayText,
		Tooltip: tooltip,
		Class:   OutputClass,
	}

	if err = json.NewEncoder(s.output).Encode(output); err != nil {
		s.logger.Error("failed to encode weather data", logger.Err(err))
	}
}

// fillDisplayData populates the provided DisplayData object with details based on current or
// forecasted weather information. It locks relevant data structures to ensure safe concurrent
// access and conditionally fills fields based on the mode.
func (s *Service) fillDisplayData() {
	/*
		s.locationLock.RLock()
		defer s.locationLock.RUnlock()
		s.weatherLock.RLock()
		defer s.weatherLock.RUnlock()

		// The target must not be nil
		if target == nil {
			return
		}

		// We need valid weather data to fill the display data
		if s.weather == nil {
			s.logger.Debug("no weather data available yet, geo location might not have returned a location yet")
			return
		}

		// Coordinates and address data
		// target.Latitude = s.weather.Latitude
		// target.Longitude = s.weather.Longitude
		// target.Elevation = s.weather.Elevation
		target.Address = s.address

		// Moon phase
		m := moonphase.New(time.Now())
		target.Moonphase = m.PhaseName()
		target.MoonphaseIcon = presenter.MoonPhaseIcon[target.Moonphase]

		// Generel weather data
		now := time.Now()
		nowHourUTC := now.UTC().Truncate(time.Hour)
		nowIdx := s.weatherIndexByTime(nowHourUTC)
		target.UpdateTime = s.weather.CurrentWeather.Time.Time
		target.TempUnit = s.weather.HourlyUnits["temperature_2m"]
		target.PressureUnit = s.weather.HourlyUnits["pressure_msl"]
		sunriseTimeUTC, sunsetTimeUTC := sunrise.SunriseSunset(s.weather.Latitude, s.weather.Longitude, now.Year(),
			now.Month(), now.Day())
		target.SunriseTime, target.SunsetTime = sunriseTimeUTC.In(now.Location()), sunsetTimeUTC.In(now.Location())
		target.Current.IsDaytime = false
		if now.After(target.SunriseTime) && now.Before(target.SunsetTime) {
			target.Current.IsDaytime = true
		}

		// Current weather data
		target.Current.Temperature = s.weather.CurrentWeather.Temperature
		target.Current.WeatherCode = s.weather.CurrentWeather.WeatherCode
		target.Current.WindDirection = s.weather.CurrentWeather.WindDirection
		target.Current.WindSpeed = s.weather.CurrentWeather.WindSpeed
		target.Current.WeatherDateForTime = s.weather.CurrentWeather.Time.Time
		target.Current.ConditionIcon = presenter.WMOWeatherIcons[int(target.Current.WeatherCode)][target.Current.IsDaytime]
		target.Current.Condition = s.t.Get(presenter.WMOWeatherCodes[int(target.Current.WeatherCode)])
		if nowIdx != -1 {
			target.Current.ApparentTemperature = s.weather.HourlyMetrics["apparent_temperature"][nowIdx]
			target.Current.Humidity = s.weather.HourlyMetrics["relative_humidity_2m"][nowIdx]
			target.Current.PressureMSL = s.weather.HourlyMetrics["pressure_msl"][nowIdx]
		}

		// Forecast weather data
		fcastHours := time.Duration(s.config.Weather.ForecastHours) * time.Hour //nolint:gosec
		fcastTime := now.Add(fcastHours).Truncate(time.Hour)
		fcastTimeUTC := now.Add(fcastHours).UTC().Truncate(time.Hour)
		fcastIdx := s.weatherIndexByTime(fcastTimeUTC)
		if fcastIdx != -1 {
			target.Forecast.WeatherDateForTime = fcastTime
			target.Forecast.IsDaytime = false
			if s.weather.HourlyMetrics["is_day"][fcastIdx] == 1 {
				target.Forecast.IsDaytime = true
			}
			target.Forecast.Temperature = s.weather.HourlyMetrics["temperature_2m"][fcastIdx]
			target.Forecast.ApparentTemperature = s.weather.HourlyMetrics["apparent_temperature"][fcastIdx]
			target.Forecast.Humidity = s.weather.HourlyMetrics["relative_humidity_2m"][fcastIdx]
			target.Forecast.PressureMSL = s.weather.HourlyMetrics["pressure_msl"][fcastIdx]
			target.Forecast.WeatherCode = s.weather.HourlyMetrics["weather_code"][fcastIdx]
			target.Forecast.WindDirection = s.weather.HourlyMetrics["wind_direction_10m"][fcastIdx]
			target.Forecast.WindSpeed = s.weather.HourlyMetrics["wind_speed_10m"][fcastIdx]
			target.Forecast.ConditionIcon = presenter.WMOWeatherIcons[int(target.Forecast.WeatherCode)][target.Forecast.IsDaytime]
			target.Forecast.Condition = s.t.Get(presenter.WMOWeatherCodes[int(target.Forecast.WeatherCode)])
		} else {
			target.Forecast = target.Current
		}

	*/
}

// updateLocation updates the service's location and address based on provided latitude and longitude.
// It locks the location for thread-safe updates and retrieves the address information using reverse geocoding.
// If valid coordinates are not provided, the update is skipped. The method also triggers all scheduled jobs.
func (s *Service) updateLocation(ctx context.Context, coords geobus.Coordinate) error {
	if !coords.Valid() {
		return fmt.Errorf("invalid coordinates: %f, %f", coords.Lat, coords.Lon)
	}

	address, err := s.geocoder.Reverse(ctx, coords)
	if err != nil {
		return fmt.Errorf("failed reverse geocode coordinates: %w", err)
	}

	s.locationLock.Lock()
	s.location = coords
	if address.AddressFound {
		s.address = address
	}
	s.locationIsSet = true
	s.locationLock.Unlock()
	s.logger.Debug("address successfully resolved", slog.Any("address", s.address.DisplayName),
		slog.Any("coordinates", s.location), slog.String("source", s.geocoder.Name()),
		slog.Bool("cache_hit", address.CacheHit))

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
				slog.Float64("lat", r.Lat), slog.Float64("lon", r.Lon),
				slog.Float64("accuracy", r.AccuracyMeters), slog.String("source", r.Source))
			if err := s.updateLocation(ctx, geobus.Coordinate{Lat: r.Lat, Lon: r.Lon}); err != nil {
				s.logger.Error("failed to apply geo update", logger.Err(err), slog.String("source", r.Source))
			}
		}
	}
}

/*
func (s *Service) weatherIndexByTime(atTime time.Time) int {
	for i, t := range s.weather.HourlyTimes {
		if t.Equal(atTime) {
			return i
		}
	}
	return -1
}


*/
