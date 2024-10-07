package check

import (
	"cmp"
	"context"
	"fmt"
	"time"

	"github.com/adamdecaf/deadcheck/internal/config"
	"github.com/adamdecaf/deadcheck/internal/provider"

	"github.com/moov-io/base/log"
)

type Instances struct {
	checks []config.Check
	conf   *config.Config
}

func Setup(ctx context.Context, logger log.Logger, conf *config.Config) (*Instances, error) {
	if conf == nil {
		return nil, nil
	}

	for idx, check := range conf.Checks {
		checkLogger := logger.Info().With(log.Fields{
			"check_name": log.String(check.Name),
		})

		client, err := provider.NewClient(checkLogger, cmp.Or(check.Alert, conf.Alert))
		if err != nil {
			return nil, fmt.Errorf("setting up check[%d] provider: %w", idx, err)
		}

		err = client.Setup(ctx, check)
		if err != nil {
			return nil, fmt.Errorf("problem setting up check %v: %w", check.ID, err)
		}

		checkLogger.Logf("setup check %v (%v)", check.Name, check.ID)
	}

	return &Instances{
		checks: conf.Checks,
		conf:   conf,
	}, nil
}

type CheckInResponse struct {
	NextExpectedCheckIn time.Time
}

func (xs *Instances) CheckIn(ctx context.Context, logger log.Logger, checkID string) (*CheckInResponse, error) {
	var found *config.Check
	for i := range xs.checks {
		if xs.checks[i].ID == checkID {
			found = &xs.checks[i]
			break
		}
	}
	if found == nil {
		return nil, fmt.Errorf("check %s not found", checkID)
	}

	logger = logger.With(log.Fields{
		"check_name": log.String(found.Name),
	})

	// Grab the provider client for the check
	client, err := provider.NewClient(logger, cmp.Or(found.Alert, xs.conf.Alert))
	if err != nil {
		return nil, fmt.Errorf("problem getting client for check-in: %w", err)
	}

	checkInExpected, err := client.CheckIn(ctx, *found)
	if err != nil {
		return nil, fmt.Errorf("check-in fialed: %w", err)
	}
	logger.Info().Logf("check-in complete, expected again before %v", checkInExpected.Format(time.RFC3339))

	return &CheckInResponse{
		NextExpectedCheckIn: checkInExpected,
	}, nil
}
