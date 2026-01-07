// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

type signalSource interface {
	Notify(c chan<- os.Signal, sig ...os.Signal)
	Stop(c chan<- os.Signal)
}

// RealSignalSource is the production implementation.
type stdLibSignalSource struct{}

func (stdLibSignalSource) Notify(c chan<- os.Signal, sig ...os.Signal) {
	signal.Notify(c, sig...)
}

func (stdLibSignalSource) Stop(c chan<- os.Signal) {
	signal.Stop(c)
}

// HandleAltTextToggleSignal toggles the module text display when a signal is received
func (s *Service) HandleAltTextToggleSignal(ctx context.Context, sigChan chan os.Signal) {
	for {
		select {
		case <-ctx.Done():
			return
		case sig := <-sigChan:
			switch sig {
			// USR1 toggles between displaying the text and the alt text
			case syscall.SIGUSR1:
				s.displayAltLock.Lock()
				s.displayAltText = !s.displayAltText
				s.displayAltLock.Unlock()
				s.printWeather(ctx)
			// USR2 prints the current address with the stderr logger
			case syscall.SIGUSR2:
				s.locationLock.Lock()
				address := s.address
				s.locationLock.Unlock()
				s.logger.Info("currently resolved address", slog.String("address", address.DisplayName),
					slog.Float64("latitude", address.Latitude), slog.Float64("longitude", address.Longitude))
			}
		}
	}
}
