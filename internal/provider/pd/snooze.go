package pd

import (
	"errors"
	"fmt"
	"time"

	"github.com/adamdecaf/deadcheck/internal/config"

	"github.com/moov-io/base"
	"github.com/moov-io/base/stime"
)

func calculateSnooze(timeService stime.TimeService, check config.Check) (time.Duration, error) {
	now := timeService.Now()

	switch {
	case check.Schedule.Every != nil:
		// Relative check-ins are snoozed for their interval + tolerance from the local time at setup.
		//
		// A check-in that should occur every 25m (local time 4:13pm) would be snoozed until 4:38pm.
		// A check-in that should occur every 30min between 14:00 and 18:00 but it's 19:30 should occur next at 14:30.
		if check.Schedule.Every.Start != "" {
			start, err := time.Parse("15:04", check.Schedule.Every.Start)
			if err != nil {
				return time.Second, fmt.Errorf("parsing every.start: %w", err)
			}
			start = time.Date(now.Year(), now.Month(), now.Day(), start.Hour(), start.Minute(), 0, 0, now.Location())

			// Parse the End time, if provided
			if check.Schedule.Every.End != "" {
				end, err := time.Parse("15:04", check.Schedule.Every.End)
				if err != nil {
					return time.Second, fmt.Errorf("parsing every.end: %w", err)
				}
				end.In(now.Location())

				// We're past the End timen
				if (now.Hour() > end.Hour()) || (now.Hour() == end.Hour() && now.Minute() > end.Minute()) {
					// Advance one day
					start = start.AddDate(0, 0, 1)
				}
			}

			// If we're after the start time add onto the start to find the next scheduled interval
			if (now.Hour() > start.Hour()) || (now.Hour() == start.Hour() && now.Minute() > start.Minute()) {
				for {
					if now.Before(start) {
						return start.Sub(now), nil
					}
					start = start.Add(check.Schedule.Every.Interval)
				}
			}
			// If we're before the start time find the distance until we start
			if (now.Hour() < start.Hour()) || (now.Hour() == start.Hour() && now.Minute() <= start.Minute()) {
				return start.Sub(now), nil
			}
		}
		return check.Schedule.Every.Interval, nil

	case check.Schedule.Weekdays != nil, check.Schedule.BankingDays != nil:
		// Scheduled check-ins are snoozed until their next possible occurance.
		var times []time.Time
		if check.Schedule.Weekdays != nil {
			ts, err := check.Schedule.Weekdays.GetTimes() // TODO(adam): check for zero, check .BankingDays as well
			if err != nil {
				return time.Second, fmt.Errorf("calculating snooze for weekday: %w", err)
			}
			times = ts
		}
		if check.Schedule.BankingDays != nil {
			ts, err := check.Schedule.BankingDays.GetTimes()
			if err != nil {
				return time.Second, fmt.Errorf("calculating snooze for banking day: %w", err)
			}
			times = ts
		}
		if len(times) == 0 {
			return time.Second, errors.New("no Times provided")
		}

		var tolerance time.Duration
		if check.Schedule.Weekdays != nil {
			t, err := time.ParseDuration(check.Schedule.Weekdays.Tolerance)
			if err != nil {
				return time.Second, fmt.Errorf("parsing %s as tolerance for weekday snooze: %w", check.Schedule.Weekdays.Tolerance, err)
			}
			tolerance = t
		}
		if check.Schedule.BankingDays != nil {
			t, err := time.ParseDuration(check.Schedule.BankingDays.Tolerance)
			if err != nil {
				return time.Second, fmt.Errorf("parsing %s as tolerance for bankign day snooze: %w", check.Schedule.BankingDays.Tolerance, err)
			}
			tolerance = t
		}

		// Find the next future hour:minute
		current := now.Format("15:04")
		for _, hourminute := range times {
			if hourminute.Format("15:04") > current {
				hhmm := time.Date(now.Year(), now.Month(), now.Day(), hourminute.Hour(), hourminute.Minute(), 0, 0, now.Location())
				return hhmm.Sub(now) + tolerance, nil
			}
		}

		// Find the earliest time tomorrow
		start := times[0]
		future := time.Date(now.Year(), now.Month(), now.Day(), start.Hour(), start.Minute(), 0, 0, now.Location())

		// Didn't find one so try again tomorrow
		if check.Schedule.BankingDays != nil {
			future = base.NewTime(future).AddBankingDay(1).Time
			return future.Sub(now) + tolerance, nil
		} else {
			future = future.AddDate(0, 0, 1)

			if future.Weekday() == time.Saturday {
				future = future.AddDate(0, 0, 1)
			}
			if future.Weekday() == time.Sunday {
				future = future.AddDate(0, 0, 1)
			}

			return future.Sub(now) + tolerance, nil
		}
	}

	return time.Second, nil
}
