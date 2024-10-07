package pd

import (
	"testing"
	"time"

	"github.com/adamdecaf/deadcheck/internal/config"

	"github.com/moov-io/base/stime"
	"github.com/stretchr/testify/require"
)

func TestSnooze_Every(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")

	timeService := stime.NewStaticTimeService()
	timeService.Change(time.Date(2024, time.October, 7, 13, 22, 5, 0, loc))

	var check config.Check

	t.Run("30min from now", func(t *testing.T) {
		check.Schedule = config.ScheduleConfig{
			Every: &config.EveryConfig{
				Interval: 30 * time.Minute,
			},
		}

		snooze, err := calculateSnooze(timeService, check)
		require.NoError(t, err)
		require.Equal(t, "30m0s", snooze.String())
		require.Equal(t, "2024-10-07T13:52:05-04:00", timeService.Now().Add(snooze).Format(time.RFC3339))
	})

	t.Run("30min from 13:00 to 16:00", func(t *testing.T) {
		check.Schedule.Every.Start = "13:00"
		check.Schedule.Every.End = "16:00"

		snooze, err := calculateSnooze(timeService, check)
		require.NoError(t, err)
		require.Equal(t, "7m55s", snooze.String())
		require.Equal(t, "2024-10-07T13:30:00-04:00", timeService.Now().Add(snooze).Format(time.RFC3339))
	})

	t.Run("30min from 14:00 to 16:00", func(t *testing.T) {
		check.Schedule.Every.Start = "14:00"
		check.Schedule.Every.End = "16:00"

		snooze, err := calculateSnooze(timeService, check)
		require.NoError(t, err)
		require.Equal(t, "37m55s", snooze.String())
		require.Equal(t, "2024-10-07T14:00:00-04:00", timeService.Now().Add(snooze).Format(time.RFC3339))
	})

	t.Run("30min from 12:00 to 13:00", func(t *testing.T) {
		check.Schedule.Every.Start = "12:00"
		check.Schedule.Every.End = "13:00"

		snooze, err := calculateSnooze(timeService, check)
		require.NoError(t, err)

		require.Equal(t, "22h37m55s", snooze.String())
		require.Equal(t, "2024-10-08T12:00:00-04:00", timeService.Now().Add(snooze).Format(time.RFC3339))
	})
}

func TestSnooze_Weekdays(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")

	timeService := stime.NewStaticTimeService()
	timeService.Change(time.Date(2024, time.October, 7, 13, 22, 5, 0, loc))

	var check config.Check

	t.Run("14:00 today", func(t *testing.T) {
		check.Schedule.Weekdays = &config.PartialDay{
			Timezone:  "America/New_York",
			Times:     []string{"14:00", "15:00", "17:00"},
			Tolerance: "5m",
		}

		snooze, err := calculateSnooze(timeService, check)
		require.NoError(t, err)

		require.Equal(t, "42m55s", snooze.String())
		require.Equal(t, "2024-10-07T14:05:00-04:00", timeService.Now().Add(snooze).Format(time.RFC3339))
	})

	t.Run("15:00 today", func(t *testing.T) {
		check.Schedule.Weekdays = &config.PartialDay{
			Timezone:  "America/New_York",
			Times:     []string{"12:00", "15:00", "17:00"},
			Tolerance: "5m",
		}

		snooze, err := calculateSnooze(timeService, check)
		require.NoError(t, err)

		require.Equal(t, "1h42m55s", snooze.String())
		require.Equal(t, "2024-10-07T15:05:00-04:00", timeService.Now().Add(snooze).Format(time.RFC3339))
	})

	t.Run("13:00 tomorrow", func(t *testing.T) {
		check.Schedule.Weekdays.Times = []string{"13:00", "13:10", "13:20"}

		snooze, err := calculateSnooze(timeService, check)
		require.NoError(t, err)

		require.Equal(t, "23h42m55s", snooze.String())
		require.Equal(t, "2024-10-08T13:05:00-04:00", timeService.Now().Add(snooze).Format(time.RFC3339))
	})
}

func TestSnooze_BankingDays(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")

	timeService := stime.NewStaticTimeService()
	timeService.Change(time.Date(2024, time.October, 7, 13, 22, 5, 0, loc))

	var check config.Check

	t.Run("13:00 tomorrow over weekend + holiday", func(t *testing.T) {
		timeService.Change(time.Date(2024, time.October, 11, 13, 22, 5, 0, loc))

		check.Schedule.BankingDays = &config.PartialDay{
			Timezone:  "America/New_York",
			Times:     []string{"13:00", "13:10", "13:20"},
			Tolerance: "5m",
		}

		snooze, err := calculateSnooze(timeService, check)
		require.NoError(t, err)

		require.Equal(t, "95h42m55s", snooze.String())
		require.Equal(t, "2024-10-15T13:05:00-04:00", timeService.Now().Add(snooze).Format(time.RFC3339))
	})
}
