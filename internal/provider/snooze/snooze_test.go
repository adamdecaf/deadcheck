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

		snooze, err := Calculate(now, schedule)
		require.NoError(t, err)
		require.Equal(t, "30m0s", snooze.String())
		require.Equal(t, "2024-10-07T13:52:05-04:00", now.Add(snooze).Format(time.RFC3339))
	})

	t.Run("30min from 13:00 to 16:00", func(t *testing.T) {
		schedule.Every.Start = "13:00"
		schedule.Every.End = "16:00"

		snooze, err := Calculate(now, schedule)
		require.NoError(t, err)
		require.Equal(t, "7m55s", snooze.String())
		require.Equal(t, "2024-10-07T13:30:00-04:00", now.Add(snooze).Format(time.RFC3339))
	})

	t.Run("30min from 14:00 to 16:00", func(t *testing.T) {
		schedule.Every.Start = "14:00"
		schedule.Every.End = "16:00"

		snooze, err := Calculate(now, schedule)
		require.NoError(t, err)
		require.Equal(t, "37m55s", snooze.String())
		require.Equal(t, "2024-10-07T14:00:00-04:00", now.Add(snooze).Format(time.RFC3339))
	})

	t.Run("30min from 12:00 to 13:00", func(t *testing.T) {
		schedule.Every.Start = "12:00"
		schedule.Every.End = "13:00"

		snooze, err := Calculate(now, schedule)
		require.NoError(t, err)

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

		snooze, err := Calculate(now, schedule)
		require.NoError(t, err)

		require.Equal(t, "4h42m55s", snooze.String())
		require.Equal(t, "2024-10-07T14:05:00-04:00", now.In(nyc).Add(snooze).Format(time.RFC3339))
	})

	t.Run("15:00 today", func(t *testing.T) {
		schedule.Weekdays = &config.PartialDay{
			Timezone:  "America/New_York",
			Times:     []string{"12:00", "15:00", "17:00"},
			Tolerance: "5m",
		}

		snooze, err := Calculate(now, schedule)
		require.NoError(t, err)

		require.Equal(t, "2h42m55s", snooze.String())
		require.Equal(t, "2024-10-07T12:05:00-04:00", now.In(nyc).Add(snooze).Format(time.RFC3339))
	})

	t.Run("09:00 tomorrow", func(t *testing.T) {
		schedule.Weekdays.Times = []string{"09:00", "09:10", "09:20"}

		snooze, err := Calculate(now, schedule)
		require.NoError(t, err)

		require.Equal(t, "23h42m55s", snooze.String())
		require.Equal(t, "2024-10-08T09:05:00-04:00", now.In(nyc).Add(snooze).Format(time.RFC3339))
	})
}

func TestSnooze_BankingDays(t *testing.T) {
	nyc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	t.Run("13:00 tomorrow over weekend + holiday", func(t *testing.T) {
		now := time.Date(2024, time.October, 11, 13, 22, 5, 0, time.UTC)

		var schedule config.ScheduleConfig
		schedule.BankingDays = &config.PartialDay{
			Timezone:  "America/New_York",
			Times:     []string{"09:00", "09:10", "09:20"},
			Tolerance: "5m",
		}

		snooze, err := Calculate(now, schedule)
		require.NoError(t, err)

		require.Equal(t, "95h42m55s", snooze.String())
		require.Equal(t, "2024-10-15T09:05:00-04:00", now.In(nyc).Add(snooze).Format(time.RFC3339))
	})
}
