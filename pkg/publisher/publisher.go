// Package publisher handles collecting documentation files and publishing them
// to an Omnidex instance via the ingest API.
package publisher

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/ksysoev/omnidex/pkg/core"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
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
// Referenced images are automatically detected in markdown files and bundled as assets.
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

	assets, err := CollectAssets(docsPath, files)
	if err != nil {
		return nil, fmt.Errorf("failed to collect assets: %w", err)
	}

	if len(assets) > 0 {
		slog.Info("Collected referenced assets", "count", len(assets))
	}

	req := BuildIngestRequest(repo, commitSHA, files, assets, sync)

	resp, err := p.SendIngestRequest(ctx, &req)
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

// BuildIngestRequest constructs an IngestRequest from the collected file contents and assets.
// All documents and assets are set to action "upsert". Entries are sorted by path for deterministic ordering.
// When sync is true, the server will treat this as the complete document set and remove stale entries.
func BuildIngestRequest(repo, commitSHA string, files map[string]string, assets map[string][]byte, sync bool) core.IngestRequest {
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

	// Build asset entries with base64 encoding.
	// Always set a non-nil pointer so the server knows this client is aware of
	// the assets field and can correctly run stale-asset sync cleanup.
	ingestAssets := make([]core.IngestAsset, 0, len(assets))

	if len(assets) > 0 {
		assetPaths := make([]string, 0, len(assets))
		for p := range assets {
			assetPaths = append(assetPaths, p)
		}

		sort.Strings(assetPaths)

		for _, p := range assetPaths {
			ingestAssets = append(ingestAssets, core.IngestAsset{
				Path:    p,
				Content: base64.StdEncoding.EncodeToString(assets[p]),
				Action:  "upsert",
			})
		}
	}

	return core.IngestRequest{
		Repo:      repo,
		CommitSHA: commitSHA,
		Documents: documents,
		Assets:    &ingestAssets,
		Sync:      sync,
	}
}

// ExtractImageRefs parses markdown content and returns the destination URLs
// of all image nodes. Only relative paths are returned; absolute URLs
// (http, https, //, data:, /) are filtered out.
func ExtractImageRefs(content string) []string {
	md := goldmark.New()
	reader := text.NewReader([]byte(content))
	doc := md.Parser().Parse(reader)

	var refs []string

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		img, ok := n.(*ast.Image)
		if !ok {
			return ast.WalkContinue, nil
		}

		dest := string(img.Destination)
		if dest == "" {
			return ast.WalkContinue, nil
		}

		// Skip absolute URLs and data URIs.
		if strings.HasPrefix(dest, "http://") ||
			strings.HasPrefix(dest, "https://") ||
			strings.HasPrefix(dest, "//") ||
			strings.HasPrefix(dest, "data:") ||
			strings.HasPrefix(dest, "/") {
			return ast.WalkContinue, nil
		}

		refs = append(refs, dest)

		return ast.WalkContinue, nil
	})

	return refs
}

// CollectAssets scans markdown documents for relative image references, reads the
// referenced files from disk, and returns a map of resolved asset paths to their binary content.
// Paths are resolved relative to each markdown file's directory within docsPath.
// References that escape the docsPath boundary are logged and skipped.
func CollectAssets(docsPath string, docs map[string]string) (map[string][]byte, error) {
	assets := make(map[string][]byte)

	for docRelPath, content := range docs {
		ct := core.DetectContentType(docRelPath, []byte(content))
		if ct != core.ContentTypeMarkdown {
			continue
		}

		refs := ExtractImageRefs(content)

		for _, ref := range refs {
			// Parse the ref so only the path component is used for filesystem
			// resolution. A ref like "sprite.svg#icon" or "img.png?raw=1" must
			// resolve to "sprite.svg" / "img.png" on disk; the fragment and query
			// string are not part of the filename. This matches what
			// RewriteImageURLs does when building the asset URL.
			u, err := url.Parse(ref)
			if err != nil {
				slog.Warn("skipping malformed image reference",
					"doc", docRelPath, "ref", ref, "error", err)

				continue
			}

			refPath := u.Path

			// Resolve relative to the markdown file's directory.
			docDir := path.Dir(docRelPath)
			resolved := path.Clean(path.Join(docDir, refPath))

			// Prevent directory traversal outside the docs root.
			// Use == ".." or HasPrefix("../") to avoid false-positives on paths
			// like "..images/logo.png" that start with ".." but don't escape the root.
			if resolved == ".." || strings.HasPrefix(resolved, "../") {
				slog.Warn("skipping image reference outside docs directory", //nolint:gosec // ref is a structured log field value, not written to output
					"doc", docRelPath, "ref", ref, "resolved", resolved)

				continue
			}

			// Skip if already collected (deduplication).
			if _, exists := assets[resolved]; exists {
				continue
			}

			absPath := filepath.Join(docsPath, filepath.FromSlash(resolved))

			data, err := os.ReadFile(absPath) //nolint:gosec // path traversal is prevented by the resolved == ".." / HasPrefix("../") check above
			if err != nil {
				slog.Warn("skipping unreadable image reference", //nolint:gosec // ref is a structured log field value, not written to output
					"doc", docRelPath, "ref", ref, "path", absPath, "error", err)

				continue
			}

			assets[resolved] = data
		}
	}

	return assets, nil
}

// SendIngestRequest POSTs the IngestRequest to the Omnidex server's ingest API endpoint.
// It returns the parsed IngestResponse or an error if the request fails or the server returns a non-2xx status.
func (p *Publisher) SendIngestRequest(ctx context.Context, req *core.IngestRequest) (*core.IngestResponse, error) {
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
