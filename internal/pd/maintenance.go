// Licensed to Adam Shannon under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. The Moov Authors licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package pd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/adamdecaf/deadcheck/internal/config"
)

// TODO
//  Find maint windows
//  Ensure configured windows are present (every, weekdays, banking-days)
//   If not, then setup
//
//  Support extending a maint window

func (c *client) setupMaintenanceWindows(check config.Check) error {
	_, err := c.findMaintenanceWindows(check)
	return err
}

const (
	everyDescription       = "every"
	weekdaysDescription    = "weekdays"
	bankingDaysDescription = "banking-days"
)

func (c *client) findMaintenanceWindows(check config.Check) ([]*pagerduty.MaintenanceWindow, error) {
	ctx := context.Background()
	resp, err := c.underlying.ListMaintenanceWindowsWithContext(ctx, pagerduty.ListMaintenanceWindowsOptions{
		Limit:      100,
		ServiceIDs: []string{c.service.ID},
	})
	if err != nil {
		return nil, fmt.Errorf("listing maint windows: %v", err)
	}

	foundEvery := false
	foundWeekday := false
	foundBankingDays := false

	var found []*pagerduty.MaintenanceWindow
	for i := range resp.MaintenanceWindows {
		switch resp.MaintenanceWindows[i].Description {
		case everyDescription:
			foundEvery = true
			maint, err := c.ensureMaintenanceWindow_Every(check.Schedule.Every, &resp.MaintenanceWindows[i])
			if err != nil {
				return nil, err
			}
			if maint != nil {
				found = append(found, maint)
			}

		case weekdaysDescription:
			foundWeekday = true
			maint, err := c.ensureMaintenanceWindow_PartialDay(check.Schedule.Weekdays, &resp.MaintenanceWindows[i])
			if err != nil {
				return nil, err
			}
			if maint != nil {
				found = append(found, maint)
			}

		case bankingDaysDescription:
			foundBankingDays = true
			maint, err := c.ensureMaintenanceWindow_PartialDay(check.Schedule.BankingDays, &resp.MaintenanceWindows[i])
			if err != nil {
				return nil, err
			}
			if maint != nil {
				found = append(found, maint)
			}
		}
	}

	if !foundEvery {
		maint, err := c.ensureMaintenanceWindow_Every(check.Schedule.Every, nil)
		if err != nil {
			return nil, err
		}
		if maint != nil {
			found = append(found, maint)
		}
	}

	if !foundWeekday {
		maint, err := c.ensureMaintenanceWindow_PartialDay(check.Schedule.Weekdays, nil)
		if err != nil {
			return nil, err
		}
		if maint != nil {
			found = append(found, maint)
		}
	}

	if !foundBankingDays {
		maint, err := c.ensureMaintenanceWindow_PartialDay(check.Schedule.BankingDays, nil)
		if err != nil {
			return nil, err
		}
		if maint != nil {
			found = append(found, maint)
		}
	}

	return found, nil
}

func (c *client) ensureMaintenanceWindow_Every(every *time.Duration, maintWindow *pagerduty.MaintenanceWindow) (*pagerduty.MaintenanceWindow, error) {
	if every == nil {
		return nil, nil
	}
	if maintWindow == nil {
		start := time.Now().In(time.UTC)
		end := start.Add(*every)
		return c.createMaintenanceWindow(c.service.ID, fmt.Sprintf("every %v", every), start, end)
	}

	// TODO(adam): update if start/end times are off

	return maintWindow, nil
}

func (c *client) ensureMaintenanceWindow_PartialDay(partial *config.PartialDay, maintWindow *pagerduty.MaintenanceWindow) (*pagerduty.MaintenanceWindow, error) {
	if partial == nil {
		return nil, nil
	}
	if maintWindow == nil {
		if len(partial.Times) == 0 {
			return nil, errors.New("missing Times")
		}

		start, end, err := determineStartEnd(time.Now(), partial.Timezone, partial.Times[0])
		if err != nil {
			return nil, err
		}

		// TODO(adam): Need to create multiple windows...
		return c.createMaintenanceWindow(c.service.ID, "partial day", start, end)
	}

	// TODO(adam): update if start/end times are off

	return maintWindow, nil
}

func determineStartEnd(initial time.Time, tz string, times config.Times) (time.Time, time.Time, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parsing Timezone: %v", err)
	}

	start, err := times.StartTime()
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parsing StartTime: %v", err)
	}

	end, err := times.EndTime()
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parsing EndTime: %v", err)
	}

	today := initial.In(loc).Truncate(24 * time.Hour).Add(24 * time.Hour)

	// When times don't have a year/month/day adjust it to today.
	if start.Year() <= 1 {
		start = time.Date(today.Year(), today.Month(), today.Day(), start.Hour(), start.Minute(), 0, 0, loc)
	}
	if end.Year() <= 1 {
		end = time.Date(today.Year(), today.Month(), today.Day(), end.Hour(), end.Minute(), 0, 0, loc)
	}

	// When the start or end times are before now push it to the next day.
	now := initial.In(loc)
	if start.Before(now) || end.Before(now) {
		start = start.Add(24 * time.Hour)
		end = end.Add(24 * time.Hour)
	}

	return start.In(loc), end.In(loc), nil
}

// TODO(adam): endpoint check-in extends maint window

const (
	maintWindowTimeFormat = "2006-01-02T15:04:05-07:00" // Example: 2015-11-09T22:00:00-05:00
)

func (c *client) createMaintenanceWindow(serviceID, desc string, start, end time.Time) (*pagerduty.MaintenanceWindow, error) {
	ctx := context.Background()
	return c.underlying.CreateMaintenanceWindowWithContext(ctx, "from - todo", pagerduty.MaintenanceWindow{
		StartTime:   start.Format(maintWindowTimeFormat),
		EndTime:     end.Format(maintWindowTimeFormat),
		Description: desc,
		Services: []pagerduty.APIObject{
			{
				ID:   serviceID,
				Type: "service_reference",
			},
		},
	})
}
