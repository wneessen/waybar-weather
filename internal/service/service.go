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
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/vorlif/spreak"

	"github.com/wneessen/waybar-weather/internal/config"
	"github.com/wneessen/waybar-weather/internal/geobus"
	"github.com/wneessen/waybar-weather/internal/geobus/provider/geoapi"
	"github.com/wneessen/waybar-weather/internal/geobus/provider/geoip"
	"github.com/wneessen/waybar-weather/internal/geobus/provider/geolocation_file"
	"github.com/wneessen/waybar-weather/internal/geobus/provider/gpsd"
	"github.com/wneessen/waybar-weather/internal/geobus/provider/ichnaea"
	"github.com/wneessen/waybar-weather/internal/geocode"
	"github.com/wneessen/waybar-weather/internal/geocode/provider/opencage"
	nominatim "github.com/wneessen/waybar-weather/internal/geocode/provider/osm-nominatim"
	"github.com/wneessen/waybar-weather/internal/http"
	"github.com/wneessen/waybar-weather/internal/logger"
	"github.com/wneessen/waybar-weather/internal/template"

	"github.com/go-co-op/gocron/v2"
	"github.com/hectormalot/omgo"
	"github.com/nathan-osman/go-sunrise"
	"github.com/wneessen/go-moonphase"
)

const (
	OutputClass  = "waybar-weather"
	DesktopID    = "waybar-weather"
	cacheHitTTL  = 1 * time.Hour
	cacheMissTTL = 10 * time.Minute
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
	geocoder     geocode.Geocoder
	omclient     omgo.Client
	orchestrator *geobus.Orchestrator
	scheduler    gocron.Scheduler
	templates    *template.Templates
	t            *spreak.Localizer

	locationLock  sync.RWMutex
	address       geocode.Address
	locationIsSet bool
	location      omgo.Location

	weatherLock  sync.RWMutex
	weatherIsSet bool
	weather      *omgo.Forecast

	displayAltLock sync.RWMutex
	displayAltText bool
}

