// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package template

import (
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/wneessen/waybar-weather/internal/config"
	"github.com/wneessen/waybar-weather/internal/nominatim"

	"github.com/mattn/go-runewidth"
)

type DisplayData struct {
	// Location data
	Latitude  float64
	Longitude float64
	Elevation float64
	Address   nominatim.Address

	// General weather and moon phase data
	UpdateTime             time.Time
	TempUnit               string
	PressureUnit           string
	SunsetTime             time.Time
	SunriseTime            time.Time
	Moonphase              string
	MoonphaseIcon          string
	MoonphaseIconWithSpace string

	// Current weather and forecast data
	Current  WeatherData
	Forecast WeatherData
}

type WeatherData struct {
	WeatherDateForTime     time.Time
	Temperature            float64
	ApparentTemperature    float64
	Humidity               float64
	PressureMSL            float64
	WeatherCode            float64
	WindDirection          float64
	WindSpeed              float64
	ConditionIcon          string
	ConditionIconWithSpace string
	Condition              string
	IsDaytime              bool
}

type Templates struct {
	Text    *template.Template
	AltText *template.Template
	Tooltip *template.Template
}

func NewTemplate(conf *config.Config) (*Templates, error) {
	tpls := new(Templates)
	tpl, err := template.New("text").Funcs(templateFuncMap()).Parse(conf.Templates.Text)
	if err != nil {
		return tpls, fmt.Errorf("failed to parse text template: %w", err)
	}
	tpls.Text = tpl

	tpl, err = template.New("alt_text").Funcs(templateFuncMap()).Parse(conf.Templates.AltText)
	if err != nil {
		return tpls, fmt.Errorf("failed to parse alt text template: %w", err)
	}
	tpls.AltText = tpl

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

func EmojiWithSpace(emoji string) string {
	width := runewidth.StringWidth(emoji)
	return fmt.Sprintf("%s%s", emoji, strings.Repeat(" ", width+1))
}
