package check

import (
	"testing"

	"github.com/adamdecaf/deadcheck/internal/config"

	"github.com/stretchr/testify/require"
)

func TestMergeAlertConfigs(t *testing.T) {
	t.Run("pagerduty", func(t *testing.T) {
		var local config.Alert
		global := config.Alert{
			PagerDuty: &config.PagerDuty{
				ApiKey: "api-key",
			},
		}
		got := mergeAlertConfigs(local, global)
		require.Equal(t, "api-key", got.PagerDuty.ApiKey)

		local.PagerDuty = &config.PagerDuty{
			ApiKey: "other-key",
		}
		got = mergeAlertConfigs(local, global)
		require.Equal(t, "other-key", got.PagerDuty.ApiKey)

		global.PagerDuty.Urgency = "high"
		got = mergeAlertConfigs(local, global)
		require.Equal(t, "high", got.PagerDuty.Urgency)

		local.PagerDuty.Urgency = "low"
		got = mergeAlertConfigs(local, global)
		require.Equal(t, "low", got.PagerDuty.Urgency)
	})
}
