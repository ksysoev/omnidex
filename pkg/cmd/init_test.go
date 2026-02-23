package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitCommand(t *testing.T) {
	cmd := InitCommand(BuildInfo{
		AppName: "app",
	})

	assert.Equal(t, "app", cmd.Use)
	assert.Contains(t, cmd.Short, "")
	assert.Contains(t, cmd.Long, "")

	require.Len(t, cmd.Commands(), 0)

	assert.Equal(t, "info", cmd.PersistentFlags().Lookup("log-level").DefValue)
	assert.Equal(t, "true", cmd.PersistentFlags().Lookup("log-text").DefValue)
	assert.Equal(t, "runtime/config.yml", cmd.PersistentFlags().Lookup("config").DefValue)
}
