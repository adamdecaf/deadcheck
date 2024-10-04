package pd

import (
	"context"
	"fmt"
	"strings"

	"github.com/PagerDuty/go-pagerduty"
)

type escalationPolicySetup struct {
	id   string
	name string

	userIDs []string
}

func (c *client) findEscalationPolicy(ctx context.Context, setup escalationPolicySetup) (*pagerduty.EscalationPolicy, error) {
	opts := pagerduty.ListEscalationPoliciesOptions{
		Limit: 100,
	}
	eps, err := c.underlying.ListEscalationPoliciesWithContext(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("listing escalation policies: %w", err)
	}
	for _, ep := range eps.EscalationPolicies {
		if ep.ID == setup.id {
			return &ep, nil
		}
		if strings.EqualFold(ep.Name, setup.name) {
			return &ep, nil
		}
	}

	// Can't find it so create one
	req := pagerduty.EscalationPolicy{
		Name:        setup.name,
		Description: "managed by deadcheck, DO NOT MODIFY",
	}
	// Add an escalation rule
	for _, userID := range setup.userIDs {
		rule := pagerduty.EscalationRule{
			Delay: 1,
			Targets: []pagerduty.APIObject{
				{
					Type: "user_reference",
					ID:   userID,
				},
			},
		}
		req.EscalationRules = append(req.EscalationRules, rule)
	}

	ep, err := c.underlying.CreateEscalationPolicyWithContext(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("creating escalation policy: %w", err)
	}
	return ep, nil
}
