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
