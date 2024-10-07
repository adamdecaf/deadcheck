package provider

import (
	"context"
	"errors"
	"time"

	"github.com/adamdecaf/deadcheck/internal/config"
	"github.com/adamdecaf/deadcheck/internal/provider/pd"

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
	case conf.PagerDuty != nil:
		return pd.NewClient(logger, conf.PagerDuty, timeService)
	}
	return nil, errors.New("no provider configured")
}
