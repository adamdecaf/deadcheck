package snooze

import (
	"errors"
	"fmt"
	"time"

	"github.com/adamdecaf/deadcheck/internal/config"

	"github.com/moov-io/base"
)

// Calculate computes the interval of time to delay notifications for based on the current time and scheduled check-in time.
//
// Returned is the wall clock time (as a time.Time) of the closest scheduled time the snooze was calculated from and
// the time.Duration of time to delay notifications for.
func Calculate(now time.Time, schedule config.ScheduleConfig) (time.Time, time.Duration, error) {
	switch {
	case schedule.Every != nil:
		// Relative check-ins are snoozed for their interval + tolerance from the local time at setup.
		//
		// A check-in that should occur every 25m (local time 4:13pm) would be snoozed until 4:38pm.
		// A check-in that should occur every 30min between 14:00 and 18:00 but it's 19:30 should occur next at 14:30.
		if schedule.Every.Start != "" {
			start, err := time.Parse("15:04", schedule.Every.Start)
			if err != nil {
				return time.Time{}, time.Second, fmt.Errorf("parsing every.start: %w", err)
			}
			start = time.Date(now.Year(), now.Month(), now.Day(), start.Hour(), start.Minute(), 0, 0, now.Location())

			// Parse the End time, if provided
			if schedule.Every.End != "" {
				end, err := time.Parse("15:04", schedule.Every.End)
				if err != nil {
					return time.Time{}, time.Second, fmt.Errorf("parsing every.end: %w", err)
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
						return start, start.Sub(now), nil
					}
					start = start.Add(schedule.Every.Interval)
				}
			}
			// If we're before the start time find the distance until we start
			if (now.Hour() < start.Hour()) || (now.Hour() == start.Hour() && now.Minute() <= start.Minute()) {
				return start, start.Sub(now), nil
			}
		}
		return now, schedule.Every.Interval, nil

	case schedule.Weekdays != nil, schedule.BankingDays != nil:
		// Scheduled check-ins are snoozed until their next possible occurrence.
		var times []time.Time
		if schedule.Weekdays != nil {
			ts, err := schedule.Weekdays.GetTimes()
			if err != nil {
				return time.Time{}, time.Second, fmt.Errorf("calculating snooze for weekday: %w", err)
			}
			times = ts

			if schedule.Weekdays.Timezone != "" {
				tz, err := time.LoadLocation(schedule.Weekdays.Timezone)
				if err != nil {
					return time.Time{}, time.Second, fmt.Errorf("reading weekday timezone: %w", err)
				}
				now = now.In(tz)
			}
		}
		if schedule.BankingDays != nil {
			ts, err := schedule.BankingDays.GetTimes()
			if err != nil {
				return time.Time{}, time.Second, fmt.Errorf("calculating snooze for banking day: %w", err)
			}
			times = ts

			if schedule.BankingDays.Timezone != "" {
				tz, err := time.LoadLocation(schedule.BankingDays.Timezone)
				if err != nil {
					return time.Time{}, time.Second, fmt.Errorf("reading banking day timezone: %w", err)
				}
				now = now.In(tz)
			}
		}
		if len(times) == 0 {
			return time.Time{}, time.Second, errors.New("no Times provided")
		}

		var tolerance time.Duration
		if schedule.Weekdays != nil && schedule.Weekdays.Tolerance != "" {
			t, err := time.ParseDuration(schedule.Weekdays.Tolerance)
			if err != nil {
				return time.Time{}, time.Second, fmt.Errorf("parsing %s as tolerance for weekday snooze: %w", schedule.Weekdays.Tolerance, err)
			}
			tolerance = t
		}
		if schedule.BankingDays != nil && schedule.BankingDays.Tolerance != "" {
			t, err := time.ParseDuration(schedule.BankingDays.Tolerance)
			if err != nil {
				return time.Time{}, time.Second, fmt.Errorf("parsing %s as tolerance for bankign day snooze: %w", schedule.BankingDays.Tolerance, err)
			}
			tolerance = t
		}

		// Find the hour:minute (within the tolerance) scheduled check-in which contains the current time.
		for idx, hourminute := range times {
			scheduledCheckIn := time.Date(now.Year(), now.Month(), now.Day(), hourminute.Hour(), hourminute.Minute(), 0, 0, now.Location())

			low := scheduledCheckIn.Add(-1 * tolerance)
			high := scheduledCheckIn.Add(tolerance)

			// The current time must be within our scheduled time +/- the tolerance
			if low.Before(now) && now.Before(high) {
				// Based on the scheduledCheckIn calculate how long to sleep for
				var nextCheckIn time.Time
				if len(times) > idx+1 {
					nextCheckIn = times[idx+1] // the next time later today
				} else {
					nextCheckIn = times[0] // tomorrow's first time
					nextCheckIn = nextCheckIn.AddDate(0, 0, 1)
				}

				next := time.Date(now.Year(), now.Month(), now.Day(), nextCheckIn.Hour(), nextCheckIn.Minute(), 0, 0, now.Location())

				if nextCheckIn.Day() > 1 { // zero value of time.Time is Jan 1 1970
					next = next.AddDate(0, 0, nextCheckIn.Day()-1)
				}

				// Check if the time after snoozing will be a banking day
				snooze := next.Sub(now) + tolerance
				if schedule.BankingDays != nil {
					snooze = snoozeUntilNextBankingDay(scheduledCheckIn, snooze)
				}
				return scheduledCheckIn, snooze, nil
			}

			// We couldn't find an interval to check-in, so return the time until the first available check-in.
			// This is useful for the initial snooze on startup
			if now.Before(low) {
				snooze := high.Sub(now)
				if schedule.BankingDays != nil {
					snooze = snoozeUntilNextBankingDay(scheduledCheckIn, snooze)
				}
				return scheduledCheckIn, snooze, nil
			}
		}

		// Find the earliest time tomorrow, since we were late to all of them.
		start := time.Date(now.Year(), now.Month(), now.Day(), times[0].Hour(), times[0].Minute(), 0, 0, now.Location())
		future := time.Date(now.Year(), now.Month(), now.Day(), start.Hour(), start.Minute(), 0, 0, now.Location())

		// Didn't find one so try again tomorrow
		if schedule.BankingDays != nil {
			future = base.NewTime(future).AddBankingDay(1).Time
			return start, future.Sub(now) + tolerance, nil
		} else {
			future = future.AddDate(0, 0, 1)

			if future.Weekday() == time.Saturday {
				future = future.AddDate(0, 0, 1)
			}
			if future.Weekday() == time.Sunday {
				future = future.AddDate(0, 0, 1)
			}

			return start, future.Sub(now) + tolerance, nil
		}
	}

	return time.Time{}, time.Second, nil
}

func snoozeUntilNextBankingDay(scheduledCheckIn time.Time, snooze time.Duration) time.Duration {
	bt := base.NewTime(scheduledCheckIn.Add(snooze))
	if !bt.IsBankingDay() {
		bt = bt.AddBankingDay(1)
		snooze = bt.Time.Sub(scheduledCheckIn)
	}
	return snooze
}
