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
	"testing"
	"time"

	"github.com/adamdecaf/deadcheck/internal/config"

	"github.com/moov-io/base"
	"github.com/stretchr/testify/require"
)

const escalationPolicy = "PF5G8GH" // 'adam - test'

func TestService__Every(t *testing.T) {
	every := 30 * time.Minute
	conf := config.Check{
		ID:   base.ID(),
		Name: t.Name(),
		Schedule: config.ScheduleConfig{
			Every: &every,
		},
		PagerDuty: &config.PagerDuty{
			EscalationPolicy: escalationPolicy,
		},
	}
	pdc := newTestClient(t)
	err := pdc.Setup(conf)
	require.NoError(t, err)

	defer deleteService(t, pdc)
}

func TestService__Weekdays(t *testing.T) {
	conf := config.Check{
		ID:   base.ID(),
		Name: t.Name(),
		Schedule: config.ScheduleConfig{
			Weekdays: &config.PartialDay{
				Timezone: "America/New_York",
				Times: []config.Times{
					{
						Start: "15:04",
						End:   "17:32",
					},
				},
			},
		},
		PagerDuty: &config.PagerDuty{
			EscalationPolicy: escalationPolicy,
		},
	}
	pdc := newTestClient(t)
	err := pdc.Setup(conf)
	require.NoError(t, err)

	defer deleteService(t, pdc)
}

func deleteService(t *testing.T, cc *client) {
	t.Helper()

	err := cc.deleteService()
	if err != nil {
		t.Fatal(err)
	}
}
