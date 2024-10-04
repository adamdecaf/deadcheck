package pd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/PagerDuty/go-pagerduty"
)

func (c *client) setupInitialIncident(ctx context.Context, service *pagerduty.Service, ep *pagerduty.EscalationPolicy) (*pagerduty.Incident, error) {
	req := pagerduty.ListIncidentsOptions{
		Statuses:   []string{"acknowledged", "triggered"},
		ServiceIDs: []string{service.ID},
	}
	resp, err := c.underlying.ListIncidentsWithContext(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("listing incidents: %w", err)
	}

	for _, inc := range resp.Incidents {
		if strings.Contains(inc.Body.Details, "check-in") {
			return &inc, nil
		}
	}

	return c.createInitialIncident(ctx, service, ep)
}

func (c *client) createInitialIncident(ctx context.Context, service *pagerduty.Service, ep *pagerduty.EscalationPolicy) (*pagerduty.Incident, error) {
	req := &pagerduty.CreateIncidentOptions{
		Title: fmt.Sprintf("Creating ongoing incdient for %s", service.Name),
		Body: &pagerduty.APIDetails{
			Details: "This incident will be active and used by deadcheck to alert you when check-ins do not occur as expected. Deadcheck will update this incident to reflect the current status of check-in.",
		},
		Urgency: "low",
		EscalationPolicy: &pagerduty.APIReference{
			ID:   ep.ID,
			Type: "escalation_policy",
		},
		Service: &pagerduty.APIReference{
			ID:   service.ID,
			Type: "service",
		},
	}
	inc, err := c.underlying.CreateIncidentWithContext(ctx, c.pdConfig.From, req)
	if err != nil {
		return nil, fmt.Errorf("creating incident: %w", err)
	}
	return inc, nil
}

func (c *client) snoozeIncident(ctx context.Context, inc *pagerduty.Incident, service *pagerduty.Service, snooze time.Duration) error {
	// Ack the incident
	update := []pagerduty.ManageIncidentsOptions{
		{
			ID:     inc.ID,
			Status: "acknowledged",
		},
	}
	_, err := c.underlying.ManageIncidentsWithContext(ctx, c.pdConfig.From, update)
	if err != nil {
		return fmt.Errorf("incident acknowledged: %w", err)
	}

	// Snooze the incident
	inc, err = c.underlying.SnoozeIncidentWithContext(ctx, inc.ID, c.pdConfig.From, uint(snooze.Seconds()))
	if err != nil {
		return fmt.Errorf("snoozing incident: %w", err)
	}

	// Update the incident details for humans to read
	expectedCheckin := time.Now().In(time.UTC).Add(snooze).Format("2006-01-02 15:04 UTC")
	update = []pagerduty.ManageIncidentsOptions{
		{
			ID:      inc.ID,
			Title:   fmt.Sprintf("%s did not check-in, expected check-in at %v", service.Name, expectedCheckin),
			Urgency: "high",
		},
	}
	_, err = c.underlying.ManageIncidentsWithContext(ctx, c.pdConfig.From, update)
	if err != nil {
		return fmt.Errorf("updating incidnet for after snooze: %w", err)
	}
	return nil
}

func (c *client) resolveIncident(ctx context.Context, inc *pagerduty.Incident) error {
	update := []pagerduty.ManageIncidentsOptions{
		{
			ID:     inc.ID,
			Status: "resolved",
		},
	}
	_, err := c.underlying.ManageIncidentsWithContext(ctx, c.pdConfig.From, update)
	if err != nil {
		return fmt.Errorf("resolving incident: %w", err)
	}
	return nil
}
