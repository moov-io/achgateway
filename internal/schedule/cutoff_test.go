// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package schedule

import (
	"testing"
	"time"

	"github.com/moov-io/base"
	"github.com/stretchr/testify/require"
)

func TestCutoffTimes(t *testing.T) {
	if testing.Short() {
		t.Skip("this test can take up to 60s, skipping")
	}
	if now := base.Now(time.UTC); now.IsWeekend() || !now.IsBankingDay() {
		t.Skip("not a banking day")
	}

	next := time.Now().UTC().Add(time.Minute).Format("15:04")

	cutoffs, err := ForCutoffTimes("UTC", []string{next})
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
	_, err := ForCutoffTimes("bad_zone", nil)
	if err == nil {
		t.Error("expected error")
	}
	_, err = ForCutoffTimes(time.Local.String(), nil)
	if err == nil {
		t.Error("expected error")
	}
	_, err = ForCutoffTimes(time.Local.String(), []string{"bad:time"})
	if err == nil {
		t.Error("expected error")
	}
}

func TestCutoffTimes__firstCutoff(t *testing.T) {
	ct, err := ForCutoffTimes("America/New_York", []string{"16:15", "08:30", "12:00"})
	require.NoError(t, err)
	defer ct.Stop()

	require.Equal(t, "08:30", ct.firstCutoff)
}
