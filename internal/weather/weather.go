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

	Temperature         map[time.Time]float64
	ApparentTemperature map[time.Time]float64
	WeatherCode         map[time.Time]int
	WindSpeed           map[time.Time]float64
	IsDay               map[time.Time]bool
	WindDirection       map[time.Time]float64
	RelativeHumidity    map[time.Time]float64
	PressureMSL         map[time.Time]float64
}

func NewData() *Data {
	return &Data{
		Temperature:         make(map[time.Time]float64),
		ApparentTemperature: make(map[time.Time]float64),
		WeatherCode:         make(map[time.Time]int),
		WindSpeed:           make(map[time.Time]float64),
		IsDay:               make(map[time.Time]bool),
		WindDirection:       make(map[time.Time]float64),
		RelativeHumidity:    make(map[time.Time]float64),
		PressureMSL:         make(map[time.Time]float64),
	}
}
