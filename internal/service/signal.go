// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"fmt"
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
			// USR2 prints the current address to stderr
			case syscall.SIGUSR2:
				s.locationLock.Lock()
				address := s.address
				s.locationLock.Unlock()
				_, _ = fmt.Fprintf(os.Stderr, "Current address: %s\n", address.DisplayName)
			}
		}
	}
}
