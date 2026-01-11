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
		"timeFormat":    p.timeFormat,
		"localizedTime": p.localizedTime,
		"floatFormat":   p.floatFormat,
		"loc":           p.loc,
		"lc":            strings.ToLower,
		"uc":            strings.ToUpper,
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
func forecast(ctx TemplateContext, offset int) WeatherView {
	if offset < 0 || offset >= len(ctx.Forecasts) {
		return WeatherView{} // zero value; templates will see empty fields
	}
	return ctx.Forecasts[offset]
}

// firstForecast returns the earliest forecast entry, if any.
func firstForecast(ctx TemplateContext) WeatherView {
	return forecast(ctx, 0)
}
