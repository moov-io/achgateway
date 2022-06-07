// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package schedule

import (
	"testing"
	"time"

	"github.com/moov-io/base"
	"github.com/moov-io/base/stime"

	"github.com/stretchr/testify/require"
)

func TestCutoffTimes(t *testing.T) {
	if testing.Short() {
		t.Skip("this test can take up to 60s, skipping")
	}
	if now := base.Now(time.UTC); now.IsWeekend() || !now.IsBankingDay() {
		t.Skip("not a banking day")
	}

	timeService := stime.NewSystemTimeService()
	next := time.Now().UTC().Add(time.Minute).Format("15:04")

	cutoffs, err := ForCutoffTimes(timeService, "UTC", []string{next})
	require.NoError(t, err)
	defer cutoffs.Stop()

	day := <-cutoffs.C // block on channel read

	expected := day.Time.Format("15:04")
	if next != expected {
		t.Errorf("next=%q expected=%q", next, expected)
	}

	require.True(t, day.FirstWindow)
}

func TestCutoffTimesErr(t *testing.T) {
	timeService := stime.NewSystemTimeService()

	_, err := ForCutoffTimes(timeService, "bad_zone", nil)
	if err == nil {
		t.Error("expected error")
	}
	_, err = ForCutoffTimes(timeService, time.Local.String(), nil)
	if err == nil {
		t.Error("expected error")
	}
	_, err = ForCutoffTimes(timeService, time.Local.String(), []string{"bad:time"})
	if err == nil {
		t.Error("expected error")
	}
}

func TestCutoffTimes__firstCutoff(t *testing.T) {
	timeService := stime.NewSystemTimeService()
	ct, err := ForCutoffTimes(timeService, "America/New_York", []string{"16:15", "08:30", "12:00"})
	require.NoError(t, err)
	defer ct.Stop()

	require.Equal(t, "08:30", ct.firstCutoff)
}

func TestCutoffTimes__Holiday(t *testing.T) {
	holiday := time.Date(2022, time.July, 4, 15, 30, 0, 0, time.UTC)

	timeService := stime.NewStaticTimeService()
	timeService.Change(holiday)

	ct, err := ForCutoffTimes(timeService, "America/New_York", []string{"15:30"})
	require.NoError(t, err)
	defer ct.Stop()

	go ct.maybeTick(time.UTC)

	cutoff := <-ct.C
	require.Equal(t, holiday, cutoff.Time)
	require.False(t, cutoff.IsBankingDay)
	require.True(t, cutoff.IsHoliday)
	require.False(t, cutoff.IsWeekend)
	require.True(t, cutoff.FirstWindow)
}
