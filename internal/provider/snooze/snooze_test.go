package snooze

import (
	"testing"
	"time"

	"github.com/adamdecaf/deadcheck/internal/config"

	"github.com/stretchr/testify/require"
)

func TestSnooze_Every(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")

	now := time.Date(2024, time.October, 7, 13, 22, 5, 0, loc)

	var schedule config.ScheduleConfig

	t.Run("30min from now", func(t *testing.T) {
		schedule = config.ScheduleConfig{
			Every: &config.EveryConfig{
				Interval: 30 * time.Minute,
			},
		}

		clockTime, snooze, err := Calculate(now, schedule)
		require.NoError(t, err)

		require.Equal(t, "2024-10-07T13:22:05-04:00", clockTime.Format(time.RFC3339))
		require.Equal(t, "30m0s", snooze.String())
		require.Equal(t, "2024-10-07T13:52:05-04:00", now.Add(snooze).Format(time.RFC3339))
	})

	t.Run("30min from 13:00 to 16:00", func(t *testing.T) {
		schedule.Every.Start = "13:00"
		schedule.Every.End = "16:00"

		clockTime, snooze, err := Calculate(now, schedule)
		require.NoError(t, err)

		require.Equal(t, "2024-10-07T13:30:00-04:00", clockTime.Format(time.RFC3339))
		require.Equal(t, "7m55s", snooze.String())
		require.Equal(t, "2024-10-07T13:30:00-04:00", now.Add(snooze).Format(time.RFC3339))
	})

	t.Run("30min from 14:00 to 16:00", func(t *testing.T) {
		schedule.Every.Start = "14:00"
		schedule.Every.End = "16:00"

		clockTime, snooze, err := Calculate(now, schedule)
		require.NoError(t, err)

		require.Equal(t, "2024-10-07T14:00:00-04:00", clockTime.Format(time.RFC3339))
		require.Equal(t, "37m55s", snooze.String())
		require.Equal(t, "2024-10-07T14:00:00-04:00", now.Add(snooze).Format(time.RFC3339))
	})

	t.Run("30min from 12:00 to 13:00", func(t *testing.T) {
		schedule.Every.Start = "12:00"
		schedule.Every.End = "13:00"

		clockTime, snooze, err := Calculate(now, schedule)
		require.NoError(t, err)

		require.Equal(t, "2024-10-08T12:00:00-04:00", clockTime.Format(time.RFC3339))
		require.Equal(t, "22h37m55s", snooze.String())
		require.Equal(t, "2024-10-08T12:00:00-04:00", now.Add(snooze).Format(time.RFC3339))
	})
}

func TestSnooze_Weekdays(t *testing.T) {
	nyc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	now := time.Date(2024, time.October, 7, 13, 22, 5, 0, time.UTC)

	var schedule config.ScheduleConfig

	t.Run("14:00 today", func(t *testing.T) {
		schedule.Weekdays = &config.PartialDay{
			Timezone:  "America/New_York",
			Times:     []string{"14:00", "15:00", "17:00"},
			Tolerance: "5m",
		}

		clockTime, snooze, err := Calculate(now, schedule)
		require.NoError(t, err)

		require.Equal(t, "2024-10-07T14:00:00-04:00", clockTime.Format(time.RFC3339))
		require.Equal(t, "4h42m55s", snooze.String())
		require.Equal(t, "2024-10-07T14:05:00-04:00", now.In(nyc).Add(snooze).Format(time.RFC3339))
	})

	t.Run("15:00 today", func(t *testing.T) {
		schedule.Weekdays = &config.PartialDay{
			Timezone:  "America/New_York",
			Times:     []string{"12:00", "15:00", "17:00"},
			Tolerance: "5m",
		}

		clockTime, snooze, err := Calculate(now, schedule)
		require.NoError(t, err)

		require.Equal(t, "2024-10-07T12:00:00-04:00", clockTime.Format(time.RFC3339))
		require.Equal(t, "2h42m55s", snooze.String())
		require.Equal(t, "2024-10-07T12:05:00-04:00", now.In(nyc).Add(snooze).Format(time.RFC3339))
	})

	t.Run("09:00 tomorrow", func(t *testing.T) {
		now = now.Add(5 * time.Minute)

		schedule.Weekdays.Times = []string{"09:00", "09:10", "09:20"}

		clockTime, snooze, err := Calculate(now, schedule)
		require.NoError(t, err)

		require.Equal(t, "2024-10-07T09:00:00-04:00", clockTime.Format(time.RFC3339))
		require.Equal(t, "23h37m55s", snooze.String())
		require.Equal(t, "2024-10-08T09:05:00-04:00", now.In(nyc).Add(snooze).Format(time.RFC3339))
	})
}

