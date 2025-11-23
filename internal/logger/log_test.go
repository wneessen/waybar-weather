// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package logger

import (
	"bytes"
	"errors"
	"log/slog"
	"testing"
)

func TestNew(t *testing.T) {
	t.Run("new should successfully create a logger", func(t *testing.T) {
		l := New(slog.LevelInfo)
		if l == nil {
			t.Fatal("expected logger to be non-nil")
		}
	})
}

func TestNewLogger(t *testing.T) {
	t.Run("logger logs successfully with different levels", func(t *testing.T) {
		tests := []struct {
			name        string
			level       slog.Level
			shouldDebug bool
			shouldInfo  bool
			shouldWarn  bool
			shouldError bool
		}{
			{"DEBUG", slog.LevelDebug, true, true, true, true},
			{"INFO", slog.LevelInfo, false, true, true, true},
			{"WARN", slog.LevelWarn, false, false, true, true},
			{"ERROR", slog.LevelError, false, false, false, true},
		}

		for _, tc := range tests {
			buf := bytes.NewBuffer(nil)
			t.Run(tc.name, func(t *testing.T) {
				l := NewLogger(tc.level, buf)
				l.Debug("debug")
				l.Info("info")
				l.Warn("warn")
				l.Error("error")

				if tc.shouldDebug && !bytes.Contains(buf.Bytes(), []byte("debug")) {
					t.Errorf("expected debug message to be logged")
				}
				if !tc.shouldDebug && bytes.Contains(buf.Bytes(), []byte("debug")) {
					t.Errorf("did not expect debug message to be logged")
				}
				if tc.shouldInfo && !bytes.Contains(buf.Bytes(), []byte("info")) {
					t.Errorf("expected info message to be logged")
				}
				if !tc.shouldInfo && bytes.Contains(buf.Bytes(), []byte("info")) {
					t.Errorf("did not expect info message to be logged")
				}
				if tc.shouldWarn && !bytes.Contains(buf.Bytes(), []byte("warn")) {
					t.Errorf("expected warn message to be logged")
				}
				if !tc.shouldWarn && bytes.Contains(buf.Bytes(), []byte("warn")) {
					t.Errorf("did not expect warn message to be logged")
				}
				if tc.shouldError && !bytes.Contains(buf.Bytes(), []byte("error")) {
					t.Errorf("expected error message to be logged")
				}
				if !tc.shouldError && bytes.Contains(buf.Bytes(), []byte("error")) {
					t.Errorf("did not expect error message to be logged")
				}
			})
		}
	})
}

func TestErr(t *testing.T) {
	t.Run("error attributes should be logged", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		l := NewLogger(slog.LevelDebug, buf)
		want := "intentionally failing"
		err := errors.New(want)
		l.Error("this is a test", Err(err))

		if !bytes.Contains(buf.Bytes(), []byte(`error="`+want+`"`)) {
			t.Errorf("expected error message to contain %q, got: %q", want, buf.String())
		}
	})
}
