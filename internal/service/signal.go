// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package service

import (
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
