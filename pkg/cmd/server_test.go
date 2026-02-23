package cmd

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRunCommand_InitLoggerFails(t *testing.T) {
	flags := &cmdFlags{
		LogLevel: "WrongLogLevel",
	}

	err := RunCommand(t.Context(), flags)
	assert.ErrorContains(t, err, "failed to init logger")
}

func TestRunCommand_Success(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "repos")
	indexPath := filepath.Join(tmpDir, "search.bleve")

	t.Setenv("API_LISTEN", ":0")
	t.Setenv("STORAGE_PATH", storagePath)
	t.Setenv("SEARCH_INDEX_PATH", indexPath)

	ctx, cancel := context.WithCancel(t.Context())

	go func() {
		time.Sleep(100 * time.Millisecond)

		cancel()
	}()

	err := RunCommand(ctx, &cmdFlags{LogLevel: "info"})
	assert.NoError(t, err, "expected RunCommand to succeed with valid configuration")
}
