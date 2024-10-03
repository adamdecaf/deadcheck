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
	"strings"
	"time"

	"github.com/adamdecaf/deadcheck/internal/config"

	"github.com/PagerDuty/go-pagerduty"
)

func (c *client) SetupTrigger(ctx context.Context, check config.Check, service *pagerduty.Service) error {
	// Trigger an alert
	event, err := c.createEvent(ctx, check, service)
	if err != nil {
		return fmt.Errorf("triggering event: %w", err)
	}
	if !strings.EqualFold(event.Status, "success") {
		return fmt.Errorf("unexpected status when creating event: %#v", event)
	}
	return nil
}

const (
	pagerDutyEventTrigger = "trigger"
	pagerDutyEventResolve = "resolve"
)

func (c *client) createEvent(ctx context.Context, check config.Check, service *pagerduty.Service) (*pagerduty.V2EventResponse, error) {
	if check.Alert.PagerDuty == nil {
		return nil, errors.New("missing Alert.PagerDuty config")
	}

	event := &pagerduty.V2Event{
		RoutingKey: check.Alert.PagerDuty.RoutingKey,
		Action:     "trigger",
		Payload: &pagerduty.V2Payload{
			Summary:   fmt.Sprintf("%s did not check-in", check.Name),
			Source:    "my-app",
			Severity:  "critical",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Details: map[string]interface{}{
				"detail": "CPU usage exceeded 90%",
			},
		},
	}
	return c.underlying.ManageEventWithContext(ctx, event)
}
