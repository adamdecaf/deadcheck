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
