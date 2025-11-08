// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package main

import (
	"context"

	"github.com/hectormalot/omgo"
)

// WMOWeatherCodes maps WMO weather code integers to their descriptions
var WMOWeatherCodes = map[float64]string{
	0:  "Clear sky",
	1:  "Mainly clear",
	2:  "Partly cloudy",
	3:  "Overcast",
	45: "Fog",
	48: "Depositing rime fog",
	51: "Light drizzle",
	53: "Moderate drizzle",
	55: "Dense drizzle",
	56: "Light freezing drizzle",
	57: "Dense freezing drizzle",
	61: "Slight rain",
	63: "Moderate rain",
	65: "Heavy rain",
	66: "Light freezing rain",
	67: "Heavy freezing rain",
	71: "Slight snow fall",
	73: "Moderate snow fall",
	75: "Heavy snow fall",
	77: "Snow grains",
	80: "Slight rain showers",
	81: "Moderate rain showers",
	82: "Violent rain showers",
	85: "Slight snow showers",
	86: "Heavy snow showers",
	95: "Thunderstorm",
	96: "Thunderstorm with slight hail",
	99: "Thunderstorm with heavy hail",
}

// WMOWeatherIcons maps WMO weather codes to single emoji icons for day and night
var WMOWeatherIcons = map[float64]map[bool]string{
	0:  {true: "☀️", false: "🌕"},  // Clear sky
	1:  {true: "🌤️", false: "🌤️"}, // Mainly clear
	2:  {true: "⛅", false: "☁️"},  // Partly cloudy
	3:  {true: "☁️", false: "☁️"}, // Overcast
	45: {true: "🌫️", false: "🌫️"}, // Fog
	48: {true: "🌫️", false: "🌫️"}, // Depositing rime fog
	51: {true: "🌦️", false: "🌧️"}, // Light drizzle
	53: {true: "🌧️", false: "🌧️"}, // Moderate drizzle
	55: {true: "🌧️", false: "🌧️"}, // Dense drizzle
	56: {true: "🌨️", false: "🌨️"}, // Light freezing drizzle
	57: {true: "🌨️", false: "🌨️"}, // Dense freezing drizzle
	61: {true: "🌦️", false: "🌧️"}, // Slight rain
	63: {true: "🌧️", false: "🌧️"}, // Moderate rain
	65: {true: "🌧️", false: "🌧️"}, // Heavy rain
	66: {true: "🌨️", false: "🌨️"}, // Light freezing rain
	67: {true: "🌨️", false: "🌨️"}, // Heavy freezing rain
	71: {true: "🌨️", false: "🌨️"}, // Slight snow fall
	73: {true: "❄️", false: "❄️"}, // Moderate snow fall
	75: {true: "❄️", false: "❄️"}, // Heavy snow fall
	77: {true: "🌨️", false: "🌨️"}, // Snow grains
	80: {true: "🌦️", false: "🌧️"}, // Slight rain showers
	81: {true: "🌧️", false: "🌧️"}, // Moderate rain showers
	82: {true: "🌧️", false: "🌧️"}, // Violent rain showers
	85: {true: "🌨️", false: "🌨️"}, // Slight snow showers
	86: {true: "🌨️", false: "🌨️"}, // Heavy snow showers
	95: {true: "⛈️", false: "⛈️"}, // Thunderstorm
	96: {true: "🌩️", false: "🌩️"}, // Thunderstorm with slight hail
	99: {true: "🌩️", false: "🌩️"}, // Thunderstorm with heavy hail
}

func (s *Service) fetchWeather(ctx context.Context) {
	s.weatherLock.Lock()
	defer s.weatherLock.Unlock()
	s.locationLock.RLock()
	defer s.locationLock.RUnlock()

	if s.address == nil {
		return
	}

	opts := &omgo.Options{
		Timezone: "auto",
		HourlyMetrics: []string{
			"temperature_2m", "weather_code", "wind_speed_10m", "is_day", "wind_direction_10m",
		},
	}
	switch s.config.Units {
	case "metric":
		opts.TemperatureUnit = "celsius"
		opts.PrecipitationUnit = "mm"
		opts.WindspeedUnit = "kmh"
	case "imperial":
		opts.TemperatureUnit = "fahrenheit"
		opts.PrecipitationUnit = "inch"
		opts.WindspeedUnit = "mph"
	}

	forecast, err := s.omclient.Forecast(ctx, s.location, opts)
	if err != nil {
		s.logger.Error("failed to get forecast data", logError(err))
		return
	}
	s.weather = forecast
	s.weatherIsSet = true
}
