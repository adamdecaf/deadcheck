package healthchecksio

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/adamdecaf/deadcheck/internal/config"
	"github.com/adamdecaf/deadcheck/internal/crontab"
	"github.com/adamdecaf/deadcheck/internal/provider/snooze"
	"github.com/adamdecaf/go-healthchecksio/pkg/healthchecksio"
	"github.com/moov-io/base/log"
	"github.com/moov-io/base/stime"
	"github.com/moov-io/base/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Client interface {
	Setup(ctx context.Context, check config.Check) error
	CheckIn(ctx context.Context, check config.Check) (time.Time, error)
}

func NewClient(logger log.Logger, conf *config.HealthChecksIO, timeService stime.TimeService) (Client, error) {
	if conf == nil {
		return nil, nil
	}
	underlying := healthchecksio.NewClient(conf.ApiKey)
	if underlying == nil {
		return nil, errors.New("no healthchecks.io client created")
	}
	return &client{
		logger:      logger,
		conf:        *conf,
		timeService: timeService,
		underlying:  underlying,
	}, nil
}

type client struct {
	logger      log.Logger
	conf        config.HealthChecksIO
	timeService stime.TimeService
	underlying  healthchecksio.Client
}

func (c *client) Setup(ctx context.Context, check config.Check) error {
	ctx, span := telemetry.StartSpan(ctx, "healthchecksio-setup", trace.WithAttributes(
		attribute.String("check_id", check.ID),
	))
	defer span.End()

	_, err := c.setupCheck(ctx, check)
	return err
}

