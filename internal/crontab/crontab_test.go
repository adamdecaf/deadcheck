package crontab

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFormatTime(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	when := time.Date(2024, time.October, 11, 13, 15, 38, 0, loc)
	require.Equal(t, "15 13 11 10 5", FormatTime(when))
}

func TestFormatDuration(t *testing.T) {
	includeSeconds := true

	got := FormatDuration(30*time.Minute, includeSeconds)
	require.Equal(t, "0 1/30 * * * *", got)

	got = FormatDuration(127*time.Minute, includeSeconds)
	require.Equal(t, "0 7 1/2 * * *", got)

	got = FormatDuration(time.Hour, includeSeconds)
	require.Equal(t, "0 0 1/1 * * *", got)

	got = FormatDuration(12*time.Hour, includeSeconds)
	require.Equal(t, "0 0 1/12 * * *", got)

	// Over 24h isn't really supported by crontab
	got = FormatDuration(30*time.Hour, includeSeconds)
	require.Equal(t, "0 0 6 1/2 * *", got) // this is really every 48hrs

	// complex cases
	got = FormatDuration(3*time.Hour+30*time.Minute, includeSeconds)
	require.Equal(t, "0 30 1/3 * * *", got)
}
