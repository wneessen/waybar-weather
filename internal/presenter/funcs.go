// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package presenter

import (
	"fmt"
	"math"
	"strings"
	"text/template"
	"time"

	"github.com/vorlif/humanize"

	"github.com/wneessen/waybar-weather/internal/vartype"
)

func (p *Presenter) templateFuncMap() template.FuncMap {
	return template.FuncMap{
		"timeFormat":      p.timeFormat,
		"localizedTime":   p.localizedTime,
		"floatFormat":     p.floatFormat,
		"loc":             p.loc,
		"hum":             p.hum,
		"lc":              strings.ToLower,
		"uc":              strings.ToUpper,
		"fcastHourOffset": p.forecastByOffset,
		"windDir":         p.degToString,
		"windDirIcon":     p.windDirIcon,
	}
}

func (p *Presenter) loc(val string) string {
	val = strings.ToLower(val)
	if raw, ok := i18nVars[val]; ok {
		return p.localizer.Get(raw)
	}
	return val
}

func (p *Presenter) hum(val vartype.VarFloat64) string {
	return p.printer.Sprintf("%.1f", val.Value())
}

func (p *Presenter) localizedTime(val time.Time) string {
	return p.humanizer.FormatTime(val, humanize.TimeFormat)
}

func (p *Presenter) timeFormat(val time.Time, fmt string) string {
	return val.Format(fmt)
}

func (p *Presenter) floatFormat(val vartype.VarFloat64, precision int) string {
	pow := math.Pow(10, float64(precision))
	return fmt.Sprintf("%.*f", precision, math.Trunc(val.Value()*pow)/pow)
}

// forecast returns the forecast at the given offset (0-based).
func (p *Presenter) forecastByOffset(ctx TemplateContext, offset int) WeatherView {
	if offset < 0 || offset >= len(ctx.Forecasts) {
		return WeatherView{}
	}

	currentUTC := ctx.Current.InstantTime.Truncate(time.Hour)
	want := currentUTC.In(time.Local).Add(time.Hour * time.Duration(offset))
	for _, fcast := range ctx.Forecasts {
		if fcast.InstantTime.Equal(want) {
			return fcast
		}
	}

	return WeatherView{}
}

func (p *Presenter) degToString(deg vartype.VarFloat64) string {
	switch {
	case deg.Value() < 22.5:
		return "N"
	case deg.Value() < 67.5:
		return "NE"
	case deg.Value() < 112.5:
		return "E"
	case deg.Value() < 157.5:
		return "SE"
	case deg.Value() < 202.5:
		return "S"
	case deg.Value() < 247.5:
		return "SW"
	case deg.Value() < 292.5:
		return "W"
	case deg.Value() < 337.5:
		return "NW"
	default:
		return "N"
	}
}

func (p *Presenter) windDirIcon(dir string) string {
	if icon, ok := windDirIcons[strings.ToUpper(dir)]; ok {
		return icon
	}
	return ""
}
