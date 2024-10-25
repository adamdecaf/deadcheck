package crontab

import (
	"fmt"
	"time"
)

func FormatTime(when time.Time) string {
	// crontab formatting looks like the following:
	// minute   hour   day (month)   month   day (week)
	return fmt.Sprintf("%d %d %d %d %d", when.Minute(), when.Hour(), when.Day(), when.Month(), when.Weekday())
}

// FormatDuration converts a time.Duration into a crontab schedule.
// Not all durations are possible to represnet as a crontab schedule.
//
// Set seconds true to include seconds as part of the crontab schedule.
// Not all crontab parsers support seconds.
func FormatDuration(dur time.Duration, seconds bool) string {
	s, m, h, d := "0", "0", "*", "*"

	days := int(dur.Truncate(24*time.Hour).Hours()) / 24
	if days >= 1 {
		days += 1 // account for Truncate

		d = fmt.Sprintf("1/%d", days)
	}

	hours := int(dur.Truncate(time.Hour).Hours()) % 24
	if hours >= 1 {
		if days >= 1 {
			h = fmt.Sprintf("%d", hours)
		} else {
			h = fmt.Sprintf("1/%d", hours)
		}
	}

	mins := int(dur.Truncate(time.Minute).Minutes()) % 60
	if mins >= 1 {
		if hours > 1 {
			m = fmt.Sprintf("%d", mins)
		} else {
			m = fmt.Sprintf("1/%d", mins)
		}
	}

	secs := int(dur.Seconds()) % 60
	if secs >= 1 {
		s = fmt.Sprintf("1/%d", secs)
	}

	if seconds {
		return fmt.Sprintf("%s %s %s %s * *", s, m, h, d)
	}
	return fmt.Sprintf("%s %s %s * *", m, h, d)
}
