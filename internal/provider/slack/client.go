package slack

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"sort"
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

	messages, err := c.findScheduledMessages(ctx, logger, check)
	if err != nil {
		return fmt.Errorf("finding scheduled message: %w", err)
	}
	if len(messages) == 0 {
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
		logger.Info().Logf("found scheduled message %s (and %d more)", messages[0].ID, len(messages)-1)
	}

	return nil
}

func (c *client) findScheduledMessages(ctx context.Context, logger log.Logger, check config.Check) ([]slack.ScheduledMessage, error) {
	params := &slack.GetScheduledMessagesParameters{
		Channel: c.conf.ChannelID,
		Limit:   20,
	}
	messages, _, err := c.underlying.GetScheduledMessagesContext(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("getting scheduled messages from %v failed: %w", params.Channel, err)
	}

	var out []slack.ScheduledMessage
	for _, msg := range messages {
		postAt := time.Unix(int64(msg.PostAt), 0)

		logger := logger.Debug().With(log.Fields{
			"post_at":              log.String(postAt.Format(time.RFC3339)),
			"scheduled_message_id": log.String(msg.ID),
			"text":                 log.String(msg.Text),
		})

		if strings.Contains(msg.Text, check.ID) && strings.Contains(msg.Text, "check-in") {
			logger.Log("found matching scheduled message")
			out = append(out, msg)
		} else {
			logger.Log("found unrelated scheduled message")
		}
	}
	return out, nil
}

func (c *client) createSnoozedMessage(ctx context.Context, logger log.Logger, check config.Check, now time.Time, wait time.Duration) (time.Time, error) {
	expectedCheckin := now.Add(wait)

	text := fmt.Sprintf("%s did not check-in at its scheduled time (%s)",
		check.ID,
		expectedCheckin.Format("3:04PM MST Mon Jan 2"))

	if check.Description != "" {
		text += fmt.Sprintf("\nDescription: %s", check.Description)
	}

	opts := []slack.MsgOption{
		slack.MsgOptionUsername(cmp.Or(c.conf.Username, "deadcheck")),
		slack.MsgOptionText(text, false),
	}
	if c.conf.ImageURI != "" {
		opts = append(opts, slack.MsgOptionIconURL(c.conf.ImageURI))
	}

	postAt := fmt.Sprintf("%d", expectedCheckin.Unix())
	respChannel, scheduledMessageID, err := c.underlying.ScheduleMessageContext(ctx, c.conf.ChannelID, postAt, opts...)
	if err != nil {
		return time.Time{}, fmt.Errorf("scheduling message: %w", err)
	}

	logger.With(log.Fields{
		"post_at":              log.String(postAt),
		"response_channel":     log.String(respChannel),
		"scheduled_message_id": log.String(scheduledMessageID),
	}).Logf("scheduled message for %v", wait)

	return expectedCheckin, nil
}

func (c *client) CheckIn(ctx context.Context, check config.Check) (time.Time, error) {
	logger := c.logger.With(log.Fields{
		"channel_id": log.String(c.conf.ChannelID),
		"check":      log.String(check.ID),
	})

	// Calculate schedule timing
	now := c.timeService.Now()
	scheduleTime, _, err := snooze.Calculate(now, check.Schedule)
	if err != nil {
		return time.Time{}, fmt.Errorf("calculating snooze: %w", err)
	}

	// Validate check-in timing
	err = config.WithinTolerance(now, scheduleTime, check.Schedule)
	if err != nil {
		return time.Time{}, logger.Error().LogError(err).Err()
	}

	// Find existing messages
	messages, err := c.findScheduledMessages(ctx, logger, check)
	if err != nil {
		return time.Time{}, fmt.Errorf("finding scheduled message: %w", err)
	}

	// Calculate next check-in window
	_, wait, err := snooze.Calculate(scheduleTime, check.Schedule)
	if err != nil {
		return time.Time{}, fmt.Errorf("calculating second snooze: %w", err)
	}
	nextCheckIn := scheduleTime.Add(wait)

	// If we found existing messages, handle rescheduling
	if len(messages) > 0 {
		// Sort by post time to get the latest
		sort.Slice(messages, func(i, j int) bool {
			return messages[i].PostAt > messages[j].PostAt
		})

		currentMsg := messages[0]
		currentPostAt := time.Unix(int64(currentMsg.PostAt), 0)

		// Only update if we're extending the time further out
		if nextCheckIn.After(currentPostAt) {
			// Delete existing messages (cleanup any duplicates too)
			for _, msg := range messages {
				err = c.deleteScheduledMessage(ctx, msg)
				if err != nil && !strings.Contains(err.Error(), "invalid_scheduled_message_id") {
					return time.Time{}, fmt.Errorf("deleting message: %w", err)
				}
			}

			// Create new message with extended time
			_, err = c.createSnoozedMessage(ctx, logger, check, now, nextCheckIn.Sub(now))
			if err != nil {
				return time.Time{}, fmt.Errorf("creating new message: %w", err)
			}
		} else {
			logger.Info().Logf("keeping existing message scheduled for %v as it's further out", currentPostAt)
			nextCheckIn = currentPostAt
		}
	} else {
		// No existing message, create new one
		_, err = c.createSnoozedMessage(ctx, logger, check, now, nextCheckIn.Sub(now))
		if err != nil {
			return time.Time{}, fmt.Errorf("creating initial message: %w", err)
		}
	}

	return nextCheckIn, nil
}

func (c *client) deleteScheduledMessage(ctx context.Context, msg slack.ScheduledMessage) error {
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
