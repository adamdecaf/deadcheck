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
	"testing"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/adamdecaf/deadcheck/internal/config"

	"github.com/moov-io/base"
	"github.com/stretchr/testify/require"
)

func TestEvents_TriggerInMaintWindow(t *testing.T) {
	ctx := context.Background()

	timezone := "America/New_York"
	loc, err := time.LoadLocation(timezone)
	require.NoError(t, err)

	now := time.Now().In(loc)

	conf := config.Check{
		ID:   base.ID(),
		Name: makeServiceName(t),
		Schedule: config.ScheduleConfig{
			Weekdays: &config.PartialDay{
				Timezone: timezone,
				Times: []config.Times{
					{
						Start: now.Add(time.Minute).Format("15:04"), // needs to be in the future
						End:   now.Add(2 * time.Hour).Format("15:04"),
					},
				},
			},
		},
	}
	pdc := newTestClient(t)

	// Setup a service and maintenance window such that triggering an event shouldn't alert
	service, err := pdc.Setup(ctx, conf)
	require.NoError(t, err)
	require.Empty(t, service.LastIncidentTimestamp) // verify no alert been triggered

	t.Logf("setup service %v named %v", service.ID, service.Name)

	// Verify maintenance windows
	resp, err := pdc.underlying.ListMaintenanceWindowsWithContext(ctx, pagerduty.ListMaintenanceWindowsOptions{
		Limit:      100,
		ServiceIDs: []string{service.ID},
	})
	require.NoError(t, err)
	require.Len(t, resp.MaintenanceWindows, 1)

	// Verify the maintenance window starts today
	start, err := time.Parse(time.RFC3339, resp.MaintenanceWindows[0].StartTime)
	require.NoError(t, err)
	require.Equal(t, now.Format("2006-01-02"), start.Format("2006-01-02"))

	// Wait for the maintenance window to start
	sleep := start.Add(time.Second).Sub(now)
	t.Logf("sleeping for %v until the maintenance window starts at %v", sleep, start.Format(time.RFC3339))
	time.Sleep(sleep)

	// Add our events v2 integration to the service
	integration, err := pdc.addEventsV2Integration(ctx, service)
	require.NoError(t, err)
	t.Logf("setup integration %v named %v", integration.ID, integration.Name)

	conf.Alert.PagerDuty = &config.PagerDuty{
		RoutingKey: integration.IntegrationKey,
	}

	// Trigger an event
	err = pdc.SetupTrigger(ctx, conf, service)
	require.NoError(t, err)

	time.Sleep(30 * time.Second)

	// Verify no incident
	found, err := pdc.findService(conf.Name)
	require.NoError(t, err)
	require.Equal(t, service.ID, found.ID)
	require.Empty(t, found.LastIncidentTimestamp)

	// Keep the service around until after, so we can debug it if needed.
	pdc.deleteService(service)
}
