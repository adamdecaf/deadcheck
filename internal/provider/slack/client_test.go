package slack

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

	conf := config.ReadSlackFromEnv()
	if conf == nil {
		t.Skip("Slack config not provided, skipping...")
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

	found, err := cc.findScheduledMessage(ctx, check)
	require.NoError(t, err)
	require.Nil(t, found)

	err = cc.Setup(ctx, check)
	require.NoError(t, err)

	found, err = cc.findScheduledMessage(ctx, check)
	require.NoError(t, err)
	require.NotNil(t, found)

	nextCheckin, err := cc.CheckIn(ctx, check)
	require.NoError(t, err)
	require.InDelta(t, nextCheckin.Unix(), int64(found.PostAt), 10) // 10s
}
