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

	"github.com/adamdecaf/deadcheck/internal/config"

	"github.com/PagerDuty/go-pagerduty"
)

func (c *client) Setup(check config.Check) error {
	// List Services, grab by name, cache for future updates
	service, err := c.findService(check.Name)
	if err != nil {
		return fmt.Errorf("finding pagerduty service: %v", err)
	}
	if service == nil {
		service, err = c.createService(check)
		if err != nil {
			return fmt.Errorf("creating pagerduty service: %v", err)
		}
	}
	// Cache the service for future calls
	if service != nil {
		c.service = service
	}
	return c.setupMaintenanceWindows(check)
}

func (c *client) findService(name string) (*pagerduty.Service, error) {
	// TODO(adam): pagination
	ctx := context.Background()
	resp, err := c.underlying.ListServicesWithContext(ctx, pagerduty.ListServiceOptions{
		Limit:  100,
		Offset: 0,
	})
	if err != nil {
		return nil, fmt.Errorf("listing services: %v", err)
	}

	for i := range resp.Services {
		if resp.Services[i].Name == name {
			return &resp.Services[i], nil
		}
	}

	return nil, nil
}

func (c *client) createService(check config.Check) (*pagerduty.Service, error) {
	svc := pagerduty.Service{
		Name:        check.Name,
		Description: check.Description,
	}

	if check.PagerDuty.EscalationPolicy != "" {
		svc.EscalationPolicy.ID = check.PagerDuty.EscalationPolicy
		svc.EscalationPolicy.Type = "escalation_policy_reference"
	}

	ctx := context.Background() // TODO(adam):
	return c.underlying.CreateServiceWithContext(ctx, svc)
}

func (c *client) deleteService() error {
	if c == nil || c.service == nil {
		return nil
	}

	ctx := context.Background()
	return c.underlying.DeleteServiceWithContext(ctx, c.service.ID)
}
