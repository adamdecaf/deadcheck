package api_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/adamdecaf/deadcheck/internal/api"
	"github.com/adamdecaf/deadcheck/internal/check"
	"github.com/adamdecaf/deadcheck/internal/config"

	"github.com/moov-io/base/log"
	"github.com/stretchr/testify/require"
)

func TestServer(t *testing.T) {
	logger := log.NewTestLogger()

	conf := config.ServerConfig{
		BindAddress: ":58732",
	}

	instances, err := check.Setup(context.Background(), logger, &config.Config{
		Checks: []config.Check{
			{
				ID:   "foo",
				Name: "foo bar",
				Schedule: config.ScheduleConfig{
					Every: &config.EveryConfig{
						Interval: 10 * time.Minute,
					},
				},
				Alert: config.Alert{
					Mock: &config.MockAlerter{},
				},
			},
		},
	})
	require.NoError(t, err)

	server, err := api.Server(logger, conf, instances)
	require.NoError(t, err)

	t.Cleanup(func() {
		server.Close()
	})

	req, err := http.NewRequest("POST", "http://localhost"+conf.BindAddress+"/checks/foo/check-in", nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, resp.StatusCode)
}
