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

func (c *client) setupMaintenanceWindows(check config.Check, service *pagerduty.Service) error {

	return nil
}

const (
	everyDescription       = "every"
	weekdaysDescription    = "weekdays"
	bankingDaysDescription = "banking-days"
)

func (c *client) findMaintenanceWindows(check config.Check, service *pagerduty.Service) ([]*pagerduty.MaintenanceWindow, error) {
	ctx := context.Background()
	resp, err := c.underlying.ListMaintenanceWindowsWithContext(ctx, pagerduty.ListMaintenanceWindowsOptions{
		Limit:      100,
		ServiceIDs: []string{service.ID},
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
		return c.createMaintenanceWindow()
	}
	return nil, nil
}

func (c *client) ensureMaintenanceWindow_PartialDay(partial *config.PartialDay, maintWindow *pagerduty.MaintenanceWindow) (*pagerduty.MaintenanceWindow, error) {
	if partial == nil {
		return nil, nil
	}
	if maintWindow == nil {
		return c.createMaintenanceWindow()
	}
	return nil, nil
}

func (c *client) createMaintenanceWindow() (*pagerduty.MaintenanceWindow, error) {
	return nil, nil
}

// func (c *Client) CreateMaintenanceWindowWithContext(ctx context.Context, from string, o MaintenanceWindow) (*MaintenanceWindow, error) {
//
// type MaintenanceWindow struct {
// 	StartTime      string      `json:"start_time"`
// 	EndTime        string      `json:"end_time"`
// 	Description    string      `json:"description"`
// 	Services       []APIObject `json:"services"`
