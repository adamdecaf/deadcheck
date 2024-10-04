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

	"github.com/stretchr/testify/require"
)

func TestMaintenanceWindowUpdate(t *testing.T) {
	// TODO(adam):
}

func TestMaintenanceWindow__determineStartEnd(t *testing.T) {
	cst, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	// This initial time is after the maint window, so we put it for the next day.
	initial := time.Date(2022, 11, 9, 16, 39, 41, 0, cst) // 2022-11-09 at 16:39:41

	conf := config.Times{
		At:        "15:09",
		Tolerance: "2h33m",
	}
	start, end, err := determineStartEnd(initial, "America/New_York", conf)
	require.NoError(t, err)

	require.Equal(t, "2022-11-10T15:09:00-05:00", start.Format(time.RFC3339))
	require.Equal(t, "2022-11-10T17:42:00-05:00", end.Format(time.RFC3339))

	// The initial time is before today's maint window, so it's created for today.
	initial = time.Date(2022, 11, 9, 1, 0, 0, 0, cst) // 2022-11-09 at 01:00:00

	start, end, err = determineStartEnd(initial, "America/New_York", conf)
	require.NoError(t, err)

	require.Equal(t, "2022-11-09T15:09:00-05:00", start.Format(time.RFC3339))
	require.Equal(t, "2022-11-09T17:42:00-05:00", end.Format(time.RFC3339))
}
