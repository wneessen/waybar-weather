package presenter

import (
	"fmt"
	"math"
	"strings"
	"text/template"
	"time"

	"github.com/vorlif/humanize"
)

func (p *Presenter) templateFuncMap() template.FuncMap {
	return template.FuncMap{
		"timeFormat":      p.timeFormat,
		"localizedTime":   p.localizedTime,
		"floatFormat":     p.floatFormat,
		"loc":             p.loc,
		"lc":              strings.ToLower,
		"uc":              strings.ToUpper,
		"fcastHourOffset": p.forecastByOffset,
	}
}

func (p *Presenter) loc(val string) string {
	val = strings.ToLower(val)
	if raw, ok := i18nVars[val]; ok {
		return p.localizer.Get(raw)
	}
	return val
}

func (p *Presenter) localizedTime(val time.Time) string {
	return p.humanizer.FormatTime(val, humanize.TimeFormat)
}

func (p *Presenter) timeFormat(val time.Time, fmt string) string {
	return val.Format(fmt)
}

func (p *Presenter) floatFormat(val float64, precision int) string {
	pow := math.Pow(10, float64(precision))
	return fmt.Sprintf("%.*f", precision, math.Trunc(val*pow)/pow)
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
