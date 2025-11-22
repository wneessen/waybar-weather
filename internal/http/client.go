// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package http

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"runtime"
	"time"

	"github.com/wneessen/waybar-weather/internal/logger"
)

const (
	// DefaultTimeout is the default timeout value for the HTTPClient
	DefaultTimeout = time.Second * 10
)

var (
	// version is the version of the application (will be set at build time)
	version = "dev"
	// UserAgent is the User-Agent that the HTTP client sends with API requests
	UserAgent = fmt.Sprintf("Mozilla/5.0 (%s; %s) waybar-weather/%s (+https://github.com/wneessen/waybar-weather/)",
		runtime.GOOS,
		runtime.GOARCH,
		version,
	)

	ErrNonPointerTarget = errors.New("target must be a non-nil pointer")
)

// Client is a type wrapper for the Go stdlib http.Client and the Config
type Client struct {
	*http.Client
	logger *logger.Logger
}

// New returns a new HTTP client
func New(logger *logger.Logger) *Client {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	httpTransport := &http.Transport{TLSClientConfig: tlsConfig}
	httpClient := &http.Client{
		Timeout:   DefaultTimeout,
		Transport: httpTransport,
	}
	return &Client{httpClient, logger}
}

// Get performs a HTTP GET request for the given URL and json-unmarshals the response
// into target
func (h *Client) Get(ctx context.Context, endpoint string, target any, query url.Values, headers map[string]string) (int, error) {
	return h.PerformReq(ctx, http.MethodGet, endpoint, target, query, headers, nil, DefaultTimeout)
}

func (h *Client) GetWithTimeout(ctx context.Context, endpoint string, target any, query url.Values, headers map[string]string, timeout time.Duration) (int, error) {
	return h.PerformReq(ctx, http.MethodGet, endpoint, target, query, headers, nil, timeout)
}

// Post performs a HTTP POST request for the given URL and json-unmarshals the response
// into target
func (h *Client) Post(ctx context.Context, endpoint string, target any, body io.Reader, headers map[string]string) (int, error) {
	return h.PerformReq(ctx, http.MethodPost, endpoint, target, nil, headers, body, DefaultTimeout)
}

func (h *Client) PostWithTimeout(ctx context.Context, endpoint string, target any, body io.Reader, headers map[string]string, timeout time.Duration) (int, error) {
	return h.PerformReq(ctx, http.MethodPost, endpoint, target, nil, headers, body, timeout)
}

// PerformReq performs a HTTP GET or POST request for the given URL and timeout and JSON-unmarshals the
// response into target
func (h *Client) PerformReq(ctx context.Context, method string, endpoint string, target any, query url.Values, headers map[string]string, body io.Reader, timeout time.Duration) (int, error) {
	rv := reflect.ValueOf(target)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return 0, ErrNonPointerTarget
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Prepare URL and query parameters
	reqURL, err := url.Parse(endpoint)
	if err != nil {
		return 0, fmt.Errorf("failed to parse URL: %w", err)
	}
	if len(query) > 0 {
		reqURL.RawQuery = query.Encode()
	}

	// Prepare HTTP request
	request, err := http.NewRequestWithContext(ctx, method, reqURL.String(), body)
	if err != nil {
		return 0, fmt.Errorf("failed create new HTTP request with context: %w", err)
	}
	request.Header.Set("User-Agent", UserAgent)
	for k, v := range headers {
		request.Header.Set(k, v)
	}
	// Execute HTTP request
	response, err := h.Do(request)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return 0, err
		}
		return 0, fmt.Errorf("failed to perform HTTP request: %w", err)
	}
	if response == nil {
		return 0, errors.New("nil response received")
	}
	defer func(body io.ReadCloser) {
		if err := body.Close(); err != nil {
			h.logger.Error("failed to close HTTP request body", logger.Err(err))
		}
	}(response.Body)

	// Unmarshal the JSON API response into target
	if err = json.NewDecoder(response.Body).Decode(target); err != nil {
		return response.StatusCode, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return response.StatusCode, nil
}
