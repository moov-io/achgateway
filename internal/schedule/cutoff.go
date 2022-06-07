// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package schedule

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/moov-io/base"
	"github.com/moov-io/base/stime"

	"github.com/robfig/cron/v3"
)

// CutoffTimes is a time.Ticker which fires on banking days to trigger processing
// events (like end-of-day, or same-day ACH).
type CutoffTimes struct {
	C chan *Day

	sched       *cron.Cron
	firstCutoff string
	timeService stime.TimeService
}

type Day struct {
	Time time.Time

	IsBankingDay bool
	IsHoliday    bool
	IsWeekend    bool

	// FirstWindow is true when Time is the first cutoff time of the day
	FirstWindow bool
}

func ForCutoffTimes(timeService stime.TimeService, tz string, timestamps []string) (*CutoffTimes, error) {
	ct := &CutoffTimes{
		C:           make(chan *Day),
		sched:       cron.New(),
		timeService: timeService,
	}

	if len(timestamps) > 0 {
		sort.Strings(timestamps)
		ct.firstCutoff = timestamps[0]
	}

	if err := ct.registerCutoffs(tz, timestamps); err != nil {
		return nil, err
	}
	ct.sched.Start()
	return ct, nil
}

func (ct *CutoffTimes) Stop() {
	if ct == nil {
		return
	}
	if ct.C != nil {
		close(ct.C)
	}
	if ct.sched != nil {
		ct.sched.Stop()
	}
}

func (ct *CutoffTimes) maybeTick(location *time.Location) {
	now := base.NewTime(ct.timeService.Now().In(location))
	if !now.IsWeekend() {
		ct.C <- &Day{
			Time:         now.Time,
			IsBankingDay: now.IsBankingDay(),
			IsHoliday:    now.IsHoliday(),
			IsWeekend:    now.IsWeekend(),
			FirstWindow:  now.Format("15:04") == ct.firstCutoff,
		}
	}
}

func (ct *CutoffTimes) registerCutoffs(tz string, timestamps []string) error {
	if len(timestamps) == 0 {
		return errors.New("missing cutoff times")
	}
	for i := range timestamps {
		if err := ct.register(tz, timestamps[i]); err != nil {
			return fmt.Errorf("timestamp=%s error=%v", timestamps[i], err)
		}
	}
	return nil
}

func (ct *CutoffTimes) register(tz string, timestamp string) error {
	when, err := time.Parse("15:04", timestamp)
	if err != nil {
		return fmt.Errorf("failed to parse '%s' error=%v", timestamp, err)
	}

	var zone string
	var location *time.Location
	if tz != "" {
		zone = fmt.Sprintf("CRON_TZ=%s", tz)
		l, _ := time.LoadLocation(tz)
		location = l
	} else {
		location = time.UTC
	}
	schedule := fmt.Sprintf(`%s %d %d * * *`, zone, when.Minute(), when.Hour())
	ct.sched.AddFunc(schedule, func() {
		ct.maybeTick(location)
	})

	return nil
}
