package healthchecksio

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

func newTestClient(t *testing.T) *client {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping because -short is set")
	}

	conf := config.ReadHealthChecksIOFromEnv()
	if conf == nil {
		t.Skip("HealthChecks.io config not provided, skipping...")
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
	cc := newTestClient(t)

	ctx := context.Background()

	check := config.Check{
		ID:          fmt.Sprintf("%d", time.Now().Unix()),
		Name:        "Example",
		Description: "example check",
		Schedule: config.ScheduleConfig{
			Weekdays: &config.PartialDay{
				Timezone:  "America/New_York",
				Times:     []string{"10:00"},
				Tolerance: "24h",
			},
		},
		Alert: config.Alert{
			Slack: config.ReadSlackFromEnv(),
		},
	}

	created, err := cc.setupCheck(ctx, check)
	require.NoError(t, err)
	require.NotNil(t, created)

	t.Cleanup(func() {
		_, err := cc.underlying.DeleteCheck(ctx, created.UUID)
		require.NoError(t, err)
	})

	require.Equal(t, check.ID, created.Slug)
	require.Equal(t, check.Name, created.Name)
	require.Greater(t, created.Grace, 0)

	nextCheckin, err := cc.CheckIn(ctx, check)
	require.NoError(t, err)
	require.NotEmpty(t, nextCheckin)
	require.Greater(t, nextCheckin.Year(), 2025)
}
