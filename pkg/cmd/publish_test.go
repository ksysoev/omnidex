package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunPublish_MissingURL(t *testing.T) {
	cmdFlags := &cmdFlags{LogLevel: "error", TextFormat: true}
	pubFlags := &publishFlags{APIKey: "key"}

	err := runPublish(t.Context(), cmdFlags, pubFlags)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--url")
}

func TestRunPublish_MissingAPIKey(t *testing.T) {
	cmdFlags := &cmdFlags{LogLevel: "error", TextFormat: true}
	pubFlags := &publishFlags{URL: "http://localhost"}

	err := runPublish(t.Context(), cmdFlags, pubFlags)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--api-key")
}

func TestRunPublish_MissingRepo(t *testing.T) {
	cmdFlags := &cmdFlags{LogLevel: "error", TextFormat: true}
	pubFlags := &publishFlags{
		URL:    "http://localhost",
		APIKey: "key",
	}

	err := runPublish(t.Context(), cmdFlags, pubFlags)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--repo")
}

func TestNewPublishCmd(t *testing.T) {
	flags := &cmdFlags{}
	cmd := newPublishCmd(flags)

	assert.Equal(t, "publish", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	// Verify all flags exist with correct defaults.
	urlFlag := cmd.Flags().Lookup("url")
	assert.NotNil(t, urlFlag)

	apiKeyFlag := cmd.Flags().Lookup("api-key")
	assert.NotNil(t, apiKeyFlag)

	docsPathFlag := cmd.Flags().Lookup("docs-path")
	assert.NotNil(t, docsPathFlag)
	assert.Equal(t, ".", docsPathFlag.DefValue)

	filePatternFlag := cmd.Flags().Lookup("file-pattern")
	assert.NotNil(t, filePatternFlag)
	assert.Equal(t, "**/*.md", filePatternFlag.DefValue)

	repoFlag := cmd.Flags().Lookup("repo")
	assert.NotNil(t, repoFlag)

	commitSHAFlag := cmd.Flags().Lookup("commit-sha")
	assert.NotNil(t, commitSHAFlag)
}
