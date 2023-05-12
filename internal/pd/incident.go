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
	"strings"

	"github.com/adamdecaf/deadcheck/internal/config"

	"github.com/PagerDuty/go-pagerduty"
)

func (c *client) setupIncident(check config.Check, service *pagerduty.Service) (*pagerduty.Incident, error) {
	opts := pagerduty.ListIncidentsOptions{
		ServiceIDs: []string{
			service.ID,
		},
	}
	ctx := context.Background()
	resp, err := c.underlying.ListIncidentsWithContext(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("problem listing %s incidents: %w", service.Name, err)
	}

	// Ensure an incident has been created
	for i := range resp.Incidents {
		inc := resp.Incidents[i]

		// Look for a triggered incident of the same name.
		if strings.EqualFold(inc.Title, check.Name) && strings.EqualFold(inc.Status, "triggered") {
			return &inc, nil // incident found, so we're done
		}
	}

	// Create the incident we're looking for
	return c.createIncident(check, service)
}

// TODO(adam): We might need to retrigger/recreate the incident (if humans resolve) on setup

func (c *client) createIncident(check config.Check, service *pagerduty.Service) (*pagerduty.Incident, error) {
	if check.PagerDuty == nil || check.PagerDuty.EscalationPolicy == "" {
		return nil, fmt.Errorf("missing config for PagerDuty.EscalationPolicy on %s", check.Name)
	}

	opts := &pagerduty.CreateIncidentOptions{
		Title: check.Name,
		Service: &pagerduty.APIReference{
			ID:   service.ID,
			Type: "service_reference",
		},
		EscalationPolicy: &pagerduty.APIReference{
			ID:   check.PagerDuty.EscalationPolicy,
			Type: "escalation_policy_reference",
		},
	}

	ctx := context.Background()
	inc, err := c.underlying.CreateIncidentWithContext(ctx, check.PagerDuty.From, opts)
	if err != nil {
		return nil, fmt.Errorf("creating incident: %w", err)
	}
	return inc, nil
}
