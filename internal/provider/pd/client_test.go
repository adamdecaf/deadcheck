package pd

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/adamdecaf/deadcheck/internal/config"
	"github.com/moov-io/base/log"
	"github.com/moov-io/base/stime"

	"github.com/stretchr/testify/require"
)

var (
	adamUserID = "P1F29KL"

	defaultEscalationPolicy = "POHSZE0"
)

func newTestClient(t *testing.T) *client {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping because -short is set")
	}

	conf := config.ReadPagerDutyFromEnv()
	if conf == nil {
		t.Skip("Pagerduty config not provided, skipping...")
	}

	logger := log.NewTestLogger()
	timeService := stime.NewSystemTimeService()
	cc, err := NewClient(logger, conf, timeService)
	require.NoError(t, err)

	cl, ok := cc.(*client)
	require.True(t, ok)

	return cl
}

func skipInCI(t *testing.T) {
	t.Helper()

	inGithubActions := os.Getenv("GITHUB_ACTIONS") != ""
	if inGithubActions {
		t.Skip("not running test in GITHUB_ACTIONS")
	}
}

func TestClient(t *testing.T) {
	pdc := newTestClient(t)
	require.NoError(t, pdc.ping())
}

func TestClient_RejectedEarlyCheckIn(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	now := time.Now().In(loc)
	ctx := context.Background()

	pdc := newTestClient(t)
	t.Cleanup(func() {
		service, err := pdc.findService(ctx, t.Name())
		require.NoError(t, err)

		err = pdc.deleteService(ctx, service)
		require.NoError(t, err)
	})

	nextCheckInExpected, err := pdc.CheckIn(ctx, config.Check{
		Name: t.Name(),
		Schedule: config.ScheduleConfig{
			Weekdays: &config.PartialDay{
				Timezone: "America/New_York",
				Times: []string{
					// Never allow the current time to check-in
					now.Add(1*time.Hour + 30*time.Minute).Format("15:04"),
				},
				Tolerance: "1m",
			},
		},
	})
	require.ErrorContains(t, err, "check-in not allowed for 1h29m")
	require.True(t, nextCheckInExpected.IsZero())
}

func TestClient_RejectedLateCheckIn(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	now := time.Now().In(loc)
	ctx := context.Background()

	pdc := newTestClient(t)
	t.Cleanup(func() {
		service, err := pdc.findService(ctx, t.Name())
		require.NoError(t, err)

		err = pdc.deleteService(ctx, service)
		require.NoError(t, err)
	})

	conf := config.Check{
		Name: t.Name(),
		Schedule: config.ScheduleConfig{
			Weekdays: &config.PartialDay{
				Timezone: "America/New_York",
				Times: []string{
					// Never allow the current time to check-in
					now.Add(-1*time.Hour - 30*time.Minute).Format("15:04"),
					// Add a future time that's too far in the future
					now.Add(2*time.Hour + 30*time.Minute).Format("15:04"),
				},
				Tolerance: "5m",
			},
		},
	}
	nextCheckInExpected, err := pdc.CheckIn(ctx, conf)
	require.ErrorContains(t, err, fmt.Sprintf("%s check-in is late by 1h25m", conf.Schedule.Weekdays.Times[0]))
	require.True(t, nextCheckInExpected.IsZero())
}

func TestClient_CheckInJustBefore(t *testing.T) {
	ctx := context.Background()

	pdc := newTestClient(t)

	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	timeService := stime.NewStaticTimeService()
	timeService.Change(time.Date(2024, time.October, 16, 13, 59, 30, 0, loc)) // just before expected check-in
	pdc.timeService = timeService

	t.Cleanup(func() {
		service, err := pdc.findService(ctx, t.Name())
		require.NoError(t, err)

		err = pdc.deleteService(ctx, service)
		require.NoError(t, err)
	})

	nextCheckInExpected, err := pdc.CheckIn(ctx, config.Check{
		Name: t.Name(),
		Schedule: config.ScheduleConfig{
			Weekdays: &config.PartialDay{
				Timezone:  "America/New_York",
				Times:     []string{"13:30", "14:00", "14:30"},
				Tolerance: "5m",
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "14:35", nextCheckInExpected.Format("15:04"))
}

func TestClient_CheckInJustAfter(t *testing.T) {
	ctx := context.Background()

	pdc := newTestClient(t)

	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	timeService := stime.NewStaticTimeService()
	timeService.Change(time.Date(2024, time.October, 16, 14, 1, 1, 0, loc)) // just after expected check-in
	pdc.timeService = timeService

	t.Cleanup(func() {
		service, err := pdc.findService(ctx, t.Name())
		require.NoError(t, err)

		err = pdc.deleteService(ctx, service)
		require.NoError(t, err)
	})

	nextCheckInExpected, err := pdc.CheckIn(ctx, config.Check{
		Name: t.Name(),
		Schedule: config.ScheduleConfig{
			Weekdays: &config.PartialDay{
				Timezone:  "America/New_York",
				Times:     []string{"13:30", "14:00", "14:30"},
				Tolerance: "5m",
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "14:35", nextCheckInExpected.Format("15:04"))
}
