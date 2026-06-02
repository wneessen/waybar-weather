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
	return NewLogger(level, nil, nil)
}

func NewLogger(level slog.Level, textTarget io.Writer, jsonTarget io.Writer) *Logger {
	multiLogger := make([]slog.Handler, 0)

	if textTarget == nil {
		textTarget = defaultLogOutput
	}
	defaultLogger := slog.NewTextHandler(textTarget, &slog.HandlerOptions{Level: level})
	multiLogger = append(multiLogger, defaultLogger)

	if jsonTarget != nil {
		fileLogger := slog.NewJSONHandler(jsonTarget, &slog.HandlerOptions{Level: level})
		multiLogger = append(multiLogger, fileLogger)
	}

	logger := slog.New(slog.NewMultiHandler(multiLogger...))
	return &Logger{Logger: logger}
}

func Err(err error) slog.Attr {
	return slog.Any("error", err)
}
