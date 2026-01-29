// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package weather

import (
	"context"
	"time"

	"github.com/wneessen/waybar-weather/internal/geobus"
)

// Provider is implemented by each weather API backend.
type Provider interface {
	Name() string
	GetWeather(ctx context.Context, coords geobus.Coordinate) (*Data, error)
}

type Data struct {
	GeneratedAt time.Time
	Coordinates geobus.Coordinate

	Current  Instant
	Forecast map[DayHour]Instant
}

type Instant struct {
	InstantTime              time.Time
	Temperature              float64
	ApparentTemperature      float64
	WeatherCode              int
	WindSpeed                float64
	WindGusts                float64
	WindDirection            float64
	RelativeHumidity         float64
	PrecipitationProbability float64
	PressureMSL              float64
	IsDay                    bool
	Units                    Units
}

type Units struct {
	Temperature   string
	WindSpeed     string
	Humidity      string
	Pressure      string
	WindDirection string
}

type DayHour int64

func NewData() *Data {
	return &Data{
		Forecast: make(map[DayHour]Instant),
	}
}

func NewDayHour(t time.Time) DayHour {
	return DayHour(t.Truncate(time.Hour).Unix())
}

func (t DayHour) Time() time.Time {
	return time.Unix(int64(t), 0)
}
