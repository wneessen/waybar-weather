// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package weather

import (
	"context"
	"time"

	"github.com/wneessen/waybar-weather/internal/geobus"
	"github.com/wneessen/waybar-weather/internal/vartype"
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
	Daily    map[Day]Instant
}

type Instant struct {
	InstantTime              time.Time
	Temperature              vartype.VarFloat64
	ApparentTemperature      vartype.VarFloat64
	WeatherCode              vartype.VarInt
	WindSpeed                vartype.VarFloat64
	WindGusts                vartype.VarFloat64
	WindDirection            vartype.VarFloat64
	RelativeHumidity         vartype.VarFloat64
	PrecipitationProbability vartype.VarInt
	PressureMSL              vartype.VarFloat64
	IsDay                    vartype.VarBool
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
type Day int64

func NewData() *Data {
	return &Data{
		Forecast: make(map[DayHour]Instant),
		Daily:    make(map[Day]Instant),
	}
}

func NewDayHour(t time.Time) DayHour {
	return DayHour(t.Truncate(time.Hour).Unix())
}

func NewDay(t time.Time) Day {
	return Day(t.Truncate(time.Hour * 24).Unix())
}

func (t DayHour) Time() time.Time {
	return time.Unix(int64(t), 0)
}

func (t Day) Time() time.Time {
	return time.Unix(int64(t), 0)
}
