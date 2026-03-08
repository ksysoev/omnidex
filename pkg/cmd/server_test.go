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

func TestRunCommand_LoadConfigFails(t *testing.T) {
	flags := &cmdFlags{
		LogLevel:   "info",
		ConfigPath: "/nonexistent/path/config.yaml",
	}

	err := RunCommand(t.Context(), flags)
	assert.ErrorContains(t, err, "failed to load config")
}

func TestRunCommand_InvalidStoragePath(t *testing.T) {
	tmpDir := t.TempDir()

	// Set STORAGE_PATH to a path inside a file (not a dir), which should fail.
	invalidPath := filepath.Join(tmpDir, "not-a-dir")

	// Create a regular file at invalidPath so subdirectory creation will fail.
	err := writeFile(invalidPath)
	require.NoError(t, err)

	storagePath := filepath.Join(invalidPath, "repos")

	t.Setenv("API_LISTEN", ":0")
	t.Setenv("STORAGE_PATH", storagePath)
	t.Setenv("SEARCH_INDEX_PATH", filepath.Join(tmpDir, "search.bleve"))

	err = RunCommand(t.Context(), &cmdFlags{LogLevel: "info"})
	assert.Error(t, err)
}

func TestRunCommand_UnknownStorageType(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "search.bleve")

	t.Setenv("API_LISTEN", ":0")
	t.Setenv("STORAGE_TYPE", "unknowntype")
	t.Setenv("SEARCH_INDEX_PATH", indexPath)

	err := RunCommand(t.Context(), &cmdFlags{LogLevel: "info"})
	assert.ErrorContains(t, err, "unknown storage type")
}

func TestRunCommand_S3StorageType(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "search.bleve")

	t.Setenv("API_LISTEN", ":0")
	t.Setenv("STORAGE_TYPE", "s3")
	t.Setenv("SEARCH_INDEX_PATH", indexPath)

	ctx, cancel := context.WithCancel(t.Context())

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	// s3store.New succeeds (no network call at construction time); the server
	// exits cleanly when the context is cancelled.
	err := RunCommand(ctx, &cmdFlags{LogLevel: "info"})
	assert.NoError(t, err)
}

// writeFile creates a regular file at the given path.
func writeFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}

	return f.Close()
}
