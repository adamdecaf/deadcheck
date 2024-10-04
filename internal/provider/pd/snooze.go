package pd

import (
	"fmt"
	"time"

	"github.com/adamdecaf/deadcheck/internal/config"

	"github.com/moov-io/base"
	"github.com/moov-io/base/stime"
)

func calculateSnooze(timeService stime.TimeService, check config.Check) (time.Duration, error) {
	now := timeService.Now()
	fmt.Printf("calculateSnooze: %v\n", now.Format(time.RFC3339))

	switch {
	case check.Schedule.Every != nil:
		// Relative check-ins are snoozed for their interval + tolerance from the local
		// time at setup.
		//
		// A check-in that should occur every 25m (local time 4:13pm) would be snoozed
		// until 4:38pm.
		return *check.Schedule.Every, nil

	case check.Schedule.Weekdays != nil, check.Schedule.BankingDays != nil:
		// Scheduled check-ins are snoozed until their next possible occurance.
		times, err := check.Schedule.Weekdays.GetTimes()
		if err != nil {
			return time.Second, fmt.Errorf("calculating snooze for weekday: %w", err)
		}

		tolerance, err := time.ParseDuration(check.Schedule.Weekdays.Tolerance)
		if err != nil {
			return time.Second, fmt.Errorf("parsing %s as tolerance for weekday snooze: %w", check.Schedule.Weekdays.Tolerance, err)
		}

		// Find the next future hour:minute
	findFutureTime:
		current := now.Format("15:04")
		for _, hourminute := range times {
			if hourminute.Format("15:04") > current {
				return hourminute.Sub(now) + tolerance, nil
			}
		}
		// Didn't find one so try again tomorrow
		if check.Schedule.BankingDays != nil {
			now = base.NewTime(now).AddBankingDay(1).Time
		} else {
			now = now.AddDate(0, 0, 1)
		}
		now = now.Truncate(24 * time.Hour) // start of tomorrow
		goto findFutureTime
	}

	return time.Second, nil
}
