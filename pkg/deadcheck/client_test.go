package deadcheck

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClient_CheckIn(t *testing.T) {
	config := Config{
		BaseAddress: os.Getenv("DEADCHECK_BASE_ADDRESS"),
	}
	if config.BaseAddress == "" {
		t.Skip("DEADCHECK_BASE_ADDRESS is not provided")
	}

	client, err := NewClient(config)
	require.NoError(t, err)

	resp, err := client.CheckIn(context.Background(), "2pm-checkin") // from docs/examples/config.yaml
	require.NoError(t, err)

	t.Logf("Next Check-In Expected At: %v", resp.NextExpectedCheckIn.Format(time.RFC3339))
}
