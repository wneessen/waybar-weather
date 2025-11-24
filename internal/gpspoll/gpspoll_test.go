// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package gpspoll

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"net"
	"sync"
	"testing"
	"time"
)

const (
	tvpFull = `{"class":"TPV","device":"/dev/ttyACM0","mode":3,"time":"2025-11-24T10:44:41.000Z","leapseconds":18,"ept":0.005,"lat":51.000000000,"lon":7.000000000,"altHAE":120.0000,"altMSL":75.0000,"alt":75.0000,"epx":8.100,"epy":11.400,"epv":27.600,"track":332.6961,"magtrack":334.8207,"magvar":2.1,"speed":0.229,"climb":-0.217,"eps":1.02,"epc":55.20,"ecefx":3980000.00,"ecefy":500000.00,"ecefz":4930000.00,"ecefvx":-0.28,"ecefvy":-0.14,"ecefvz":-0.04,"ecefpAcc":15.28,"ecefvAcc":1.02,"velN":0.204,"velE":-0.105,"velD":0.217,"geoidSep":46.037,"eph":17.670,"sep":28.880}`
)

func TestNewClient(t *testing.T) {
	client := New("localhost", "2497")
	if client == nil {
		t.Fatal("expected client to be non-nil")
	}
	if client.Addr != "localhost:2497" {
		t.Errorf("expected client address to be localhost:2497, got %s", client.Addr)
	}
}

func TestClient_Poll(t *testing.T) {
	t.Run("poll succeeds with different TPV results", func(t *testing.T) {
		tests := []struct {
			name string
			tpv  string
			lat  float64
			lon  float64
			acc  float64
			mode int
		}{
			{
				"full response",
				tvpFull,
				51, 7, 17.67, 3,
			},
			{
				"no Eph use Epx/Epy",
				`{"class":"TPV","device":"/dev/ttyACM0","mode":3,"time":"2025-11-24T10:44:41.000Z","lat":51.0,"lon":7.0,"alt":75.0000,"epx":8.100,"epy":11.400}`,
				51, 7, math.Hypot(8.100, 11.400), 3,
			},
			{
				"no Eph, Epx and Epy - fallback to 3d fix accuracy",
				`{"class":"TPV","device":"/dev/ttyACM0","mode":3,"time":"2025-11-24T10:44:41.000Z","lat":51.0,"lon":7.0,"alt":75.0000}`,
				51, 7, fallbackAccuracy3DFix, 3,
			},
			{
				"no Eph, Epx and Epy - fallback to 2d fix accuracy",
				`{"class":"TPV","device":"/dev/ttyACM0","mode":2,"time":"2025-11-24T10:44:41.000Z","lat":51.0,"lon":7.0,"alt":75.0000}`,
				51, 7, fallbackAccuracy2DFix, 2,
			},
			{
				"no accuracy information at all",
				`{"class":"TPV","device":"/dev/ttyACM0","mode":1,"time":"2025-11-24T10:44:41.000Z","lat":51.0,"lon":7.0,"alt":75.0000}`,
				51, 7, fallbackAccuracyNoFix, 1,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				addr := startMockGPSD(t.Context(), t, tc.tpv)
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					t.Fatalf("failed to parse mock gpsd address: %v", err)
				}
				client := New(host, port)
				fix, err := client.Poll(t.Context())
				if err != nil {
					t.Fatalf("failed to poll for fix: %v", err)
				}
				if fix.Lat != tc.lat {
					t.Errorf("expected latitude to be %f, got %f", tc.lat, fix.Lat)
				}
				if fix.Lon != tc.lon {
					t.Errorf("expected longitude to be %f, got %f", tc.lon, fix.Lon)
				}
				if fix.Acc != tc.acc {
					t.Errorf("expected accuracy to be %f, got %f", tc.acc, fix.Acc)
				}
				if fix.Mode != tc.mode {
					t.Errorf("expected mode to be %d, got %d", tc.mode, fix.Mode)
				}
			})
		}
	})
	t.Run("poll with a canceled context", func(t *testing.T) {
		addr := startMockGPSD(t.Context(), t, tvpFull)

		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			t.Fatalf("failed to parse mock gpsd address: %v", err)
		}

		ctxPoll, ctxCancel := context.WithCancel(t.Context())
		client := New(host, port)
		ctxCancel()
		_, err = client.Poll(ctxPoll)
		if err == nil {
			t.Fatal("expected Poll() to fail with context canceled")
		}
	})
	t.Run("poll with with broken JSON returned", func(t *testing.T) {
		addr := startMockGPSD(t.Context(), t, "invalid")

		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			t.Fatalf("failed to parse mock gpsd address: %v", err)
		}

		client := New(host, port)
		_, err = client.Poll(t.Context())
		if err == nil {
			t.Fatal("expected Poll() to fail on broken JSON")
		}
	})
}

