package pd

import (
	"context"
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

func TestClient_rejectedCheckIn(t *testing.T) {
	now := time.Now()
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
				Times: []string{
					// Never allow the currnet time to check-in
					now.Add(1*time.Hour + 30*time.Minute).Format("15:04"),
				},
				Tolerance: "1m",
			},
		},
	})
	require.ErrorContains(t, err, "check-in not allowed for")
	require.True(t, nextCheckInExpected.IsZero())
}
