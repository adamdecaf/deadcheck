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
	"testing"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/adamdecaf/deadcheck/internal/config"

	"github.com/moov-io/base"
	"github.com/stretchr/testify/require"
)

func TestService__Setup(t *testing.T) {
	ctx := context.Background()

	conf := config.Check{
		ID:   base.ID(),
		Name: makeServiceName(t),
		Schedule: config.ScheduleConfig{
			Weekdays: &config.PartialDay{
				Timezone: "America/New_York",
				Times: []config.Times{
					{
						At:        "12:07",
						Tolerance: "5h25m",
					},
				},
			},
		},
	}
	pdc := newTestClient(t)

	service, err := pdc.setupService(ctx, conf)
	require.NoError(t, err)
	t.Cleanup(func() {
		pdc.deleteService(service)
	})

	t.Logf("setup service %v named %v", service.ID, service.Name)

	// Verify the service is in maintenance mode
	found, err := pdc.findService(conf.Name)
	require.NoError(t, err)
	require.Equal(t, service.ID, found.ID)

	// Check maintenance windows
	resp, err := pdc.underlying.ListMaintenanceWindowsWithContext(ctx, pagerduty.ListMaintenanceWindowsOptions{
		Limit:      100,
		ServiceIDs: []string{service.ID},
	})
	require.NoError(t, err)
	require.Len(t, resp.MaintenanceWindows, 1)

	mw := resp.MaintenanceWindows[0]

	loc, err := time.LoadLocation(conf.Schedule.Weekdays.Timezone)
	require.NoError(t, err)

	start, err := time.Parse(time.RFC3339, mw.StartTime)
	require.NoError(t, err)
	require.Equal(t, "12:07", start.In(loc).Format("15:04"))

	end, err := time.Parse(time.RFC3339, mw.EndTime)
	require.NoError(t, err)
	require.Equal(t, "17:32", end.In(loc).Format("15:04"))
}

// TODO(adam): V2Event triggers created during a MW are ignored by PD, so how are we going to prompt it to alert right as a MW ends?
// TODO(adam): create an incident during the MW? Can we set a future dated start time?

// Can we create an incident during the MW
// Then snooze it for the MW duration?

// Do we even need MW windows?
// SnoozeIncidentWithContext(ctx context.Context, id string, duration uint) (*Incident, error)
//  can check .PendingActions
//
// Create the incident with an empty EscalationPolicy
// Then snooze it for the MW time, and reassign to escalation policy?
//
// On check-in snooze again for however long

func TestService_SnoozedIncident(t *testing.T) {
	skipInCI(t) // This test creates real alerts, so don't run it in CI

	ctx := context.Background()

	conf := config.Check{
		ID:   base.ID(),
		Name: makeServiceName(t),
	}
	pdc := newTestClient(t)

	service, err := pdc.setupService(ctx, conf)
	require.NoError(t, err)
	t.Cleanup(func() {
		pdc.deleteService(service)
	})

	t.Logf("setup service %v named %v", service.ID, service.Name)

	// Create a new escalation policy with nothing routed
	ep, err := pdc.findEscalationPolicy(ctx, escalationPolicySetup{
		id: defaultEscalationPolicy,
	})
	require.NoError(t, err)

	// Create an incident
	inc, err := pdc.setupInitialIncident(ctx, service, ep)
	require.NoError(t, err)

	t.Logf("created incident %v escalating to %v", inc.ID, ep.Name)

	err = pdc.snoozeIncident(ctx, inc, service, time.Hour)
	require.NoError(t, err)

	// Resolve incident
	err = pdc.resolveIncident(ctx, inc)
	require.NoError(t, err)
}

func makeServiceName(t *testing.T) string {
	return fmt.Sprintf("%s_%d", t.Name(), time.Now().In(time.UTC).Unix())
}
