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
	"os"
	"strings"
	"testing"

	"github.com/adamdecaf/deadcheck/internal/config"

	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T) *client {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping because -short is set")
	}

	apiKey := strings.TrimSpace(os.Getenv("DEADCHECK_PAGERDUTY_API_KEY"))
	if apiKey == "" {
		t.Skip("no DEADCHECK_PAGERDUTY_API_KEY specified, skipping test...")
	}

	escPolicy := os.Getenv("DEADCHECK_ESCALATION_POLICY")
	if escPolicy == "" {
		t.Skip("no DEADCHECK_ESCALATION_POLICY specified, skipping test...")
	}

	cc, err := NewClient(&config.PagerDuty{
		ApiKey:           apiKey,
		EscalationPolicy: escPolicy,
		RoutingKey:       os.Getenv("DEADCHECK_ROUTING_KEY"),
	})
	require.NoError(t, err)

	cl, ok := cc.(*client)
	require.True(t, ok)

	return cl
}

func TestClient(t *testing.T) {
	pdc := newTestClient(t)
	require.NoError(t, pdc.ping())
}
