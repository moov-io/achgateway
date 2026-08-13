// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package schedule

import (
	"fmt"
	"slices"
	"time"

	"github.com/moov-io/base"
)

// NextCutoff returns the next automated cutoff that should upload a file queued at now.
//
// Predictions follow the same rules as automated cutoff processing:
//   - Windows are compared in timezone as HH:MM.
//   - A file queued at an exact window time is scheduled for a later window.
//   - on == "all-days" includes holidays and still skips weekends
//     (the scheduler never ticks on Saturday/Sunday).
//   - Any other on value (including the default) uses banking days.
//
// An empty windows list returns a zero time. Invalid timezone or window values return an error.
func NextCutoff(timezone string, windows []string, on string, now time.Time) (time.Time, error) {
	if len(windows) == 0 {
		return time.Time{}, nil
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, fmt.Errorf("loading %s failed: %w", timezone, err)
	}

	cutoffs := copysort(windows)
	dt := base.NewTime(now.In(loc))

	if isProcessingDay(on, dt) {
		nowHHMM := dt.Format("15:04")
		for _, ct := range cutoffs {
			if nowHHMM < ct {
				return adjustNowToCutoff(ct, dt.Time, loc)
			}
		}
	}

	next, err := nextProcessingDay(on, dt)
	if err != nil {
		return time.Time{}, err
	}
	return adjustNowToCutoff(cutoffs[0], next.Time, loc)
}

func isProcessingDay(on string, dt base.Time) bool {
	if on == "all-days" {
		return !dt.IsWeekend()
	}
	return dt.IsBankingDay()
}

func nextProcessingDay(on string, dt base.Time) (base.Time, error) {
	if on != "all-days" {
		return dt.AddBankingDay(1), nil
	}

	// Weekends never fire automated cutoffs, even when On=all-days.
	for i := 0; i < 7; i++ {
		dt = base.NewTime(dt.Time.AddDate(0, 0, 1))
		if !dt.IsWeekend() {
			return dt, nil
		}
	}
	return base.Time{}, fmt.Errorf("unable to find next cutoff after %s", dt.Format(time.RFC3339))
}

func adjustNowToCutoff(cutoff string, now time.Time, loc *time.Location) (time.Time, error) {
	c, err := time.Parse("15:04", cutoff)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing %s failed: %w", cutoff, err)
	}

	out := now.In(loc)
	return time.Date(out.Year(), out.Month(), out.Day(), c.Hour(), c.Minute(), 0, 0, out.Location()), nil
}

func copysort(input []string) []string {
	out := make([]string, len(input))
	copy(out, input)
	slices.Sort(out)
	return out
}
