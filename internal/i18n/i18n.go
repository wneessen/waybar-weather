// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package i18n

import (
	"embed"
	"fmt"
	"io/fs"

	"github.com/Xuanwo/go-locale"
	"github.com/vorlif/spreak"
	"golang.org/x/text/language"
)

//go:embed locale/*
var locales embed.FS

func New(loc string) (*spreak.Localizer, error) {
	tag := language.Make(loc)
	var err error
	if loc == "" {
		tag, err = locale.Detect()
		if err != nil {
			tag = language.English // Unable to detect locale, fallback to English
		}
	}

	localeFS, err := fs.Sub(locales, "locale")
	if err != nil {
		return nil, fmt.Errorf("failed to load locales: %w", err)
	}

	bundle, err := spreak.NewBundle(
		spreak.WithSourceLanguage(language.English),
		spreak.WithFallbackLanguage(language.English),
		spreak.WithDomainFs("", localeFS),
		spreak.WithLanguage(tag),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create i18n bundle: %w", err)
	}
	return spreak.NewLocalizer(bundle, tag), nil
}
