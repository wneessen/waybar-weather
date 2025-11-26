// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"os"
	"os/signal"
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
		case <-sigChan:
			s.displayAltLock.Lock()
			s.displayAltText = !s.displayAltText
			s.displayAltLock.Unlock()
			s.printWeather(ctx)
		}
	}
}
