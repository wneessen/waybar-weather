// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package service

import (
	"fmt"
	"strings"

	"golang.org/x/text/language"

	"github.com/wneessen/waybar-weather/internal/config"
	"github.com/wneessen/waybar-weather/internal/geobus"
	"github.com/wneessen/waybar-weather/internal/geobus/provider/geoapi"
	"github.com/wneessen/waybar-weather/internal/geobus/provider/geoip"
	"github.com/wneessen/waybar-weather/internal/geobus/provider/geolocation_file"
	"github.com/wneessen/waybar-weather/internal/geobus/provider/gpsd"
	"github.com/wneessen/waybar-weather/internal/geobus/provider/ichnaea"
	"github.com/wneessen/waybar-weather/internal/geocode"
	geocodeearth "github.com/wneessen/waybar-weather/internal/geocode/provider/geocode-earth"
	"github.com/wneessen/waybar-weather/internal/geocode/provider/opencage"
	nominatim "github.com/wneessen/waybar-weather/internal/geocode/provider/osm-nominatim"
	"github.com/wneessen/waybar-weather/internal/http"
	"github.com/wneessen/waybar-weather/internal/logger"
	"github.com/wneessen/waybar-weather/internal/weather"
	openmeteo "github.com/wneessen/waybar-weather/internal/weather/provider/open-meteo"
)

func (s *Service) selectGeobusProviders() ([]geobus.Provider, error) {
	httpClient := http.New(s.logger)
	var provider []geobus.Provider

	if !s.config.GeoLocation.DisableGeolocationFile {
		provider = append(provider, geolocation_file.NewGeolocationFileProvider(s.config.GeoLocation.GeoLocationFile))
	}

	if !s.config.GeoLocation.DisableGPSD {
		provider = append(provider, gpsd.NewGeolocationGPSDProvider())
	}

	if !s.config.GeoLocation.DisableGeoIP {
		gip, err := geoip.NewGeolocationGeoIPProvider(httpClient)
		if err != nil {
			return nil, fmt.Errorf("failed to create GeoIP provider: %w", err)
		}
		provider = append(provider, gip)
	}

	if !s.config.GeoLocation.DisableGeoAPI {
		gap, err := geoapi.NewGeolocationGeoAPIProvider(httpClient)
		if err != nil {
			return nil, fmt.Errorf("failed to create GeoAPI provider: %w", err)
		}
		provider = append(provider, gap)
	}

	if !s.config.GeoLocation.DisableICHNAEA {
		mls, err := ichnaea.NewGeolocationICHNAEAProvider(httpClient)
		if err != nil {
			s.logger.Error("failed to create ICHNAEA provider", logger.Err(err))
		} else {
			provider = append(provider, mls)
		}
	}
	if len(provider) == 0 {
		return nil, fmt.Errorf("no geolocation providers enabled")
	}

	return provider, nil
}

func (s *Service) selectGeocodeProvider(conf *config.Config, log *logger.Logger, lang language.Tag) (geocode.Geocoder, error) {
	var geocoder geocode.Geocoder

	switch strings.ToLower(conf.GeoCoder.Provider) {
	case "nominatim":
		geocoder = geocode.NewCachedGeocoder(nominatim.New(http.New(log), lang), cacheHitTTL, cacheMissTTL)
	case "opencage":
		if conf.GeoCoder.APIKey == "" {
			return nil, fmt.Errorf("opencage geocoder requires an API key")
		}
		geocoder = geocode.NewCachedGeocoder(opencage.New(http.New(log), lang, conf.GeoCoder.APIKey),
			cacheHitTTL, cacheMissTTL)
	case "geocode-earth":
		if conf.GeoCoder.APIKey == "" {
			return nil, fmt.Errorf("geocode-earth geocoder requires an API key")
		}
		geocoder = geocode.NewCachedGeocoder(geocodeearth.New(http.New(log), lang, conf.GeoCoder.APIKey),
			cacheHitTTL, cacheMissTTL)
	default:
		return nil, fmt.Errorf("unsupported geocoder type: %s", conf.GeoCoder.Provider)
	}

	return geocoder, nil
}

func (s *Service) selectWeatherProvider() (provider weather.Provider, err error) {
	switch strings.ToLower(s.config.Weather.Provider) {
	case "open-meteo":
		provider, err = openmeteo.New(http.New(s.logger), s.logger, s.config.Units)
		if err != nil {
			return provider, fmt.Errorf("failed to create Open-Meteo weather provider: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported weather provider: %s", s.config.Weather.Provider)
	}
	return provider, nil
}
