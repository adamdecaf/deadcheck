package pd

import (
	"context"
	"errors"
	"fmt"

	"github.com/adamdecaf/deadcheck/internal/config"

	"github.com/PagerDuty/go-pagerduty"
)

func (c *client) setupService(ctx context.Context, check config.Check) (*pagerduty.Service, error) {
	if check.Alert.PagerDuty == nil {
		check.Alert.PagerDuty = &c.pdConfig
	}

	// List Services, grab by name, cache for future updates
	service, err := c.findService(ctx, check.Name)
	if err != nil {
		return nil, fmt.Errorf("finding pagerduty service: %w", err)
	}
	if service == nil {
		service, err = c.createService(ctx, check)
		if err != nil {
			return nil, fmt.Errorf("creating pagerduty service: %w", err)
		}
	}
	if service == nil {
		return nil, errors.New("no service was setup")
	}

	return service, nil
}

func (c *client) findService(ctx context.Context, name string) (*pagerduty.Service, error) {
	resp, err := c.underlying.ListServicesWithContext(ctx, pagerduty.ListServiceOptions{
		Limit:  100, // TODO(adam): pagination
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

func (c *client) createService(ctx context.Context, check config.Check) (*pagerduty.Service, error) {
	svc := pagerduty.Service{
		Name:        check.Name,
		Description: check.Description,
	}

	if check.Alert.PagerDuty.EscalationPolicy != "" {
		svc.EscalationPolicy.ID = check.Alert.PagerDuty.EscalationPolicy
		svc.EscalationPolicy.Type = "escalation_policy_reference"
	}

	return c.underlying.CreateServiceWithContext(ctx, svc)
}

func (c *client) deleteService(ctx context.Context, service *pagerduty.Service) error {
	if c == nil || service == nil {
		return nil
	}

	return c.underlying.DeleteServiceWithContext(ctx, service.ID)
}
