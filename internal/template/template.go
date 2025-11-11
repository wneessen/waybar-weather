// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package template

import (
	"fmt"
	"text/template"
	"time"

	"github.com/wneessen/waybar-weather/internal/config"

	"github.com/doppiogancio/go-nominatim/shared"
)

type DisplayData struct {
	// Location data
	Latitude  float64
	Longitude float64
	Elevation float64
	Address   shared.Address

	// General weather and moon phase data
	UpdateTime    time.Time
	TempUnit      string
	PressureUnit  string
	SunsetTime    time.Time
	SunriseTime   time.Time
	Moonphase     string
	MoonphaseIcon string

	// Current weather and forecast data
	Current  WeatherData
	Forecast WeatherData
}

type WeatherData struct {
	WeatherDateForTime  time.Time
	Temperature         float64
	ApparentTemperature float64
	Humidity            float64
	PressureMSL         float64
	WeatherCode         float64
	WindDirection       float64
	WindSpeed           float64
	ConditionIcon       string
	Condition           string
	IsDaytime           bool
}

type Templates struct {
	Text    *template.Template
	Tooltip *template.Template
}

func NewTemplate(conf *config.Config) (*Templates, error) {
	tpls := new(Templates)
	tpl, err := template.New("text").Funcs(templateFuncMap()).Parse(conf.Templates.Text)
	if err != nil {
		return tpls, fmt.Errorf("failed to parse text template: %w", err)
	}
	tpls.Text = tpl

	tpl, err = template.New("tooltip").Funcs(templateFuncMap()).Parse(conf.Templates.Tooltip)
	if err != nil {
		return tpls, fmt.Errorf("failed to parse tooltip template: %w", err)
	}
	tpls.Tooltip = tpl

	return tpls, nil
}

func templateFuncMap() template.FuncMap {
	return template.FuncMap{
		"timeFormat":  timeFormat,
		"floatFormat": floatFormat,
	}
}

func timeFormat(val time.Time, fmt string) string {
	return val.Format(fmt)
}

func floatFormat(val float64, precision int) string {
	return fmt.Sprintf("%.*f", precision, val)
}
