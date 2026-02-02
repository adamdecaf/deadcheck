package provider

import (
	"context"
	"errors"
	"time"

	"github.com/adamdecaf/deadcheck/internal/config"
	"github.com/adamdecaf/deadcheck/internal/provider/healthchecksio"
	"github.com/adamdecaf/deadcheck/internal/provider/pd"
	"github.com/adamdecaf/deadcheck/internal/provider/slack"

	"github.com/moov-io/base/log"
	"github.com/moov-io/base/stime"
)

type Client interface {
	Setup(ctx context.Context, check config.Check) error
	CheckIn(ctx context.Context, check config.Check) (time.Time, error)
}

func NewClient(logger log.Logger, conf config.Alert) (Client, error) {
	timeService := stime.NewSystemTimeService()

	switch {
	case conf.HealthChecksIO != nil:
		return healthchecksio.NewClient(logger, conf.HealthChecksIO, timeService)

	case conf.PagerDuty != nil:
		return pd.NewClient(logger, conf.PagerDuty, timeService)

	case conf.Slack != nil:
		return slack.NewClient(logger, conf.Slack, timeService)

	case conf.Mock != nil:
		return NewMockClient(logger), nil

	}
	return nil, errors.New("no provider configured")
}
