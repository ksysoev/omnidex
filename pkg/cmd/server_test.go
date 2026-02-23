package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunCommand_InitLoggerFails(t *testing.T) {
	flags := &cmdFlags{
		LogLevel: "WrongLogLevel",
	}

	err := RunCommand(t.Context(), flags)
	assert.ErrorContains(t, err, "failed to init logger")
}

func TestRunCommand_LoadConfigFails(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	err := os.WriteFile(configPath, []byte("invalid config"), 0o600)
	require.NoError(t, err)

	flags := &cmdFlags{
		ConfigPath: configPath,
		LogLevel:   "info",
	}

	err = RunCommand(t.Context(), flags)
	assert.ErrorContains(t, err, "failed to load config:")
}

func TestRunCommand_APIFails(t *testing.T) {
	t.Setenv("API_LISTEN", "WRONG_ADDRESS_TO_LISTEN")
	err := RunCommand(t.Context(), &cmdFlags{LogLevel: "info"})
	assert.ErrorContains(t, err, "failed to run API service:")
}

func TestRunCommand_Success(t *testing.T) {
	t.Setenv("API_LISTEN", ":0")

	ctx, cancel := context.WithCancel(t.Context())

	go func() {
		time.Sleep(100 * time.Millisecond)

		cancel()
	}()

	err := RunCommand(ctx, &cmdFlags{LogLevel: "info"})
	assert.NoError(t, err, "expected RunCommand to succeed with valid configuration")
}
