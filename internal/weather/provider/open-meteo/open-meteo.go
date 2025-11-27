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
	log  *logger.Logger
	http *http.Client
}

/*
"latitude": 52.52,

		"longitude": 13.419998,
		"generationtime_ms": 0.07426738739013672,
		"utc_offset_seconds": 3600,
		"timezone": "Europe/Berlin",
		"timezone_abbreviation": "GMT+1",
		"elevation": 38.0,
		"hourly_units": {
		  "time": "iso8601",
		  "temperature_2m": "°C"
		},
	 "current_units": {
	    "time": "iso8601",
	    "interval": "seconds",
	    "temperature_2m": "°C"
	  },
	  "current": {
	    "time": "2025-11-27T22:15",
	    "interval": 900,
	    "temperature_2m": 3.5
	  },
*/
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
		Time                string  `json:"time"`
		Interval            int     `json:"interval"`
		Temperature2M       float64 `json:"temperature_2m"`
		ApparentTemperature float64 `json:"apparent_temperature"`
		WeatherCode         int     `json:"weather_code"`
		WindSpeed10M        float64 `json:"wind_speed_10m"`
		IsDay               int     `json:"is_day"`
		WindDirection10M    int     `json:"wind_direction_10m"`
		RelativeHumidity2M  int     `json:"relative_humidity_2m"`
		PressureMsl         float64 `json:"pressure_msl"`
	} `json:"current"`
	HourlyUnits struct {
		Time                string `json:"time"`
		Temperature2M       string `json:"temperature_2m"`
		ApparentTemperature string `json:"apparent_temperature"`
		WeatherCode         string `json:"weather_code"`
		WindSpeed10M        string `json:"wind_speed_10m"`
		IsDay               string `json:"is_day"`
		WindDirection10M    string `json:"wind_direction_10m"`
		RelativeHumidity2M  string `json:"relative_humidity_2m"`
		PressureMsl         string `json:"pressure_msl"`
	} `json:"hourly_units"`
	Hourly struct {
		Time                []resTime `json:"time"`
		Temperature2M       []float64 `json:"temperature_2m"`
		ApparentTemperature []float64 `json:"apparent_temperature"`
		WeatherCode         []int     `json:"weather_code"`
		WindSpeed10M        []float64 `json:"wind_speed_10m"`
		IsDay               []resBool `json:"is_day"`
		WindDirection10M    []int     `json:"wind_direction_10m"`
		RelativeHumidity2M  []int     `json:"relative_humidity_2m"`
		PressureMsl         []float64 `json:"pressure_msl"`
	} `json:"hourly"`
}

type Hourly struct {
	Time        []time.Time `json:"time"`
	Temperature []float64   `json:"temperature_2m"`
}

func New(http *http.Client, log *logger.Logger) (*OpenMeteo, error) {
	if http == nil {
		return nil, fmt.Errorf("http client is required")
	}
	if log == nil {
		return nil, fmt.Errorf("logger is required")
	}

	return &OpenMeteo{http: http, log: log}, nil
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

	code, err := o.http.GetWithTimeout(ctx, apiEndpoint, res, query, nil, apiTimeout)
	if err != nil {
		return data, fmt.Errorf("failed to retrieve weather data from Open-Meteo API: %w", err)
	}
	if code != 200 {
		return data, fmt.Errorf("Open-Meteo API returned non-positive response code: %d", code)
	}

	for i, t := range res.Hourly.Time {
		data.Temperature[t.Time] = res.Hourly.Temperature2M[i]
	}

	fmt.Printf("%+v", res)

	return nil, nil
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
