// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	nominatim "github.com/doppiogancio/go-nominatim"
	"github.com/godbus/dbus/v5"
	"github.com/hectormalot/omgo"
	"github.com/maltegrosse/go-geoclue2"
	"github.com/nathan-osman/go-sunrise"
)

const (
	DBusListNamesAddress = "org.freedesktop.DBus.ListNames"
	GeoclueAgentDBusName = "org.freedesktop.GeoClue2.DemoAgent"
	GeoClueDesktopID     = "waybar-weather"
)

var ErrLocationNotAccurate = errors.New("location service is not accurate enough")

func geoClueAgentIsRunning(ctx context.Context) (isRunning bool, err error) {
	var list []string
	conn, err := dbus.ConnectSessionBus(dbus.WithContext(ctx))
	if err != nil {
		return false, fmt.Errorf("failed to connect to session bus: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to close session bus: %w", closeErr))
		}
	}()

	if err = conn.BusObject().Call(DBusListNamesAddress, 0).Store(&list); err != nil {
		return false, fmt.Errorf("failed to call DBus ListNames: %w", err)
	}

	for _, v := range list {
		if strings.EqualFold(v, GeoclueAgentDBusName) {
			return true, nil
		}
	}
	return false, nil
}

func RegisterGeoClue() (geoclue2.GeoclueClient, error) {
	manager, err := geoclue2.NewGeoclueManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize geoclue manager: %w", err)
	}
	client, err := manager.GetClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get geoclue client: %w", err)
	}
	if err = client.SetDesktopId(GeoClueDesktopID); err != nil {
		return nil, fmt.Errorf("failed to set desktop id: %w", err)
	}
	if err = client.SetRequestedAccuracyLevel(geoclue2.GClueAccuracyLevelExact); err != nil {
		return nil, fmt.Errorf("failed to set requested accuracy availLevel: %w", err)
	}

	return client, nil
}

func (s *Service) geoLocation() (float64, float64, error) {
	location, err := s.geoclient.GetLocation()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get geo location: %w", err)
	}
	accuracy, err := location.GetAccuracy()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get geo location accuracy: %w", err)
	}
	if geoclue2.GClueAccuracyLevel(accuracy) < geoclue2.GClueAccuracyLevelCity {
		return 0, 0, ErrLocationNotAccurate
	}

	latitude, err := location.GetLatitude()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get geo location latitude: %w", err)
	}
	longitude, err := location.GetLongitude()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get geo location longitude: %w", err)
	}

	return latitude, longitude, nil
}

func (s *Service) subscribeLocationUpdates() {
	signal := s.geoclient.SubscribeLocationUpdated()
	for update := range signal {
		_, location, err := s.geoclient.ParseLocationUpdated(update)
		if err != nil {
			s.logger.Error("failed to parse geo location update", logError(err))
			continue
		}

		accuracy, err := location.GetAccuracy()
		if err != nil {
			s.logger.Error("failed to get geo location accuracy", logError(err))
			continue
		}
		if geoclue2.GClueAccuracyLevel(accuracy) < geoclue2.GClueAccuracyLevelCity {
			s.logger.Error("geo location accuracy is too low, skipping location update")
			continue
		}

		latitude, err := location.GetLatitude()
		if err != nil {
			s.logger.Error("failed to get latitude from geo location update", logError(err))
			continue
		}
		longitude, err := location.GetLongitude()
		if err != nil {
			s.logger.Error("failed to get longitude from geo location update", logError(err))
			continue
		}
		if err = s.updateLocation(latitude, longitude); err != nil {
			s.logger.Error("failed to update service geo location", logError(err))
		}
	}
}

func (s *Service) updateLocation(latitude, longitude float64) error {
	s.locationLock.Lock()
	defer s.locationLock.Unlock()
	if latitude <= 0 || longitude <= 0 {
		s.logger.Debug("coordinates empty, skipping service geo location update")
		return nil
	}

	address, err := nominatim.ReverseGeocode(latitude, longitude, s.config.Locale)
	if err != nil {
		return fmt.Errorf("failed reverse geocode coordinates: %w", err)
	}
	location, err := omgo.NewLocation(latitude, longitude)
	if err != nil {
		return fmt.Errorf("failed create Open-Meteo location from coordinates: %w", err)
	}
	s.address = address
	s.location = location
	s.logger.Debug("geo location successfully updated",
		slog.Any("address", s.address),
		slog.Any("location", s.location),
	)

	now := time.Now()
	sunriseTime, sunsetTime := sunrise.SunriseSunset(latitude, longitude, now.Year(), now.Month(), now.Day())
	s.sunriseTime = sunriseTime
	s.sunsetTime = sunsetTime
	s.isDayTime = false
	if now.After(sunriseTime) && now.Before(sunsetTime) {
		s.isDayTime = true
	}

	for _, job := range s.scheduler.Jobs() {
		if err = job.RunNow(); err != nil {
			s.logger.Error("failed to run scheduled job", logError(err))
		}
	}

	return nil
}
