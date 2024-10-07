package pd

import (
	"context"
	"fmt"

	"github.com/PagerDuty/go-pagerduty"
)

func (c *client) addEventsV2Integration(ctx context.Context, service *pagerduty.Service) (*pagerduty.Integration, error) {
	var request pagerduty.Integration
	request.Type = "events_api_v2_inbound_integration"

	integration, err := c.underlying.CreateIntegrationWithContext(ctx, service.ID, request)
	if err != nil {
		return nil, fmt.Errorf("creating integration: %w", err)
	}
	return integration, nil
}
