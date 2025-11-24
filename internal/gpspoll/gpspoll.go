// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package gpspoll

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"time"
)

const (
	fallbackAccuracy3DFix = 10  // ~10 m typical consumer GPS in open sky
	fallbackAccuracy2DFix = 25  // worse than 3D, but still accurate enough
	fallbackAccuracyNoFix = 1e6 // effectively unusable
	watchTimeout          = time.Second * 2
)

// Client is a minimal GPSd client
type Client struct {
	Addr string
}

// Fix represents a single GPS fix from gpsd.
type Fix struct {
	Lat  float64
	Lon  float64
	Alt  float64
	Acc  float64
	Mode int
}

// gpsdPollResponse matches the subset of gpsd's POLL response we care about.
type gpsdPollResponse struct {
	Class string  `json:"class"`
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
	Alt   float64 `json:"alt"`
	Acc   float64
	Mode  int     `json:"mode"`
	Epx   float64 `json:"epx"`
	Epy   float64 `json:"epy"`
	Eph   float64 `json:"eph"`
	Epv   float64 `json:"epv"`
}

// NewClient constructs a new Client for the given host and port.
func NewClient(host, port string) *Client {
	return &Client{
		Addr: net.JoinHostPort(host, port),
	}
}

// Poll connects to gpsd, sends a POLL request, and returns the first TPV
// entry from the POLL response. The connection is closed before returning.
func (c *Client) Poll(ctx context.Context) (Fix, error) {
	var zero Fix

	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", c.Addr)
	if err != nil {
		return zero, fmt.Errorf("gpspoll: dial gpsd: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	// Respect context deadline if present, otherwise we add a safety net so we don't hang
	// forever if ctx has no deadline.
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(watchTimeout))
	}

	// Request a WATCH.
	if _, err = fmt.Fprint(conn, `?WATCH={"enable":true,"json":true}`+"\n"); err != nil {
		return zero, fmt.Errorf("gpspoll: write POLL: %w", err)
	}

	// Wait for a TPV response or timeout.
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		var resp gpsdPollResponse

		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		default:
		}

		line := scanner.Bytes()
		if err = json.Unmarshal(line, &resp); err != nil {
			continue
		}
		if resp.Class != "TPV" {
			continue
		}

		return Fix{
			Lat:  resp.Lat,
			Lon:  resp.Lon,
			Alt:  resp.Alt,
			Acc:  horizontalAccuracyMeters(resp),
			Mode: resp.Mode,
		}, nil
	}

	if err = scanner.Err(); err != nil {
		return zero, fmt.Errorf("failed to scan GPSd response: %w", err)
	}

	return zero, fmt.Errorf("no TPV response received from GPSd")
}

// Has2DFix reports whether the fix has at least a 2D fix.
func (f Fix) Has2DFix() bool {
	return f.Mode >= 2
}

func horizontalAccuracyMeters(tpv gpsdPollResponse) float64 {
	switch {
	case tpv.Eph > 0:
		return tpv.Eph
	case tpv.Epx > 0 && tpv.Epy > 0:
		// sqrt(epx² + epy²)
		return math.Hypot(tpv.Epx, tpv.Epy)
	default:
		return horizontalAccuracyFallback(tpv)
	}
}

func horizontalAccuracyFallback(tpv gpsdPollResponse) float64 {
	switch tpv.Mode {
	case 3:
		return fallbackAccuracy3DFix
	case 2:
		return fallbackAccuracy2DFix
	default:
		return fallbackAccuracyNoFix
	}
}
