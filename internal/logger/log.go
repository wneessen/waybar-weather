// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package logger

import (
	"io"
	"log/slog"
	"os"
)

var defaultLogOutput = os.Stderr

type Logger struct {
	*slog.Logger
}

func New(level slog.Level) *Logger {
	return NewLogger(level, nil)
}

func NewLogger(level slog.Level, logFile io.Writer) *Logger {
	multiLogger := make([]slog.Handler, 0)
	defaultLogger := slog.NewTextHandler(defaultLogOutput, &slog.HandlerOptions{Level: level})
	multiLogger = append(multiLogger, defaultLogger)

	if logFile != nil {
		fileLogger := slog.NewJSONHandler(logFile, &slog.HandlerOptions{Level: level})
		multiLogger = append(multiLogger, fileLogger)
	}

	logger := slog.New(slog.NewMultiHandler(multiLogger...))
	return &Logger{Logger: logger}
}

func Err(err error) slog.Attr {
	return slog.Any("error", err)
}
