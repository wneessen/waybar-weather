// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"time"

	"github.com/wneessen/waybar-weather/internal/logger"

	"github.com/hectormalot/omgo"
)

const FetchTimeout = time.Second * 10

func (s *Service) fetchWeather(ctx context.Context) {
	ctxFetch, cancelFetch := context.WithTimeout(ctx, FetchTimeout)
	defer cancelFetch()

	if s.address == nil {
		return
	}

	opts := &omgo.Options{
		PastDays: 1,
		Timezone: "auto",
		HourlyMetrics: []string{
			"temperature_2m", "apparent_temperature", "weather_code", "wind_speed_10m", "is_day",
			"wind_direction_10m", "relative_humidity_2m", "pressure_msl",
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

	s.locationLock.RLock()
	defer s.locationLock.RUnlock()
	forecast, err := s.omclient.Forecast(ctxFetch, s.location, opts)
	if err != nil {
		s.logger.Error("failed to get forecast data", logger.Err(err))
		return
	}

	s.weatherLock.Lock()
	defer s.weatherLock.Unlock()
	s.weather = forecast
	s.weatherIsSet = true
}
