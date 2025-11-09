// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"text/template"
	"time"

	"github.com/doppiogancio/go-nominatim/shared"
)

type DisplayData struct {
	Latitude           float64
	Longitude          float64
	Elevation          float64
	Address            shared.Address
	UpdateTime         time.Time
	WeatherDateForTime time.Time
	Temperature        float64
	WeatherCode        float64
	WindDirection      float64
	WindSpeed          float64
	IsDaytime          bool
	TempUnit           string
	SunsetTime         time.Time
	SunriseTime        time.Time
	ConditionIcon      string
	Condition          string
	Moonphase          string
	MoonphaseIcon      string
}

type Templates struct {
	Text    *template.Template
	Tooltip *template.Template
}

func NewTemplate(conf *config) (*Templates, error) {
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
