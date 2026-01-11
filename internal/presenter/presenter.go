// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package presenter

import (
	"bytes"
	"fmt"
	"sort"
	"text/template"
	"time"

	"github.com/vorlif/humanize"
	"github.com/vorlif/humanize/locale/de"
	"github.com/vorlif/spreak"

	"github.com/wneessen/waybar-weather/internal/config"
	"github.com/wneessen/waybar-weather/internal/geocode"
	"github.com/wneessen/waybar-weather/internal/weather"
)

// WeatherView wraps a domain Instant with presentation-related fields.
type WeatherView struct {
	weather.Instant

	Condition     string
	ConditionIcon string
}

type TemplateContext struct {
	Latitude  float64
	Longitude float64
	Address   geocode.Address

	UpdateTime    time.Time
	PressureUnit  string
	SunriseTime   time.Time
	SunsetTime    time.Time
	MoonPhase     string
	MoonPhaseIcon string

	Current   WeatherView
	Forecast  WeatherView
	Forecasts []WeatherView
}

type Presenter struct {
	TextTemplate       *template.Template
	AltTextTemplate    *template.Template
	TooltipTemplate    *template.Template
	AltTooltipTemplate *template.Template

	localizer     *spreak.Localizer
	humanizer     *humanize.Humanizer
	forecastHours uint
}

// Supported languages for humanize
var supportedHumanizers = []*humanize.LocaleData{de.New()}

// New initializes and returns a new Presenter instance with the provided configuration and localizer.
// It parses templates, creates a humanizer, and validates the templates for rendering.
// Returns an error if any step in initialization fails.
func New(conf *config.Config, loc *spreak.Localizer) (*Presenter, error) {
	presenter := &Presenter{localizer: loc, forecastHours: conf.Weather.ForecastHours}

	// Parse the templates
	if err := presenter.parseTemplates(conf); err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	// Create humanizer
	collection, err := humanize.New(humanize.WithLocale(supportedHumanizers...))
	if err != nil {
		return presenter, fmt.Errorf("failed to create humanizer: %w", err)
	}
	presenter.humanizer = collection.CreateHumanizer(loc.Language())

	// Validate that the templates can be rendered
	if err = presenter.validateTemplates(); err != nil {
		return presenter, fmt.Errorf("failed to validate templates: %w", err)
	}

	return presenter, nil
}

// BuildContext constructs and returns a populated TemplateContext based on provided address, weather data,
// and timings data.
func (p *Presenter) BuildContext(addr geocode.Address, data *weather.Data, sunrise, sunset time.Time, moonPhase string) TemplateContext {
	if data == nil {
		return TemplateContext{}
	}
	fcastHour := weather.NewDayHour(time.Now().Add(time.Hour * time.Duration(p.forecastHours)))
	return TemplateContext{
		Latitude:      data.Coordinates.Lat,
		Longitude:     data.Coordinates.Lon,
		Address:       addr,
		UpdateTime:    data.GeneratedAt,
		SunriseTime:   sunrise,
		SunsetTime:    sunset,
		MoonPhase:     moonPhase,
		MoonPhaseIcon: MoonPhaseIcon[moonPhase],
		Current:       p.viewFromInstant(data.Current),
		Forecast:      p.viewFromInstant(data.Forecast[fcastHour]),
		Forecasts:     p.viewSliceFromMap(data.Forecast),
	}
}

// Render processes the given TemplateContext and generates text, alternative text, and tooltip content as strings.
func (p *Presenter) Render(tplCtx TemplateContext) (map[string]string, error) {
	buf := bytes.NewBuffer(nil)
	valMap := make(map[string]string)

	if err := p.TextTemplate.Execute(buf, tplCtx); err != nil {
		return valMap, fmt.Errorf("failed to render text template: %w", err)
	}
	valMap["text"] = buf.String()
	buf.Reset()

	if err := p.AltTextTemplate.Execute(buf, tplCtx); err != nil {
		return valMap, fmt.Errorf("failed to render alt text template: %w", err)
	}
	valMap["alt_text"] = buf.String()
	buf.Reset()

	if err := p.TooltipTemplate.Execute(buf, tplCtx); err != nil {
		return valMap, fmt.Errorf("failed to render tooltip template: %w", err)
	}
	valMap["tooltip"] = buf.String()
	buf.Reset()

	if err := p.AltTooltipTemplate.Execute(buf, tplCtx); err != nil {
		return valMap, fmt.Errorf("failed to render tooltip template: %w", err)
	}
	valMap["alt_tooltip"] = buf.String()
	buf.Reset()

	return valMap, nil
}

// parseTemplates parses the templates from the config and stores them in the Presenter struct
func (p *Presenter) parseTemplates(conf *config.Config) error {
	tpl, err := template.New("text").Funcs(p.templateFuncMap()).Parse(conf.Templates.Text)
	if err != nil {
		return fmt.Errorf("failed to parse text template: %w", err)
	}
	p.TextTemplate = tpl

	tpl, err = template.New("alt_text").Funcs(p.templateFuncMap()).Parse(conf.Templates.AltText)
	if err != nil {
		return fmt.Errorf("failed to parse alternative text template: %w", err)
	}
	p.AltTextTemplate = tpl

	tpl, err = template.New("tooltip").Funcs(p.templateFuncMap()).Parse(conf.Templates.Tooltip)
	if err != nil {
		return fmt.Errorf("failed to parse tooltip template: %w", err)
	}
	p.TooltipTemplate = tpl

	tpl, err = template.New("alt_tooltip").Funcs(p.templateFuncMap()).Parse(conf.Templates.AltTooltip)
	if err != nil {
		return fmt.Errorf("failed to parse tooltip template: %w", err)
	}
	p.AltTooltipTemplate = tpl

	return nil
}

// validateTemplates validates that the templates can be rendered
func (p *Presenter) validateTemplates() error {
	data := TemplateContext{Forecasts: make([]WeatherView, 1)}
	if err := p.TextTemplate.Execute(bytes.NewBuffer(nil), data); err != nil {
		return fmt.Errorf("failed to render text template: %w", err)
	}
	if err := p.AltTextTemplate.Execute(bytes.NewBuffer(nil), data); err != nil {
		return fmt.Errorf("failed to render alternative text template: %w", err)
	}
	if err := p.TooltipTemplate.Execute(bytes.NewBuffer(nil), data); err != nil {
		return fmt.Errorf("failed to render tooltip template: %w", err)
	}

	return nil
}

// viewFromInstant converts a weather.Instant into a WeatherView with condition details and corresponding icon.
func (p *Presenter) viewFromInstant(in weather.Instant) WeatherView {
	return WeatherView{
		Instant: in,

		Condition:     WMOWeatherCodes[in.WeatherCode],
		ConditionIcon: WMOWeatherIcons[in.WeatherCode][in.IsDay],
	}
}

// viewSliceFromMap converts a map of DayHour-Instant pairs into a sorted slice of WeatherView based on InstantTime.
func (p *Presenter) viewSliceFromMap(m map[weather.DayHour]weather.Instant) []WeatherView {
	views := make([]WeatherView, 0, len(m))
	for _, inst := range m {
		views = append(views, p.viewFromInstant(inst))
	}
	sort.Slice(views, func(i, j int) bool {
		return views[i].InstantTime.Before(views[j].InstantTime)
	})
	return views
}
