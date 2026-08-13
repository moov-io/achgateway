// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package schedule

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNextCutoff(t *testing.T) {
	type check struct {
		now      time.Time
		expected time.Time
	}

	eastern, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	central, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	cases := []struct {
		name     string
		timezone string
		windows  []string
		on       string
		checks   []check
	}{
		{
			name:     "banking-days",
			timezone: "America/New_York",
			windows:  []string{"10:00", "14:15", "16:15"},
			checks: []check{
				{
					// before 16:15 cutoff
					now:      time.Date(2025, time.January, 7, 13, 37, 0, 0, central),
					expected: time.Date(2025, time.January, 7, 16, 15, 0, 0, eastern),
				},
				{
					// exactly 14:15 ET rolls to the later window
					now:      time.Date(2025, time.January, 7, 13, 15, 0, 0, central),
					expected: time.Date(2025, time.January, 7, 16, 15, 0, 0, eastern),
				},
				{
					// before first cutoff
					now:      time.Date(2025, time.January, 7, 8, 37, 0, 0, central),
					expected: time.Date(2025, time.January, 7, 10, 0, 0, 0, eastern),
				},
				{
					// after last cutoff
					now:      time.Date(2025, time.January, 7, 16, 16, 0, 0, central),
					expected: time.Date(2025, time.January, 8, 10, 0, 0, 0, eastern),
				},
			},
		},
		{
			name:     "weekend-to-monday",
			timezone: "America/New_York",
			windows:  []string{"17:30"},
			checks: []check{
				{
					now:      time.Date(2025, time.September, 6, 2, 0, 35, 0, central),
					expected: time.Date(2025, time.September, 8, 17, 30, 0, 0, eastern),
				},
				{
					now:      time.Date(2025, time.September, 6, 18, 16, 0, 0, central),
					expected: time.Date(2025, time.September, 8, 17, 30, 0, 0, eastern),
				},
			},
		},
		{
			name:     "holiday-banking-days",
			timezone: "America/New_York",
			windows:  []string{"15:30"},
			checks: []check{
				{
					// Monday July 4 2022 is Independence Day
					now:      time.Date(2022, time.July, 4, 10, 0, 0, 0, eastern),
					expected: time.Date(2022, time.July, 5, 15, 30, 0, 0, eastern),
				},
			},
		},
		{
			name:     "holiday-all-days",
			timezone: "America/New_York",
			windows:  []string{"15:30"},
			on:       "all-days",
			checks: []check{
				{
					now:      time.Date(2022, time.July, 4, 10, 0, 0, 0, eastern),
					expected: time.Date(2022, time.July, 4, 15, 30, 0, 0, eastern),
				},
			},
		},
		{
			name:     "friday-into-labor-day-banking-days",
			timezone: "America/New_York",
			windows:  []string{"17:30"},
			checks: []check{
				{
					// Friday after last window, Monday is Labor Day 2025
					now:      time.Date(2025, time.August, 29, 18, 0, 0, 0, eastern),
					expected: time.Date(2025, time.September, 2, 17, 30, 0, 0, eastern),
				},
			},
		},
		{
			name:     "friday-into-labor-day-all-days",
			timezone: "America/New_York",
			windows:  []string{"17:30"},
			on:       "all-days",
			checks: []check{
				{
					// Weekends still do not process; Labor Day Monday does
					now:      time.Date(2025, time.August, 29, 18, 0, 0, 0, eastern),
					expected: time.Date(2025, time.September, 1, 17, 30, 0, 0, eastern),
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for idx, chk := range tc.checks {
				got, err := NextCutoff(tc.timezone, tc.windows, tc.on, chk.now)
				require.NoError(t, err)

				expected := chk.expected.Format(time.RFC3339)
				desc := fmt.Sprintf("checks[%d] windows=%s now=%s", idx, strings.Join(tc.windows, ","), chk.now.Format(time.RFC3339))
				require.Equal(t, expected, got.Format(time.RFC3339), desc)
			}
		})
	}
}

func TestNextCutoff_EmptyWindows(t *testing.T) {
	got, err := NextCutoff("America/New_York", nil, "", time.Now())
	require.NoError(t, err)
	require.True(t, got.IsZero())
}

func TestNextCutoff_InvalidTimezone(t *testing.T) {
	_, err := NextCutoff("not/a/zone", []string{"10:00"}, "", time.Now())
	require.Error(t, err)
}

func TestNextCutoff_InvalidWindow(t *testing.T) {
	_, err := NextCutoff("America/New_York", []string{"noon"}, "", time.Date(2025, time.January, 7, 9, 0, 0, 0, time.UTC))
	require.Error(t, err)
}
