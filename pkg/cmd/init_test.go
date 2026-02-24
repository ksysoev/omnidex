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
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	require.Len(t, cmd.Commands(), 3)

	subCmds := cmd.Commands()
	names := make([]string, 0, len(subCmds))

	for _, sub := range subCmds {
		names = append(names, sub.Use)
	}

	assert.Contains(t, names, "serve")
	assert.Contains(t, names, "health")
	assert.Contains(t, names, "publish")

	assert.Equal(t, "info", cmd.PersistentFlags().Lookup("log-level").DefValue)
	assert.Equal(t, "true", cmd.PersistentFlags().Lookup("log-text").DefValue)
	assert.Equal(t, "runtime/config.yml", cmd.PersistentFlags().Lookup("config").DefValue)
}
