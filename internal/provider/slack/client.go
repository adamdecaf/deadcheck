package slack

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
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
		lastMod:     make(map[string]latestModification),
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
	underlying  *slack.Client
	mu          sync.Mutex

	lastMod   map[string]latestModification
	lastModMu sync.RWMutex
}

type latestModification struct {
	modifiedAt  time.Time
	nextCheckIn time.Time
}

var _ Client = (&client{})

func (c *client) Setup(ctx context.Context, check config.Check) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	err := c.setupScheduledMessage(ctx, check)
	if err != nil {
		return fmt.Errorf("setup scheduled message: %w", err)
	}

	return nil
}

const (
	quiescencePeriod = 5 * time.Minute
)

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
		Limit:   100,
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

		if strings.Contains(msg.Text, fmt.Sprintf("%s did not check-in", check.ID)) {
			logger.Log("found matching scheduled message")
			out = append(out, msg)
		} else {
			logger.Log("found unrelated scheduled message")
		}
	}

	if len(out) > 1 {
		logger.Error().Logf("found %d duplicate messages for check %s", len(out), check.ID)
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
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lastModMu.RLock()
	lastMod, exists := c.lastMod[check.ID]
	c.lastModMu.RUnlock()

	now := c.timeService.Now()
	if exists && now.Sub(lastMod.modifiedAt) < quiescencePeriod {
		// Skip if we're within the quiescence period - another instance just handled a check-in for this check
		return lastMod.nextCheckIn, nil
	}

	logger := c.logger.With(log.Fields{
		"channel_id": log.String(c.conf.ChannelID),
		"check":      log.String(check.ID),
	})

	// Delete existing messages
	messages, err := c.findScheduledMessages(ctx, logger, check)
	if err != nil {
		return time.Time{}, fmt.Errorf("finding scheduled messages: %w", err)
	}

	for _, msg := range messages {
		logger.Info().With(log.Fields{
			"message_id": log.String(msg.ID),
			"post_at":    log.String(time.Unix(int64(msg.PostAt), 0).Format(time.RFC3339)),
		}).Log("deleting scheduled message")

		err = c.deleteScheduledMessage(ctx, msg)
		if err != nil {
			if !strings.Contains(err.Error(), "invalid_scheduled_message_id") {
				logger.Error().LogErrorf("failed to delete scheduled message: %v", err)
			}
		}
	}

	// Calculate next check-in time
	scheduleTime, _, err := snooze.Calculate(now, check.Schedule)
	if err != nil {
		return time.Time{}, fmt.Errorf("calculating snooze: %w", err)
	}

	// Validate check-in timing
	err = config.WithinTolerance(now, scheduleTime, check.Schedule)
	if err != nil {
		return time.Time{}, logger.Error().LogError(err).Err()
	}

	// Calculate next window
	_, wait, err := snooze.Calculate(scheduleTime, check.Schedule)
	if err != nil {
		return time.Time{}, fmt.Errorf("calculating next window: %w", err)
	}

	// Create new message
	nextCheckin, err := c.createSnoozedMessage(ctx, logger, check, now, wait)
	if err != nil {
		return time.Time{}, fmt.Errorf("creating new message: %w", err)
	}

	// Record the modification time
	c.lastModMu.Lock()
	c.lastMod[check.ID] = latestModification{
		modifiedAt:  now,
		nextCheckIn: nextCheckin,
	}
	c.lastModMu.Unlock()

	return nextCheckin, nil
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
