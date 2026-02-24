package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/ksysoev/omnidex/pkg/publisher"
	"github.com/spf13/cobra"
)

type publishFlags struct {
	URL         string
	APIKey      string //nolint:gosec // Suppresses false positive on field name; value is a runtime-provided bearer token from CLI/env.
	DocsPath    string
	FilePattern string
	Repo        string
	CommitSHA   string
}

// newPublishCmd creates a cobra command that publishes documentation files to an Omnidex instance.
// It walks the docs directory, matches files against a glob pattern, and POSTs them to the ingest API.
func newPublishCmd(flags *cmdFlags) *cobra.Command {
	pubFlags := &publishFlags{}

	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish documentation files to an Omnidex instance",
		Long:  "Walk a documentation directory, match files by glob pattern, and publish them to an Omnidex instance via the ingest API.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPublish(cmd.Context(), flags, pubFlags)
		},
	}

	cmd.Flags().StringVar(&pubFlags.URL, "url", "", "base URL of the Omnidex instance")
	cmd.Flags().StringVar(&pubFlags.APIKey, "api-key", "", "Bearer token for authentication")
	cmd.Flags().StringVar(&pubFlags.DocsPath, "docs-path", ".", "path to the documentation directory")
	cmd.Flags().StringVar(&pubFlags.FilePattern, "file-pattern", "**/*.md", "glob pattern for documentation files")
	cmd.Flags().StringVar(&pubFlags.Repo, "repo", "", "repository identifier (owner/repo)")
	cmd.Flags().StringVar(&pubFlags.CommitSHA, "commit-sha", "", "git commit SHA")

	// Bind environment variables as defaults for flags that are not explicitly set.
	bindEnvDefaults(cmd, pubFlags)

	return cmd
}

// bindEnvDefaults sets flag defaults from environment variables when the flags are not explicitly provided.
func bindEnvDefaults(cmd *cobra.Command, _ *publishFlags) {
	envBindings := map[string]string{
		"url":          "OMNIDEX_URL",
		"api-key":      "OMNIDEX_API_KEY",
		"docs-path":    "DOCS_PATH",
		"file-pattern": "FILE_PATTERN",
		"repo":         "GITHUB_REPOSITORY",
		"commit-sha":   "GITHUB_SHA",
	}

	for flagName, envVar := range envBindings {
		if val := os.Getenv(envVar); val != "" {
			if err := cmd.Flags().Set(flagName, val); err != nil {
				slog.Warn("failed to set flag from env", "flag", flagName, "env", envVar, "error", err)
			}
		}
	}
}

// runPublish validates inputs and delegates the publish workflow to the publisher package.
func runPublish(ctx context.Context, flags *cmdFlags, pubFlags *publishFlags) error {
	if err := initLogger(flags); err != nil {
		return fmt.Errorf("failed to init logger: %w", err)
	}

	if pubFlags.URL == "" {
		return fmt.Errorf("--url (or OMNIDEX_URL) is required")
	}

	if pubFlags.APIKey == "" {
		return fmt.Errorf("--api-key (or OMNIDEX_API_KEY) is required")
	}

	if pubFlags.Repo == "" {
		return fmt.Errorf("--repo (or GITHUB_REPOSITORY) is required")
	}

	slog.Info("Publishing documentation",
		"url", pubFlags.URL,
		"docs_path", pubFlags.DocsPath,
		"file_pattern", pubFlags.FilePattern,
		"repo", pubFlags.Repo,
		"commit_sha", pubFlags.CommitSHA,
	)

	pub := publisher.New(pubFlags.URL, pubFlags.APIKey)

	resp, err := pub.Publish(ctx, pubFlags.DocsPath, pubFlags.FilePattern, pubFlags.Repo, pubFlags.CommitSHA)
	if err != nil {
		return err
	}

	slog.Info("Documentation published successfully", "indexed", resp.Indexed, "deleted", resp.Deleted)

	return nil
}
