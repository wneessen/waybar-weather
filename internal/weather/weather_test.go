// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package weather

import (
	"testing"
	"time"
)

func TestNewData(t *testing.T) {
	data := NewData()
	if data == nil {
		t.Fatal("expected data to be non-nil")
	}
	if data.Forecast == nil {
		t.Fatal("expected forecast to be non-nil")
	}
}

func TestNewDayHour(t *testing.T) {
	want := time.Date(2025, 1, 1, 1, 2, 3, 0, time.UTC)
	dayhour := NewDayHour(want)
	if !dayhour.Time().Equal(want.Truncate(time.Hour)) {
		t.Errorf("expected time to be %s, got %s", want, dayhour.Time())
	}
}
