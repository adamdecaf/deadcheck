package config_test

import (
	"path/filepath"
	"testing"

	"github.com/adamdecaf/deadcheck/internal/config"

	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	conf, err := config.Load(filepath.Join("testdata", "empty.yaml"))
	require.NoError(t, err)
	require.NotNil(t, conf)
}

func TestReadFromEnv(t *testing.T) {
	healthchecksio := config.ReadHealthChecksIOFromEnv()
	require.Nil(t, healthchecksio)

	pd := config.ReadPagerDutyFromEnv()
	require.Nil(t, pd)

	slack := config.ReadSlackFromEnv()
	require.Nil(t, slack)
}
