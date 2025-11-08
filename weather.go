// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package main

import (
	"context"

	"github.com/hectormalot/omgo"
)

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
