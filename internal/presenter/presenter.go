// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package presenter

import (
	"sort"
	"time"

	"github.com/wneessen/waybar-weather/internal/geocode"
	"github.com/wneessen/waybar-weather/internal/weather"
)

// WeatherView wraps a domain Instant with presentation-related fields.
type WeatherView struct {
	weather.Instant

	Condition     string
	ConditionIcon string
}

type TemplateContext struct {
	Latitude  float64
	Longitude float64
	Address   geocode.Address

	UpdateTime    time.Time
	TempUnit      string
	PressureUnit  string
	SunriseTime   time.Time
	SunsetTime    time.Time
	MoonPhase     string
	MoonPhaseIcon string

	Current  WeatherView
	Forecast []WeatherView
}

type Presenter struct{}

func (p *Presenter) BuildContext(addr geocode.Address, data *weather.Data, sunrise, sunset time.Time,
	moonPhase, moonIcon string,
) TemplateContext {
	return TemplateContext{
		Latitude:      data.Coordinates.Lat,
		Longitude:     data.Coordinates.Lon,
		Address:       addr,
		UpdateTime:    data.GeneratedAt,
		SunriseTime:   sunrise,
		SunsetTime:    sunset,
		MoonPhase:     moonPhase,
		MoonPhaseIcon: moonIcon,
		Current:       p.viewFromInstant(data.Current),
		Forecast:      p.viewSliceFromMap(data.Forecast),
	}
}

func (p *Presenter) viewFromInstant(in weather.Instant) WeatherView {
	return WeatherView{
		Instant:       in,
		Condition:     WMOWeatherCodes[in.WeatherCode],
		ConditionIcon: WMOWeatherIcons[in.WeatherCode][in.IsDay],
	}
}

func (p *Presenter) viewSliceFromMap(m map[weather.DayHour]weather.Instant) []WeatherView {
	views := make([]WeatherView, 0, len(m))
	for _, inst := range m {
		views = append(views, p.viewFromInstant(inst))
	}
	sort.Slice(views, func(i, j int) bool {
		return views[i].InstantTime.Before(views[j].InstantTime)
	})
	return views
}