func (c *client) setupCheck(ctx context.Context, check config.Check) (*healthchecksio.Check, error) {
	ctx, span := telemetry.StartSpan(ctx, "healthchecksio-setup-check", trace.WithAttributes(
		attribute.String("check_id", check.ID),
	))
	defer span.End()

	checksFound, err := c.underlying.GetChecks(ctx, healthchecksio.GetChecks{
		Tags: check.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("listing checks: %w", err)
	}

	var found *healthchecksio.Check
	for i := range checksFound.Checks {
		if checksFound.Checks[i].Name == check.Name {
			found = &checksFound.Checks[i]
			break
		}
	}

	if found != nil {
		return found, nil
	}

	created, err := c.createCheck(ctx, check)
	if err != nil {
		return nil, fmt.Errorf("creating check: %w", err)
	}

	c.logger.Info().Logf("setup check %s", created.Name)

	return created, nil
}

func (c *client) createCheck(ctx context.Context, check config.Check) (*healthchecksio.Check, error) {
	create := &healthchecksio.CreateCheck{
		Name:        check.Name,
		Slug:        check.ID,
		Tags:        check.ID,
		Unique:      []string{"slug"},
		Description: check.Description,
	}

	// Otherwise set the schedule, which overrides Timeout
	loc, err := getTimezone(check)
	if err != nil {
		return nil, fmt.Errorf("getting timezone from check %s: %v", check.ID, err)
	}
	create.Timezone = loc.String()

	now := c.timeService.Now().In(loc)
	nextCheckIn, _, err := snooze.Calculate(now, check.Schedule)
	if err != nil {
		return nil, fmt.Errorf("calculating snooze: %w", err)
	}

	// We expect the next check-in at nextCheckIn, but allow for delay seconds as grace
	create.Schedule = crontab.FormatTime(nextCheckIn)

	tolerance := getTolerance(check.Schedule)
	create.Grace = max(int(tolerance.Seconds()), 60)

	logger := c.logger.Info().With(log.Fields{
		"check":        log.String(check.ID),
		"next_checkin": log.String(nextCheckIn.Format(time.RFC3339)),
	})
	logger.Info().Logf("creating check with schedule %v and grace %v", create.Schedule, create.Grace)

	// Setup the check on HealthChecks.io
	created, err := c.underlying.CreateCheck(ctx, create)
	if err != nil {
		return nil, fmt.Errorf("creating check: %w", err)
	}

	logger.Info().With(log.Fields{
		"uuid": log.String(created.UUID),
	}).Logf("setup check %s on healthchecks.io", check.ID)

	return created, nil
}

func (c *client) CheckIn(ctx context.Context, check config.Check) (time.Time, error) {
	ctx, span := telemetry.StartSpan(ctx, "healthchecksio-checkin", trace.WithAttributes(
		attribute.String("check_id", check.ID),
	))
	defer span.End()

	hcCheck, err := c.setupCheck(ctx, check)
	if err != nil {
		return time.Time{}, fmt.Errorf("setup check: %w", err)
	}

	// Send a success ping
	err = c.underlying.Ping(ctx, hcCheck.PingURL, "")
	if err != nil {
		return time.Time{}, fmt.Errorf("ping: %w", err)
	}

	logger := c.logger.Info().With(log.Fields{
		"check": log.String(check.ID),
	})

	// Easy way to calculate would be to find the remaining snooze and add that to now()
	// then calculate the next snooze.
	now := c.timeService.Now()

	scheduleTime, _, err := snooze.Calculate(now, check.Schedule)
	if err != nil {
		return time.Time{}, fmt.Errorf("calculating snooze: %w", err)
	}

	// Only allow check-ins with the tolerance specified.
	//  e.g. If the tolerance is 5mins for a check-in expected at 4pm then only between 3:55pm and 4:05pm
	//       would check-ins be allowed.
	tolerance := getTolerance(check.Schedule)

	if tolerance > time.Duration(0) {
		// Allow checkins before the scheduled check-in time according to the tolerance
		switch {
		case now.Before(scheduleTime):
			// We are early to check-in
			diff := scheduleTime.Sub(now)
			if diff > tolerance {
				err = fmt.Errorf("%v check-in not allowed for %v", scheduleTime.Format("15:04"), diff)
				return time.Time{}, err
			}
		case now.Equal(scheduleTime):
			// do nothing, we're on time

		case scheduleTime.Before(now):
			// We are late to check-in
			diff := now.Sub(scheduleTime)
			if diff > tolerance {
				err = fmt.Errorf("%v check-in is late by %v", scheduleTime.Format("15:04"), diff-tolerance)
				return time.Time{}, err
			}
		}
	}

	// future := now.Add(wait)
	_, wait, err := snooze.Calculate(scheduleTime, check.Schedule)
	if err != nil {
		return time.Time{}, fmt.Errorf("calculating second snooze: %w", err)
	}

	nextCheckIn := scheduleTime.Add(wait).Add(-1 * tolerance)

	// We expect the next check-in at nextCheckIn, but allow for delay seconds as grace
	update := &healthchecksio.UpdateCheck{
		Schedule: crontab.FormatTime(nextCheckIn),
		Grace:    max(int(tolerance.Seconds()), 60),
	}

	logger.Info().Logf("updating schedule to %v with %v grace", update.Schedule, update.Grace)

	// Update the check with the next expected checkin
	_, err = c.underlying.UpdateCheck(ctx, hcCheck.UUID, update)
	if err != nil {
		return time.Time{}, fmt.Errorf("updating check %s failed: %v", check.ID, err)
	}

	logger = logger.With(log.Fields{
		"next_checkin": log.String(nextCheckIn.Format(time.RFC3339)),
	})
	logger.Logf("%s accepted check-in on healthchecks.io", check.ID)

	return nextCheckIn, nil
}

func getTimezone(check config.Check) (*time.Location, error) {
	var tz string
	if check.Schedule.Weekdays != nil {
		tz = check.Schedule.Weekdays.Timezone
	}
	if check.Schedule.BankingDays != nil {
		tz = check.Schedule.BankingDays.Timezone
	}

	if tz != "" {
		return time.LoadLocation(tz)
	}
	return time.Now().Location(), nil
}

func getTolerance(schedule config.ScheduleConfig) time.Duration {
	var input string
	if schedule.Weekdays != nil {
		input = schedule.Weekdays.Tolerance
	}
	if schedule.BankingDays != nil {
		input = schedule.BankingDays.Tolerance
	}

	dur, _ := time.ParseDuration(input)
	return dur
}