func TestFix_Has2DFix(t *testing.T) {
	fix := Fix{Mode: 1}
	if fix.Has2DFix() {
		t.Error("expected Has2DFix() to return false for mode 1")
	}
	fix = Fix{Mode: 2}
	if !fix.Has2DFix() {
		t.Error("expected Has2DFix() to return true for mode 2")
	}
	fix = Fix{Mode: 3}
	if !fix.Has2DFix() {
		t.Error("expected Has2DFix() to return true for mode 3")
	}
}

func startMockGPSD(ctx context.Context, t *testing.T, tpv string) string {
	t.Helper()

	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to listen for mock gpsd: %v", err)
	}

	addr := ln.Addr().String()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		// Wait for either an incoming connection or context cancellation.
		connChan := make(chan net.Conn, 1)
		errChan := make(chan error, 1)

		go func() {
			conn, err := ln.Accept()
			if err != nil {
				errChan <- err
				return
			}
			connChan <- conn
		}()

		select {
		case <-ctx.Done():
			// Context canceled before any connection – exit cleanly.
			return

		case err := <-errChan:
			// Listener closed or accept error.
			_ = err
			return

		case conn := <-connChan:
			// We got a client connection.
			handleMockGPSDConnection(ctx, conn, t, tpv)
		}
	}()

	// Make the test wait for the goroutine to fully exit on cleanup
	t.Cleanup(func() {
		if closeErr := ln.Close(); closeErr != nil {
			t.Logf("failed to close mock gpsd listener: %s", closeErr)
		}
		wg.Wait()
	})

	return addr
}

func handleMockGPSDConnection(ctx context.Context, conn net.Conn, t *testing.T, tpv string) {
	go func() {
		<-ctx.Done()
		if closeErr := conn.Close(); closeErr != nil {
			t.Errorf("failed to close mock gpsd connection: %s", closeErr)
		}
	}()

	_ = conn.SetReadDeadline(time.Now().Add(time.Millisecond * 200))
	_, _ = bufio.NewReader(conn).ReadString('\n')

	// Remove read deadline so writes work normally.
	_ = conn.SetReadDeadline(time.Time{})

	// Return some mock data.
	_, err := fmt.Fprintln(conn, `{"class":"VERSION","release":"gpsd 3.26","proto_major":3,"proto_minor":14}`)
	if err != nil {
		t.Logf("failed to write mock gpsd version: %s", err)
	}
	_, err = fmt.Fprintln(conn, `{"class":"DEVICES","devices":[{"class":"DEVICE","path":"/dev/ttyACM0","driver":"MockGPS","activated":"2025-11-24T10:40:00.000Z","native":0}]}`)
	if err != nil {
		t.Logf("failed to write mock gpsd devices: %s", err)
	}
	_, err = fmt.Fprintln(conn, tpv)
	if err != nil {
		t.Logf("failed to write mock gpsd response: %s", err)
	}
}
