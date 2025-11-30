// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package open_meteo

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/wneessen/waybar-weather/internal/geobus"
	"github.com/wneessen/waybar-weather/internal/http"
	"github.com/wneessen/waybar-weather/internal/logger"
	"github.com/wneessen/waybar-weather/internal/weather"
)

const (
	name        = "open-meteo"
	apiEndpoint = "https://api.open-meteo.com/v1/forecast"
	apiTimeout  = time.Second * 10
)

var dataFields = []string{
	"temperature_2m", "apparent_temperature", "weather_code", "wind_speed_10m", "is_day",
	"wind_direction_10m", "relative_humidity_2m", "pressure_msl",
}

type OpenMeteo struct {
	unit string
	log  *logger.Logger
	http *http.Client
}

type resTime struct {
	time.Time
}

type resBool struct {
	bool
}

type response struct {
	Latitude             float64 `json:"latitude"`
	Longitude            float64 `json:"longitude"`
	GenerationTimeMs     float64 `json:"generationtime_ms"`
	UTCOffsetSeconds     int     `json:"utc_offset_seconds"`
	Timezone             string  `json:"timezone"`
	TimezoneAbbreviation string  `json:"timezone_abbreviation"`
	Elevation            float64 `json:"elevation"`
	CurrentUnits         struct {
		Time                string `json:"time"`
		Interval            string `json:"interval"`
		Temperature2M       string `json:"temperature_2m"`
		ApparentTemperature string `json:"apparent_temperature"`
		WeatherCode         string `json:"weather_code"`
		WindSpeed10M        string `json:"wind_speed_10m"`
		IsDay               string `json:"is_day"`
		WindDirection10M    string `json:"wind_direction_10m"`
		RelativeHumidity2M  string `json:"relative_humidity_2m"`
		PressureMsl         string `json:"pressure_msl"`
	} `json:"current_units"`
	Current struct {
		Time                resTime `json:"time"`
		Interval            int     `json:"interval"`
		Temperature         float64 `json:"temperature_2m"`
		ApparentTemperature float64 `json:"apparent_temperature"`
		WeatherCode         int     `json:"weather_code"`
		WindSpeed           float64 `json:"wind_speed_10m"`
		IsDay               resBool `json:"is_day"`
		WindDirection       int     `json:"wind_direction_10m"`
		RelativeHumidity    int     `json:"relative_humidity_2m"`
		PressureMSL         float64 `json:"pressure_msl"`
	} `json:"current"`
	HourlyUnits struct {
		Time                string `json:"time"`
		Temperature         string `json:"temperature_2m"`
		ApparentTemperature string `json:"apparent_temperature"`
		WeatherCode         string `json:"weather_code"`
		WindSpeed           string `json:"wind_speed_10m"`
		IsDay               string `json:"is_day"`
		WindDirection       string `json:"wind_direction_10m"`
		RelativeHumidity    string `json:"relative_humidity_2m"`
		PressureMsl         string `json:"pressure_msl"`
	} `json:"hourly_units"`
	Hourly struct {
		Time                []resTime `json:"time"`
		Temperature         []float64 `json:"temperature_2m"`
		ApparentTemperature []float64 `json:"apparent_temperature"`
		WeatherCode         []int     `json:"weather_code"`
		WindSpeed           []float64 `json:"wind_speed_10m"`
		IsDay               []resBool `json:"is_day"`
		WindDirection       []int     `json:"wind_direction_10m"`
		RelativeHumidity    []int     `json:"relative_humidity_2m"`
		PressureMsl         []float64 `json:"pressure_msl"`
	} `json:"hourly"`
}

type Hourly struct {
	Time        []time.Time `json:"time"`
	Temperature []float64   `json:"temperature_2m"`
}

func New(http *http.Client, log *logger.Logger, unit string) (*OpenMeteo, error) {
	if http == nil {
		return nil, fmt.Errorf("http client is required")
	}
	if log == nil {
		return nil, fmt.Errorf("logger is required")
	}

	return &OpenMeteo{unit: unit, http: http, log: log}, nil
}

func (o *OpenMeteo) Name() string {
	return name
}

func (o *OpenMeteo) GetWeather(ctx context.Context, coords geobus.Coordinate) (*weather.Data, error) {
	res := new(response)
	data := weather.NewData()

	// latitude=52.52&longitude=13.41&current=temperature_2m,wind_speed_10m&hourly=temperature_2m,relative_humidity_2m,wind_speed_10m
	query := url.Values{}
	query.Set("latitude", fmt.Sprintf("%f", coords.Lat))
	query.Set("longitude", fmt.Sprintf("%f", coords.Lon))
	query.Set("current", strings.Join(dataFields, ","))
	query.Set("hourly", strings.Join(dataFields, ","))
	query.Set("timezone", "auto")
	query.Set("past_days", "1")
	if strings.ToLower(o.unit) == "imperial" {
		query.Set("temperature_unit", "fahrenheit")
		query.Set("wind_speed_unit", "mph")
		query.Set("precipitation_unit", "inch")
	}

	code, err := o.http.GetWithTimeout(ctx, apiEndpoint, res, query, nil, apiTimeout)
	if err != nil {
		return data, fmt.Errorf("failed to retrieve weather data from Open-Meteo API: %w", err)
	}
	if code != 200 {
		return data, fmt.Errorf("Open-Meteo API returned non-positive response code: %d", code)
	}

	data.GeneratedAt = time.Now()
	data.Coordinates = coords
	data.Current = weather.Instant{
		InstantTime:         res.Current.Time.Time,
		Temperature:         res.Current.Temperature,
		ApparentTemperature: res.Current.ApparentTemperature,
		WeatherCode:         res.Current.WeatherCode,
		WindSpeed:           res.Current.WindSpeed,
		WindDirection:       float64(res.Current.WindDirection),
		RelativeHumidity:    float64(res.Current.RelativeHumidity),
		PressureMSL:         res.Current.PressureMSL,
		IsDay:               res.Current.IsDay.bool,
	}
	for i := range res.Hourly.Time {
		timePos := weather.NewDayHour(res.Hourly.Time[i].Time)
		instant := weather.Instant{
			InstantTime:         timePos.Time(),
			Temperature:         res.Hourly.Temperature[i],
			ApparentTemperature: res.Hourly.ApparentTemperature[i],
			WeatherCode:         res.Hourly.WeatherCode[i],
			WindSpeed:           res.Hourly.WindSpeed[i],
			WindDirection:       float64(res.Hourly.WindDirection[i]),
			RelativeHumidity:    float64(res.Hourly.RelativeHumidity[i]),
			PressureMSL:         res.Hourly.PressureMsl[i],
			IsDay:               res.Hourly.IsDay[i].bool,
		}
		data.Forecast[timePos] = instant
	}

	return data, nil
}

func (r *resTime) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return fmt.Errorf("empty time")
	}
	if b[0] != '"' {
		return fmt.Errorf("invalid time format: %s", string(b))
	}

	apiTime, err := time.Parse("2006-01-02T15:04", string(b[1:len(b)-1]))
	if err != nil {
		return fmt.Errorf("failed to parse time: %w", err)
	}
	r.Time = apiTime

	return nil
}

func (r *resBool) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return fmt.Errorf("empty bool")
	}
	if b[0] == '0' {
		return nil
	}
	r.bool = true
	return nil
}
