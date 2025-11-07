// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"time"

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
var WMOWeatherIcons = map[float64]map[string]string{
	0:  {"day": "☀️", "night": "🌕"},  // Clear sky
	1:  {"day": "🌤️", "night": "🌤️"}, // Mainly clear
	2:  {"day": "⛅", "night": "☁️"},  // Partly cloudy
	3:  {"day": "☁️", "night": "☁️"}, // Overcast
	45: {"day": "🌫️", "night": "🌫️"}, // Fog
	48: {"day": "🌫️", "night": "🌫️"}, // Depositing rime fog
	51: {"day": "🌦️", "night": "🌧️"}, // Light drizzle
	53: {"day": "🌧️", "night": "🌧️"}, // Moderate drizzle
	55: {"day": "🌧️", "night": "🌧️"}, // Dense drizzle
	56: {"day": "🌨️", "night": "🌨️"}, // Light freezing drizzle
	57: {"day": "🌨️", "night": "🌨️"}, // Dense freezing drizzle
	61: {"day": "🌦️", "night": "🌧️"}, // Slight rain
	63: {"day": "🌧️", "night": "🌧️"}, // Moderate rain
	65: {"day": "🌧️", "night": "🌧️"}, // Heavy rain
	66: {"day": "🌨️", "night": "🌨️"}, // Light freezing rain
	67: {"day": "🌨️", "night": "🌨️"}, // Heavy freezing rain
	71: {"day": "🌨️", "night": "🌨️"}, // Slight snow fall
	73: {"day": "❄️", "night": "❄️"}, // Moderate snow fall
	75: {"day": "❄️", "night": "❄️"}, // Heavy snow fall
	77: {"day": "🌨️", "night": "🌨️"}, // Snow grains
	80: {"day": "🌦️", "night": "🌧️"}, // Slight rain showers
	81: {"day": "🌧️", "night": "🌧️"}, // Moderate rain showers
	82: {"day": "🌧️", "night": "🌧️"}, // Violent rain showers
	85: {"day": "🌨️", "night": "🌨️"}, // Slight snow showers
	86: {"day": "🌨️", "night": "🌨️"}, // Heavy snow showers
	95: {"day": "⛈️", "night": "⛈️"}, // Thunderstorm
	96: {"day": "🌩️", "night": "🌩️"}, // Thunderstorm with slight hail
	99: {"day": "🌩️", "night": "🌩️"}, // Thunderstorm with heavy hail
}

func (s *Service) fetchWeather(ctx context.Context) {
	s.weatherLock.Lock()
	defer s.weatherLock.Unlock()
	s.locationLock.RLock()
	defer s.locationLock.RUnlock()

	if s.address == nil {
		return
	}

	tz, _ := time.Now().Zone()
	opts := &omgo.Options{Timezone: tz}
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

	current, err := s.omclient.CurrentWeather(ctx, s.location, opts)
	if err != nil {
		s.logger.Error("failed to get current weather data", logError(err))
		return
	}
	s.weather = current
	s.weatherIsSet = true
}
