package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/ksysoev/omnidex/pkg/core"
	"github.com/spf13/cobra"
)

const publishRequestTimeout = 30 * time.Second

type publishFlags struct {
	URL         string
	APIKey      string //nolint:gosec // Not a credential, just a flag name for the CLI
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

// runPublish orchestrates the publish workflow: validates inputs, collects files,
// builds the ingest payload, and sends it to the Omnidex server.
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

	slog.Info("Publishing documentation",
		"url", pubFlags.URL,
		"docs_path", pubFlags.DocsPath,
		"file_pattern", pubFlags.FilePattern,
		"repo", pubFlags.Repo,
		"commit_sha", pubFlags.CommitSHA,
	)

	files, err := collectFiles(pubFlags.DocsPath, pubFlags.FilePattern)
	if err != nil {
		return fmt.Errorf("failed to collect files: %w", err)
	}

	if len(files) == 0 {
		slog.Warn("No files matched the pattern", "path", pubFlags.DocsPath, "pattern", pubFlags.FilePattern)
		return nil
	}

	slog.Info("Collected documentation files", "count", len(files))

	req := buildIngestRequest(pubFlags.Repo, pubFlags.CommitSHA, files)

	resp, err := sendIngestRequest(ctx, pubFlags.URL, pubFlags.APIKey, req)
	if err != nil {
		return fmt.Errorf("failed to publish documentation: %w", err)
	}

	slog.Info("Documentation published successfully", "indexed", resp.Indexed, "deleted", resp.Deleted)

	return nil
}

// collectFiles walks the directory at docsPath and returns the content of all files
// matching the given glob pattern. The returned map keys are relative paths from docsPath.
func collectFiles(docsPath, filePattern string) (map[string]string, error) {
	files := make(map[string]string)

	err := filepath.WalkDir(docsPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(docsPath, path)
		if err != nil {
			return fmt.Errorf("failed to compute relative path for %s: %w", path, err)
		}

		// Use forward slashes for consistent matching across platforms.
		relPath = filepath.ToSlash(relPath)

		matched, err := doublestar.Match(filePattern, relPath)
		if err != nil {
			return fmt.Errorf("invalid glob pattern %q: %w", filePattern, err)
		}

		if !matched {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		files[relPath] = string(content)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", docsPath, err)
	}

	return files, nil
}

// buildIngestRequest constructs an IngestRequest from the collected file contents.
// All documents are set to action "upsert".
func buildIngestRequest(repo, commitSHA string, files map[string]string) core.IngestRequest {
	documents := make([]core.IngestDocument, 0, len(files))

	// Sort keys for deterministic ordering.
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}

	sort.Strings(paths)

	for _, p := range paths {
		documents = append(documents, core.IngestDocument{
			Path:    p,
			Content: files[p],
			Action:  "upsert",
		})
	}

	return core.IngestRequest{
		Repo:      repo,
		CommitSHA: commitSHA,
		Documents: documents,
	}
}

// sendIngestRequest POSTs the IngestRequest to the Omnidex server's ingest API endpoint.
// It returns the parsed IngestResponse or an error if the request fails or the server returns a non-2xx status.
func sendIngestRequest(ctx context.Context, baseURL, apiKey string, req core.IngestRequest) (*core.IngestResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := strings.TrimRight(baseURL, "/") + "/api/v1/docs"

	ctx, cancel := context.WithTimeout(ctx, publishRequestTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: publishRequestTimeout}

	resp, err := client.Do(httpReq) //nolint:gosec // URL is intentionally user-provided via CLI flag
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("server returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var ingestResp core.IngestResponse
	if err := json.Unmarshal(respBody, &ingestResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &ingestResp, nil
}
