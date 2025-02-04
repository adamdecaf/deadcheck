package slack

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/adamdecaf/deadcheck/internal/config"
	"github.com/adamdecaf/deadcheck/internal/provider/snooze"

	"github.com/moov-io/base/log"
	"github.com/moov-io/base/stime"
	"github.com/slack-go/slack"
)

type Client interface {
	Setup(ctx context.Context, check config.Check) error
	CheckIn(ctx context.Context, check config.Check) (time.Time, error)
}

func NewClient(logger log.Logger, conf *config.Slack, timeService stime.TimeService) (Client, error) {
	if conf == nil {
		return nil, nil
	}

	cc := &client{
		logger:      logger,
		conf:        *conf,
		timeService: timeService,
	}

	underlying := slack.New(conf.ApiToken)
	if underlying == nil {
		return nil, errors.New("no slack client created")
	}
	cc.underlying = underlying

	return cc, nil
}

type client struct {
	logger      log.Logger
	conf        config.Slack
	timeService stime.TimeService

	underlying *slack.Client
}

var _ Client = (&client{})

func (c *client) Setup(ctx context.Context, check config.Check) error {
	err := c.setupScheduledMessage(ctx, check)
	if err != nil {
		return fmt.Errorf("setup scheduled message: %w", err)
	}

	return nil
}

func (c *client) setupScheduledMessage(ctx context.Context, check config.Check) error {
	logger := c.logger.With(log.Fields{
		"channel_id": log.String(c.conf.ChannelID),
		"check":      log.String(check.ID),
	})

	msg, err := c.findScheduledMessage(ctx, check)
	if err != nil {
		return fmt.Errorf("finding scheduled message: %w", err)
	}
	if msg == nil {
		now := c.timeService.Now()
		_, wait, err := snooze.Calculate(now, check.Schedule)
		if err != nil {
			return fmt.Errorf("calculating snooze: %w", err)
		}

		_, err = c.createSnoozedMessage(ctx, logger, check, now, wait)
		if err != nil {
			return fmt.Errorf("setting up snoozed message: %w", err)
		}
	} else {
		logger.Info().Logf("using scheduled message %s", msg.ID)
	}

	return nil
}

func (c *client) findScheduledMessage(ctx context.Context, check config.Check) (*slack.ScheduledMessage, error) {
	params := &slack.GetScheduledMessagesParameters{
		Channel: c.conf.ChannelID,
		Limit:   20,
	}
	messages, _, err := c.underlying.GetScheduledMessagesContext(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("getting scheduled messages from %v failed: %w", params.Channel, err)
	}
	for _, msg := range messages {
		if strings.Contains(msg.Text, check.ID) && strings.Contains(msg.Text, "check-in") {
			return &msg, nil
		}
	}
	return nil, nil
}

func (c *client) createSnoozedMessage(ctx context.Context, logger log.Logger, check config.Check, now time.Time, wait time.Duration) (time.Time, error) {
	scheduleTime, _, err := snooze.Calculate(now, check.Schedule)
	if err != nil {
		return time.Time{}, fmt.Errorf("calculating snooze: %w", err)
	}

	expectedCheckin := now.Add(wait)
	text := fmt.Sprintf("%s did not check-in, expected check-in at %v", check.ID, expectedCheckin.Format(time.RFC3339))
	opts := []slack.MsgOption{
		slack.MsgOptionUsername(cmp.Or(c.conf.Username, "deadcheck")),
		slack.MsgOptionText(text, false),
	}
	if c.conf.ImageURI != "" {
		opts = append(opts, slack.MsgOptionIconURL(c.conf.ImageURI))
	}

	postAt := fmt.Sprintf("%d", scheduleTime.Add(wait).Unix())
	respChannel, scheduledMessageID, err := c.underlying.ScheduleMessageContext(ctx, c.conf.ChannelID, postAt, opts...)
	if err != nil {
		return time.Time{}, fmt.Errorf("problem with ScheduleMessageContext: %w", err)
	}

	logger.With(log.Fields{
		"post_at":              log.String(postAt),
		"response_channel":     log.String(respChannel),
		"scheduled_message_id": log.String(scheduledMessageID),
	}).Logf("scheduled snoozed message for %v", wait)

	return scheduleTime.Add(wait), nil
}

func (c *client) CheckIn(ctx context.Context, check config.Check) (time.Time, error) {
	logger := c.logger.With(log.Fields{
		"channel_id": log.String(c.conf.ChannelID),
		"check":      log.String(check.ID),
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
	err = config.WithinTolerance(now, scheduleTime, check.Schedule)
	if err != nil {
		return time.Time{}, logger.Error().LogError(err).Err()
	}

	// Check-in is allowed, delete the scheduled message and queue a new one
	msg, err := c.findScheduledMessage(ctx, check)
	if err != nil {
		return time.Time{}, fmt.Errorf("finding scheduled message: %w", err)
	}

	// Delete the existing message (assume it's in a very-near future)
	if msg != nil {
		postAt := time.Unix(int64(msg.PostAt), 0)

		logger.With(log.Fields{
			"post_at":              log.String(postAt.Format(time.RFC3339)),
			"scheduled_message_id": log.String(msg.ID),
		}).Log("deleting scheduled message")

		err = c.deleteScheduledMessage(ctx, msg)
		if err != nil {
			// Skip invalid_scheduled_message_id as another instance may have deleted it
			if !strings.Contains(err.Error(), "invalid_scheduled_message_id") {
				return time.Time{}, logger.Error().LogErrorf("problem deleting scheduled message: %w", err).Err()
			}
		}
	}

	// Find the future check-in time
	_, wait, err := snooze.Calculate(scheduleTime, check.Schedule)
	if err != nil {
		return time.Time{}, fmt.Errorf("calculating second snooze: %w", err)
	}

	future := scheduleTime.Add(wait)
	logger.Info().Logf("snoozing %s scheduled message until %v", check.ID, future.Format(time.RFC3339))

	// Create a new scheduled message
	nextCheckin, err := c.createSnoozedMessage(ctx, logger, check, now, future.Sub(now))
	if err != nil {
		return time.Time{}, fmt.Errorf("problem creating snoozed message: %w", err)
	}

	return nextCheckin, nil
}

func (c *client) deleteScheduledMessage(ctx context.Context, msg *slack.ScheduledMessage) error {
	params := &slack.DeleteScheduledMessageParameters{
		Channel:            c.conf.ChannelID,
		ScheduledMessageID: msg.ID,
		AsUser:             true,
	}

	_, err := c.underlying.DeleteScheduledMessageContext(ctx, params)
	if err != nil {
		return fmt.Errorf("deleting scheduled message: %w", err)
	}

	return nil
}
