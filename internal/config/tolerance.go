package config

import (
	"fmt"
	"time"
)

func GetTolerance(schedule ScheduleConfig) time.Duration {
	var input string
	if schedule.Weekdays != nil {
		input = schedule.Weekdays.Tolerance
	}
	if schedule.BankingDays != nil {
		input = schedule.BankingDays.Tolerance
	}

	dur, _ := time.ParseDuration(input)
	return dur
}

func WithinTolerance(now, scheduleTime time.Time, schedule ScheduleConfig) error {
	tolerance := GetTolerance(schedule)

	if tolerance > time.Duration(0) {
		// Allow checkins before the scheduled check-in time according to the tolerance
		switch {
		case now.Before(scheduleTime):
			// We are early to check-in
			diff := scheduleTime.Sub(now)
			if diff > tolerance {
				return fmt.Errorf("%v check-in not allowed for %v", scheduleTime.Format("15:04"), diff)
			}
		case now.Equal(scheduleTime):
			// do nothing, we're on time
			return nil

		case scheduleTime.Before(now):
			// We are late to check-in
			diff := now.Sub(scheduleTime)
			if diff > tolerance {
				return fmt.Errorf("%v check-in is late by %v", scheduleTime.Format("15:04"), diff-tolerance)
			}
		}
	}

	return nil
}
