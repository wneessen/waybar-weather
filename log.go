// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package main

import (
	"log/slog"
	"os"
)

type logger struct {
	*slog.Logger
}

func newLogger(level slog.Level) *logger {
	output := os.Stderr
	return &logger{slog.New(slog.NewTextHandler(output, &slog.HandlerOptions{Level: level}))}
}

func logError(err error) slog.Attr {
	return slog.Any("error", err)
}
