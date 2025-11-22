// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package testhelper

import (
	"bytes"
	"io"
	"log/slog"
	stdhttp "net/http"
	"testing"

	"github.com/wneessen/waybar-weather/internal/http"
	"github.com/wneessen/waybar-weather/internal/logger"
)

func TestPerformIntegrationTests(t *testing.T) {
	t.Run("perform integration tests", func(t *testing.T) {
		t.Setenv("PERFORM_INTEGRATION_TEST", "true")
		PerformIntegrationTests(t)
	})
	t.Run("skip integration tests", func(t *testing.T) {
		t.Setenv("PERFORM_INTEGRATION_TEST", "false")
		PerformIntegrationTests(t)
	})
}

func TestMockRoundTripper(t *testing.T) {
	PerformIntegrationTests(t)
	t.Run("perform a mocked http request", func(t *testing.T) {
		rtFn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
			return &stdhttp.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString("{}")),
				Header:     make(stdhttp.Header),
			}, nil
		}

		type testType struct{}
		target := new(testType)
		client := http.New(logger.NewLogger(slog.LevelInfo, io.Discard))
		client.Transport = MockRoundTripper{Fn: rtFn}
		_, err := client.Get(t.Context(), TestOnlineAPIURL, target, nil, nil)
		if err != nil {
			t.Fatalf("http request failed: %s", err)
		}
	})
}
