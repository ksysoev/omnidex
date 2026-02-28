// Package publisher handles collecting documentation files and publishing them
// to an Omnidex instance via the ingest API.
package publisher

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
)

const requestTimeout = 30 * time.Second

// Publisher handles publishing documentation to an Omnidex instance.
type Publisher struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
}

// New creates a new Publisher configured with the given base URL and API key.
func New(baseURL, apiKey string) *Publisher {
	return &Publisher{
		httpClient: &http.Client{Timeout: requestTimeout},
		baseURL:    baseURL,
		apiKey:     apiKey,
	}
}

// Publish collects documentation files from docsPath matching filePattern,
// builds an ingest request, and sends it to the Omnidex server.
// When sync is true, the server will remove any stored documents not present in this publish.
// It returns the server response or an error if any step fails.
func (p *Publisher) Publish(ctx context.Context, docsPath, filePattern, repo, commitSHA string, sync bool) (*core.IngestResponse, error) {
	files, err := CollectFiles(docsPath, filePattern)
	if err != nil {
		return nil, fmt.Errorf("failed to collect files: %w", err)
	}

	if len(files) == 0 {
		slog.Warn("No files matched the pattern", "path", docsPath, "pattern", filePattern)
		return &core.IngestResponse{}, nil
	}

	slog.Info("Collected documentation files", "count", len(files))

	req := BuildIngestRequest(repo, commitSHA, files, sync)

	resp, err := p.SendIngestRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to publish documentation: %w", err)
	}

	return resp, nil
}

// CollectFiles walks the directory at docsPath and returns the content of all files
// matching the given glob pattern. The returned map keys are relative paths from docsPath
// using forward slashes.
func CollectFiles(docsPath, filePattern string) (map[string]string, error) {
	info, err := os.Stat(docsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat docs path %s: %w", docsPath, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("docs path %s is not a directory", docsPath)
	}

	// Normalize the file pattern to use forward slashes so that patterns with
	// backslashes (common on Windows) match the forward-slash normalized relPath.
	filePattern = filepath.ToSlash(filePattern)

	files := make(map[string]string)

	err = filepath.WalkDir(docsPath, func(path string, d fs.DirEntry, err error) error {
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

// BuildIngestRequest constructs an IngestRequest from the collected file contents.
// All documents are set to action "upsert". Documents are sorted by path for deterministic ordering.
// When sync is true, the server will treat this as the complete document set and remove stale entries.
func BuildIngestRequest(repo, commitSHA string, files map[string]string, sync bool) core.IngestRequest {
	documents := make([]core.IngestDocument, 0, len(files))

	// Sort keys for deterministic ordering.
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}

	sort.Strings(paths)

	for _, p := range paths {
		ct := core.DetectContentType(p, []byte(files[p]))

		// Skip files whose content type could not be determined (e.g. arbitrary
		// YAML/JSON that is not an OpenAPI spec).
		if ct == "" {
			slog.Debug("skipping file with unrecognized content type", "path", p)
			continue
		}

		documents = append(documents, core.IngestDocument{
			Path:        p,
			Content:     files[p],
			Action:      "upsert",
			ContentType: ct,
		})
	}

	return core.IngestRequest{
		Repo:      repo,
		CommitSHA: commitSHA,
		Documents: documents,
		Sync:      sync,
	}
}

// SendIngestRequest POSTs the IngestRequest to the Omnidex server's ingest API endpoint.
// It returns the parsed IngestResponse or an error if the request fails or the server returns a non-2xx status.
func (p *Publisher) SendIngestRequest(ctx context.Context, req core.IngestRequest) (*core.IngestResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := strings.TrimRight(p.baseURL, "/") + "/api/v1/docs"

	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(httpReq) //nolint:gosec // URL is intentionally user-provided via CLI flag
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
