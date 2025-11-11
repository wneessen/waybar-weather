// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/godbus/dbus/v5"

	"github.com/wneessen/waybar-weather/internal/logger"
)

const (
	dbusInterface   = "org.freedesktop.login1.Manager"
	dbusWatchMember = "PrepareForSleep"

	debounceWindow   = 2 // seconds
	signalBufferSize = 8

	busReconnectDelay   = 5 * time.Second
	networkWakeupDelay  = 10 * time.Second
	reconnectDelay      = 2 * time.Second
	subscribeRetryDelay = 10 * time.Second
)

// monitorSleepResume monitors system sleep and resume events using D-Bus signals and handles
// reconnections as needed.
func (s *Service) monitorSleepResume(ctx context.Context) {
	var lastResumeUnix int64

	for {
		conn := s.connectToSystemBus(ctx)
		if conn == nil {
			return // the context was cancelled, exit
		}

		// try to reconnect or exit if we can't if the context was cancelled
		if !s.setupSleepMonitoring(ctx, conn) {
			continue
		}

		sigCh := make(chan *dbus.Signal, signalBufferSize)
		conn.Signal(sigCh)
		s.logger.Debug("subscribed to dbus signal", slog.String("interface", dbusInterface),
			slog.String("member", dbusWatchMember))

		s.handleSleepSignals(ctx, sigCh, &lastResumeUnix)

		// Clean up before reconnect
		conn.RemoveSignal(sigCh)
		if err := conn.Close(); err != nil {
			s.logger.Error("failed to close system bus connection", logger.Err(err))
		}

		// If we're here because of ctx cancel, exit; otherwise reconnect
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(reconnectDelay)
		}
	}
}

// connectToSystemBus establishes a connection to the system D-Bus with automatic reconnection handling
// on failure. It continuously retries on connection failures until the provided context is canceled.
// On context cancellation, it ensures the connection is cleanly closed.
func (s *Service) connectToSystemBus(ctx context.Context) *dbus.Conn {
	for {
		conn, err := dbus.ConnectSystemBus()
		if err != nil {
			select {
			case <-time.After(busReconnectDelay):
				continue
			case <-ctx.Done():
				return nil
			}
		}

		// Ensure cleanup on context cancellation
		go func() {
			<-ctx.Done()
			if err := conn.Close(); err != nil {
				s.logger.Error("failed to close system bus connection", logger.Err(err))
			}
		}()

		return conn
	}
}

// setupSleepMonitoring configures sleep monitoring by subscribing to specific dbus signals and
// handles error retries.
func (s *Service) setupSleepMonitoring(ctx context.Context, conn *dbus.Conn) bool {
	if err := conn.AddMatchSignal(dbus.WithMatchInterface(dbusInterface),
		dbus.WithMatchMember(dbusWatchMember),
	); err != nil {
		s.logger.Error("failed to subscribe to dbus signal", slog.String("interface", dbusInterface),
			slog.String("member", dbusWatchMember), logger.Err(err))
		if err = conn.Close(); err != nil {
			s.logger.Error("failed to close system bus connection", logger.Err(err))
		}
		select {
		case <-time.After(subscribeRetryDelay):
			return false
		case <-ctx.Done():
			return false
		}
	}
	return true
}

// handleSleepSignals listens for sleep-related signals and processes them accordingly using the
// provided signal channel. Takes a context to handle cancellation, a signal channel for receiving
// dbus signals, and a timestamp pointer for updates.
func (s *Service) handleSleepSignals(ctx context.Context, sigCh chan *dbus.Signal, lastResumeUnix *int64) {
	for {
		select {
		case <-ctx.Done():
			return
		case sgn, ok := <-sigCh:
			if !ok {
				// connection likely closed; try to reconnect
				return
			}
			s.processSleepSignal(ctx, sgn, lastResumeUnix)
		}
	}
}

// processSleepSignal handles the sleep signal received from dbus and triggers resume event processing
// if conditions are met.
func (s *Service) processSleepSignal(ctx context.Context, sgn *dbus.Signal, lastResumeUnix *int64) {
	if len(sgn.Body) != 1 {
		return
	}
	sleeping, ok := sgn.Body[0].(bool)
	if !ok || sleeping {
		return
	}
	s.handleResumeEvent(ctx, lastResumeUnix)
}

// handleResumeEvent handles the system wake-up event and triggers necessary actions to refresh weather data.
// It ensures debouncing of multiple consecutive resume events and provides time for network readiness.
func (s *Service) handleResumeEvent(ctx context.Context, lastResumeUnix *int64) {
	now := time.Now().Unix()

	// debounce in case of multiple resume events
	if now-atomic.LoadInt64(lastResumeUnix) < debounceWindow {
		return
	}
	atomic.StoreInt64(lastResumeUnix, now)

	// Give the system time to wake up and establish network connection
	time.Sleep(networkWakeupDelay)

	s.logger.Debug("resuming from sleep, fetching latest weather data")
	s.fetchWeather(ctx)
}