func New(conf *config.Config, log *logger.Logger, t *spreak.Localizer) (*Service, error) {
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduler: %w", err)
	}

	omclient, err := omgo.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Open-Meteo client: %w", err)
	}

	tpls, err := template.New(conf, t)
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	var geocoder geocode.Geocoder
	switch strings.ToLower(conf.GeoCoder.Provider) {
	case "nominatim":
		geocoder = geocode.NewCachedGeocoder(nominatim.New(http.New(log), t.Language()), cacheHitTTL, cacheMissTTL)
	case "opencage":
		if conf.GeoCoder.APIKey == "" {
			return nil, fmt.Errorf("opencage geocoder requires an API key")
		}
		geocoder = geocode.NewCachedGeocoder(opencage.New(http.New(log), t.Language(), conf.GeoCoder.APIKey),
			cacheHitTTL, cacheMissTTL)
	default:
		return nil, fmt.Errorf("unsupported geocoder type: %s", conf.GeoCoder.Provider)
	}

	service := &Service{
		config:         conf,
		geocoder:       geocoder,
		geobus:         geobus.New(log),
		logger:         log,
		omclient:       omclient,
		scheduler:      scheduler,
		templates:      tpls,
		t:              t,
		displayAltText: false,
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

	// Validate that the templates can be rendered
	if err := s.templates.Text.Execute(bytes.NewBuffer(nil), template.DisplayData{}); err != nil {
		return fmt.Errorf("failed to render text template: %w", err)
	}
	if err := s.templates.AltText.Execute(bytes.NewBuffer(nil), template.DisplayData{}); err != nil {
		return fmt.Errorf("failed to render alt text template: %w", err)
	}
	if err := s.templates.Tooltip.Execute(bytes.NewBuffer(nil), template.DisplayData{}); err != nil {
		return fmt.Errorf("failed to render tooltip template: %w", err)
	}

	// Create the orchestrator
	s.orchestrator = s.createOrchestrator()

	// Subscribe to geolocation updates from the geobus
	sub, unsub := s.geobus.Subscribe(DesktopID, 32)
	go s.processLocationUpdates(ctx, sub)
	go s.orchestrator.Track(ctx, DesktopID)

	// Set up signal handler for SIGUSR1 to toggle alt text display
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGUSR1)
	go s.handleAltTextToggleSignal(ctx, sigChan)

	// Detect sleep/wake events and update the weather
	go s.monitorSleepResume(ctx)

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

	if !s.config.GeoLocation.DisableGPSD {
		provider = append(provider, gpsd.NewGeolocationGPSDProvider())
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
	if len(provider) == 0 {
		s.logger.Error(s.t.Get("no geolocation providers enabled, will not be able to fetch weather data " + "" +
			"due to missing location"))
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

	s.displayAltLock.RLock()
	displayAltText := s.displayAltText
	s.displayAltLock.RUnlock()

	displayData := new(template.DisplayData)
	s.fillDisplayData(displayData)

	textBuf := bytes.NewBuffer(nil)
	if err := s.templates.Text.Execute(textBuf, displayData); err != nil {
		s.logger.Error("failed to render text template", logger.Err(err))
		return
	}

	altTextBuf := bytes.NewBuffer(nil)
	if err := s.templates.AltText.Execute(altTextBuf, displayData); err != nil {
		s.logger.Error("failed to render alt text template", logger.Err(err))
		return
	}

	tooltipBuf := bytes.NewBuffer(nil)
	if err := s.templates.Tooltip.Execute(tooltipBuf, displayData); err != nil {
		s.logger.Error("failed to render tooltip template", logger.Err(err))
		return
	}

	var displayText string
	if displayAltText {
		displayText = altTextBuf.String()
	} else {
		displayText = textBuf.String()
	}

	output := outputData{
		Text:    displayText,
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
	target.Latitude = s.weather.Latitude
	target.Longitude = s.weather.Longitude
	target.Elevation = s.weather.Elevation
	target.Address = s.address

	// Moon phase
	m := moonphase.New(time.Now())
	target.Moonphase = m.PhaseName()
	target.MoonphaseIcon = MoonPhaseIcon[target.Moonphase]

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
	target.Current.ConditionIcon = WMOWeatherIcons[target.Current.WeatherCode][target.Current.IsDaytime]
	target.Current.Condition = s.t.Get(WMOWeatherCodes[target.Current.WeatherCode])
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
		target.Forecast.ConditionIcon = WMOWeatherIcons[target.Forecast.WeatherCode][target.Forecast.IsDaytime]
		target.Forecast.Condition = s.t.Get(WMOWeatherCodes[target.Forecast.WeatherCode])
	} else {
		target.Forecast = target.Current
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

	address, err := s.geocoder.Reverse(ctx, latitude, longitude)
	if err != nil {
		return fmt.Errorf("failed reverse geocode coordinates: %w", err)
	}
	location, err := omgo.NewLocation(latitude, longitude)
	if err != nil {
		return fmt.Errorf("failed create Open-Meteo location from coordinates: %w", err)
	}

	s.locationLock.Lock()
	s.location = location
	if address.AddressFound {
		s.address = address
	}
	s.locationIsSet = true
	s.locationLock.Unlock()
	s.logger.Debug("address successfully resolved", slog.Any("address", s.address.DisplayName),
		slog.Any("coordinates", s.location), slog.String("source", s.geocoder.Name()))

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

func (s *Service) weatherIndexByTime(atTime time.Time) int {
	for i, t := range s.weather.HourlyTimes {
		if t.Equal(atTime) {
			return i
		}
	}
	return -1
}

// handleAltTextToggleSignal toggles the module text display when a signal is received
func (s *Service) handleAltTextToggleSignal(ctx context.Context, sigChan chan os.Signal) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-sigChan:
			s.displayAltLock.Lock()
			s.displayAltText = !s.displayAltText
			s.displayAltLock.Unlock()
			s.printWeather(ctx)
		}
	}
}