func TestSnooze_BankingDays(t *testing.T) {
	nyc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	t.Run("13:00 tomorrow over weekend + holiday", func(t *testing.T) {
		now := time.Date(2024, time.October, 11, 13, 26, 5, 0, time.UTC)

		var schedule config.ScheduleConfig
		schedule.BankingDays = &config.PartialDay{
			Timezone:  "America/New_York",
			Times:     []string{"09:00", "09:10", "09:20"},
			Tolerance: "5m",
		}

		clockTime, snooze, err := Calculate(now, schedule)
		require.NoError(t, err)

		require.Equal(t, "2024-10-11T09:00:00-04:00", clockTime.Format(time.RFC3339))
		require.Equal(t, "95h38m55s", snooze.String())
		require.Equal(t, "2024-10-15T09:05:00-04:00", now.In(nyc).Add(snooze).Format(time.RFC3339))
	})
}

func TestSnooze_Close(t *testing.T) {
	nyc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	t.Run("1s before 14:00 expected check-in", func(t *testing.T) {
		now := time.Date(2024, time.October, 17, 13, 59, 59, 0, nyc)

		var schedule config.ScheduleConfig
		schedule.BankingDays = &config.PartialDay{
			Timezone:  "America/New_York",
			Times:     []string{"13:30", "14:00", "14:30"},
			Tolerance: "5m",
		}

		clockTime, snooze, err := Calculate(now, schedule)
		require.NoError(t, err)

		require.Equal(t, "2024-10-17T14:00:00-04:00", clockTime.Format(time.RFC3339))
		require.Equal(t, "35m1s", snooze.String())
		require.Equal(t, "2024-10-17T14:35:00-04:00", now.In(nyc).Add(snooze).Format(time.RFC3339))
	})

	t.Run("1s after 14:00 expected check-in", func(t *testing.T) {
		now := time.Date(2024, time.October, 17, 14, 0, 1, 0, nyc)

		var schedule config.ScheduleConfig
		schedule.BankingDays = &config.PartialDay{
			Timezone:  "America/New_York",
			Times:     []string{"13:30", "14:00", "14:30"},
			Tolerance: "5m",
		}

		clockTime, snooze, err := Calculate(now, schedule)
		require.NoError(t, err)

		require.Equal(t, "2024-10-17T14:00:00-04:00", clockTime.Format(time.RFC3339))
		require.Equal(t, "34m59s", snooze.String())
		require.Equal(t, "2024-10-17T14:35:00-04:00", now.In(nyc).Add(snooze).Format(time.RFC3339))
	})

	t.Run("1m before 14:00 expected check-in", func(t *testing.T) {
		now := time.Date(2024, time.October, 17, 13, 59, 0, 0, nyc)

		var schedule config.ScheduleConfig
		schedule.BankingDays = &config.PartialDay{
			Timezone:  "America/New_York",
			Times:     []string{"13:30", "14:00", "14:30"},
			Tolerance: "5m",
		}

		clockTime, snooze, err := Calculate(now, schedule)
		require.NoError(t, err)

		require.Equal(t, "2024-10-17T14:00:00-04:00", clockTime.Format(time.RFC3339))
		require.Equal(t, "36m0s", snooze.String())
		require.Equal(t, "2024-10-17T14:35:00-04:00", now.In(nyc).Add(snooze).Format(time.RFC3339))
	})

	t.Run("1m after 14:00 expected check-in", func(t *testing.T) {
		now := time.Date(2024, time.October, 17, 14, 1, 0, 0, nyc)

		var schedule config.ScheduleConfig
		schedule.BankingDays = &config.PartialDay{
			Timezone:  "America/New_York",
			Times:     []string{"13:30", "14:00", "14:30"},
			Tolerance: "5m",
		}

		clockTime, snooze, err := Calculate(now, schedule)
		require.NoError(t, err)

		require.Equal(t, "2024-10-17T14:00:00-04:00", clockTime.Format(time.RFC3339))
		require.Equal(t, "34m0s", snooze.String())
		require.Equal(t, "2024-10-17T14:35:00-04:00", now.In(nyc).Add(snooze).Format(time.RFC3339))
	})
}
