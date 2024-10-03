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
						Start: "12:07",
						End:   "17:32",
					},
				},
			},
		},
	}
	pdc := newTestClient(t)

	service, err := pdc.Setup(ctx, conf)
	require.NoError(t, err)

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

	t.Cleanup(func() {
		pdc.deleteService(service)
	})
}

func makeServiceName(t *testing.T) string {
	return fmt.Sprintf("%s_%d", t.Name(), time.Now().In(time.UTC).Unix())
}
