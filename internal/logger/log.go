// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package logger

import (
	"io"
	"log/slog"
	"os"
)

type Logger struct {
	*slog.Logger
}

func New(level slog.Level) *Logger {
	return NewLogger(level, os.Stderr)
}

func NewLogger(level slog.Level, output io.Writer) *Logger {
	return &Logger{slog.New(slog.NewTextHandler(output, &slog.HandlerOptions{Level: level}))}
}

func Err(err error) slog.Attr {
	return slog.Any("error", err)
}
