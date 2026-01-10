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

	Current  WeatherView
	Forecast []WeatherView
}

type Presenter struct {
	TextTemplate    *template.Template
	AltTextTemplate *template.Template
	TooltipTemplate *template.Template

	localizer *spreak.Localizer
	humanizer *humanize.Humanizer
}

// Supported languages for humanize
var supportedHumanizers = []*humanize.LocaleData{de.New()}

func New(conf *config.Config, loc *spreak.Localizer) (*Presenter, error) {
	presenter := &Presenter{localizer: loc}

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

func (p *Presenter) BuildContext(addr geocode.Address, data *weather.Data, sunrise, sunset time.Time,
	moonPhase, moonIcon string,
) TemplateContext {
	if data == nil {
		return TemplateContext{}
	}
	return TemplateContext{
		Latitude:      data.Coordinates.Lat,
		Longitude:     data.Coordinates.Lon,
		Address:       addr,
		UpdateTime:    data.GeneratedAt,
		SunriseTime:   sunrise,
		SunsetTime:    sunset,
		MoonPhase:     moonPhase,
		MoonPhaseIcon: moonIcon,
		Current:       p.viewFromInstant(data.Current),
		Forecast:      p.viewSliceFromMap(data.Forecast),
	}
}

func (p *Presenter) Render(tplCtx TemplateContext) (string, string, string, error) {
	var textBuf, altBuf, tooltipBuf bytes.Buffer

	if err := p.TextTemplate.Execute(&textBuf, tplCtx); err != nil {
		return "", "", "", fmt.Errorf("failed to render text template: %w", err)
	}
	if err := p.AltTextTemplate.Execute(&altBuf, tplCtx); err != nil {
		return "", "", "", fmt.Errorf("failed to render alt text template: %w", err)
	}
	if err := p.TooltipTemplate.Execute(&tooltipBuf, tplCtx); err != nil {
		return "", "", "", fmt.Errorf("failed to render tooltip template: %w", err)
	}

	return textBuf.String(), altBuf.String(), tooltipBuf.String(), nil
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

	return nil
}

// validateTemplates validates that the templates can be rendered
func (p *Presenter) validateTemplates() error {
	data := TemplateContext{Forecast: make([]WeatherView, 1)}
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

func (p *Presenter) viewFromInstant(in weather.Instant) WeatherView {
	return WeatherView{
		Instant: in,

		Condition:     WMOWeatherCodes[in.WeatherCode],
		ConditionIcon: WMOWeatherIcons[in.WeatherCode][in.IsDay],
	}
}

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
