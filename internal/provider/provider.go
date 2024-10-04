package provider

import (
	"context"
	"errors"

	"github.com/adamdecaf/deadcheck/internal/config"
	"github.com/adamdecaf/deadcheck/internal/provider/pd"

	"github.com/moov-io/base/log"
)

type Client interface {
	Setup(ctx context.Context, check config.Check) error
}

func NewClient(logger log.Logger, conf config.Alert) (Client, error) {
	switch {
	case conf.PagerDuty != nil:
		return pd.NewClient(logger, conf.PagerDuty)
	}
	return nil, errors.New("no provider configured")
}
