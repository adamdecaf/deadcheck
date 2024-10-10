package pd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/google/uuid"
	"github.com/moov-io/base/log"
)

func (c *client) setupInitialIncident(ctx context.Context, service *pagerduty.Service, ep *pagerduty.EscalationPolicy) (*pagerduty.Incident, error) {
	req := pagerduty.ListIncidentsOptions{
		Limit:      100, // TODO(adam): pagination
		Statuses:   []string{"acknowledged", "triggered"},
		ServiceIDs: []string{service.ID},
		SortBy:     "created_at:DESC",
	}
	resp, err := c.underlying.ListIncidentsWithContext(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("listing incidents: %w", err)
	}

	for _, inc := range resp.Incidents {
		if strings.Contains(inc.Title, service.Name) {
			return &inc, nil
		}
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
		IncidentKey: uuid.NewString(),
		Urgency:     "low",
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

func (c *client) snoozeIncident(ctx context.Context, logger log.Logger, inc *pagerduty.Incident, service *pagerduty.Service, now time.Time, snooze time.Duration) error {
	// Only snooze an incident if we will snooze it further out into the future than it already is snoozed for.
	// This prevents a bug on startup where we wipe away check-ins (snoozes) by snoozing for a shorter duration.
	//
	//   Time   Action     Snooze
	//   5:01  check-in   +1d 5:05
	//   5:02   restart   +0d 5:05 (3 mins from now)
	for _, action := range inc.PendingActions {
		if strings.EqualFold("unacknowledge", action.Type) {
			futureUnsnooze, err := time.Parse(time.RFC3339, action.At)
			if err != nil {
				return fmt.Errorf("%s pending action had unexpected %s as timestamp: %w", action.Type, action.At, err)
			}

			distanceToSnooze := now.Add(snooze)
			if distanceToSnooze.Before(futureUnsnooze) {
				// Do nothing since we're already snoozed for longer than this snooze is asking for.
				return nil
			}
		}
	}

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

	logger.Info().Logf("snoozed %s for %v", service.Name, snooze)

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
