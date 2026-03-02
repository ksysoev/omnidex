package publisher

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ksysoev/omnidex/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectFiles_MatchesMarkdown(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# Hello"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("plain text"), 0o600))

	files, err := CollectFiles(dir, "**/*.md")
	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, "# Hello", files["readme.md"])
}

func TestCollectFiles_NestedDirectories(t *testing.T) {
	dir := t.TempDir()

	nested := filepath.Join(dir, "sub", "deep")
	require.NoError(t, os.MkdirAll(nested, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "top.md"), []byte("top"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sub", "mid.md"), []byte("mid"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(nested, "bottom.md"), []byte("bottom"), 0o600))

	files, err := CollectFiles(dir, "**/*.md")
	require.NoError(t, err)
	assert.Len(t, files, 3)
	assert.Equal(t, "top", files["top.md"])
	assert.Equal(t, "mid", files["sub/mid.md"])
	assert.Equal(t, "bottom", files["sub/deep/bottom.md"])
}

func TestCollectFiles_NoMatches(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "data.json"), []byte("{}"), 0o600))

	files, err := CollectFiles(dir, "**/*.md")
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestCollectFiles_NonExistentDirectory(t *testing.T) {
	files, err := CollectFiles("/nonexistent/path/12345", "**/*.md")
	assert.Error(t, err)
	assert.Nil(t, files)
}

func TestCollectFiles_FileInsteadOfDirectory(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "notadir.md")
	require.NoError(t, os.WriteFile(filePath, []byte("# Hello"), 0o600))

	files, err := CollectFiles(filePath, "**/*.md")
	assert.Error(t, err)
	assert.Nil(t, files)
	assert.Contains(t, err.Error(), "is not a directory")
}

func TestCollectFiles_CustomPattern(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(dir, "docs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "root.md"), []byte("root"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "docs", "guide.md"), []byte("guide"), 0o600))

	files, err := CollectFiles(dir, "docs/*.md")
	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, "guide", files["docs/guide.md"])
}

func TestBuildIngestRequest(t *testing.T) {
	files := map[string]string{
		"guide.md":      "# Guide",
		"api/readme.md": "# API",
	}

	req := BuildIngestRequest("owner/repo", "abc123", files, nil, true)

	assert.Equal(t, "owner/repo", req.Repo)
	assert.Equal(t, "abc123", req.CommitSHA)
	assert.True(t, req.Sync)
	assert.Len(t, req.Documents, 2)

	// Documents should be sorted by path.
	assert.Equal(t, "api/readme.md", req.Documents[0].Path)
	assert.Equal(t, "# API", req.Documents[0].Content)
	assert.Equal(t, "upsert", req.Documents[0].Action)
	assert.Equal(t, core.ContentTypeMarkdown, req.Documents[0].ContentType)

	assert.Equal(t, "guide.md", req.Documents[1].Path)
	assert.Equal(t, "# Guide", req.Documents[1].Content)
	assert.Equal(t, "upsert", req.Documents[1].Action)
	assert.Equal(t, core.ContentTypeMarkdown, req.Documents[1].ContentType)
}

func TestBuildIngestRequest_SyncFalse(t *testing.T) {
	files := map[string]string{
		"readme.md": "# Hello",
	}

	req := BuildIngestRequest("owner/repo", "sha", files, nil, false)

	assert.Equal(t, "owner/repo", req.Repo)
	assert.False(t, req.Sync)
	assert.Len(t, req.Documents, 1)
}

func TestBuildIngestRequest_Empty(t *testing.T) {
	req := BuildIngestRequest("owner/repo", "sha", map[string]string{}, nil, true)

	assert.Equal(t, "owner/repo", req.Repo)
	assert.True(t, req.Sync)
	assert.Empty(t, req.Documents)
}

func TestBuildIngestRequest_DetectsOpenAPI(t *testing.T) {
	files := map[string]string{
		"api/petstore.yaml": `openapi: "3.0.3"
info:
  title: Petstore API
  version: "1.0.0"
paths: {}`,
		"docs/readme.md": "# Hello",
		"config.yaml":    "name: my-app\nversion: 1.0.0",
	}

	req := BuildIngestRequest("owner/repo", "sha", files, nil, false)

	// config.yaml is not OpenAPI, so it should be skipped entirely.
	assert.Len(t, req.Documents, 2)

	// Sorted: api/petstore.yaml, docs/readme.md (config.yaml filtered out)
	assert.Equal(t, "api/petstore.yaml", req.Documents[0].Path)
	assert.Equal(t, core.ContentTypeOpenAPI, req.Documents[0].ContentType)

	assert.Equal(t, "docs/readme.md", req.Documents[1].Path)
	assert.Equal(t, core.ContentTypeMarkdown, req.Documents[1].ContentType)
}

func TestBuildIngestRequest_DetectsSwagger2(t *testing.T) {
	files := map[string]string{
		"api/legacy.yaml": `swagger: "2.0"
info:
  title: Legacy API
  version: "1.0.0"
basePath: /v1
paths: {}`,
		"docs/readme.md": "# Hello",
	}

	req := BuildIngestRequest("owner/repo", "sha", files, nil, false)

	assert.Len(t, req.Documents, 2)

	assert.Equal(t, "api/legacy.yaml", req.Documents[0].Path)
	assert.Equal(t, core.ContentTypeOpenAPI, req.Documents[0].ContentType)

	assert.Equal(t, "docs/readme.md", req.Documents[1].Path)
	assert.Equal(t, core.ContentTypeMarkdown, req.Documents[1].ContentType)
}

func TestBuildIngestRequest_SkipsNonOpenAPIYAML(t *testing.T) {
	files := map[string]string{
		"config.yaml":        "name: my-app\nversion: 1.0.0",
		"settings.json":      `{"debug": true}`,
		"docs/readme.md":     "# Hello",
		"docker-compose.yml": "version: '3'\nservices: {}",
	}

	req := BuildIngestRequest("owner/repo", "sha", files, nil, false)

	// Only the markdown file should remain; all YAML/JSON without OpenAPI keys are skipped.
	assert.Len(t, req.Documents, 1)
	assert.Equal(t, "docs/readme.md", req.Documents[0].Path)
	assert.Equal(t, core.ContentTypeMarkdown, req.Documents[0].ContentType)
}

func TestSendIngestRequest_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/docs", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var ingestReq core.IngestRequest
		require.NoError(t, json.Unmarshal(body, &ingestReq))
		assert.Equal(t, "owner/repo", ingestReq.Repo)
		assert.Equal(t, "sha123", ingestReq.CommitSHA)
		assert.True(t, ingestReq.Sync)
		assert.Len(t, ingestReq.Documents, 1)

		resp := core.IngestResponse{Indexed: 1, Deleted: 0}

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	defer srv.Close()

	pub := New(srv.URL, "test-key")

	req := core.IngestRequest{
		Repo:      "owner/repo",
		CommitSHA: "sha123",
		Sync:      true,
		Documents: []core.IngestDocument{
			{Path: "doc.md", Content: "# Doc", Action: "upsert"},
		},
	}

	resp, err := pub.SendIngestRequest(t.Context(), &req)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Indexed)
	assert.Equal(t, 0, resp.Deleted)
}
func TestSendIngestRequest_Non2xxStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("unauthorized"))
	}))
	defer srv.Close()

	pub := New(srv.URL, "bad-key")

	req := core.IngestRequest{Repo: "owner/repo"}

	resp, err := pub.SendIngestRequest(t.Context(), &req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "server returned HTTP 401")
}

func TestSendIngestRequest_ServerDown(t *testing.T) {
	pub := New("http://localhost:1", "key")

	req := core.IngestRequest{Repo: "owner/repo"}

	resp, err := pub.SendIngestRequest(t.Context(), &req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "HTTP request failed")
}

func TestSendIngestRequest_InvalidURL(t *testing.T) {
	pub := New("://invalid", "key")

	req := core.IngestRequest{Repo: "owner/repo"}

	resp, err := pub.SendIngestRequest(t.Context(), &req)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestSendIngestRequest_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	pub := New("http://localhost:8080", "key")

	req := core.IngestRequest{Repo: "owner/repo"}

	resp, err := pub.SendIngestRequest(ctx, &req)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestSendIngestRequest_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not-json"))
	}))
	defer srv.Close()

	pub := New(srv.URL, "key")

	req := core.IngestRequest{Repo: "owner/repo"}

	resp, err := pub.SendIngestRequest(t.Context(), &req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to parse response")
}

func TestPublish_EndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer secret", r.Header.Get("Authorization"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var ingestReq core.IngestRequest
		require.NoError(t, json.Unmarshal(body, &ingestReq))
		assert.Equal(t, "owner/repo", ingestReq.Repo)
		assert.Equal(t, "abc123", ingestReq.CommitSHA)
		assert.True(t, ingestReq.Sync)
		assert.Len(t, ingestReq.Documents, 2)

		resp := core.IngestResponse{Indexed: 2, Deleted: 0}

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	defer srv.Close()

	dir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(dir, "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "doc.md"), []byte("# Doc"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sub", "nested.md"), []byte("# Nested"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ignore.txt"), []byte("skip"), 0o600))

	pub := New(srv.URL, "secret")

	resp, err := pub.Publish(t.Context(), dir, "**/*.md", "owner/repo", "abc123", true)
	require.NoError(t, err)
	assert.Equal(t, 2, resp.Indexed)
	assert.Equal(t, 0, resp.Deleted)
}

func TestPublish_NoFiles(t *testing.T) {
	dir := t.TempDir()

	pub := New("http://localhost", "key")

	resp, err := pub.Publish(t.Context(), dir, "**/*.md", "owner/repo", "", true)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Indexed)
	assert.Equal(t, 0, resp.Deleted)
}

func TestPublish_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "doc.md"), []byte("# Doc"), 0o600))

	pub := New(srv.URL, "key")

	resp, err := pub.Publish(t.Context(), dir, "**/*.md", "owner/repo", "sha", true)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to publish documentation")
}

func TestExtractImageRefs(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "relative image",
			content: "![diagram](./images/arch.png)",
			want:    []string{"./images/arch.png"},
		},
		{
			name:    "multiple images",
			content: "![a](a.png)\n\n![b](sub/b.jpg)",
			want:    []string{"a.png", "sub/b.jpg"},
		},
		{
			name:    "skips absolute http URL",
			content: "![remote](http://example.com/img.png)",
			want:    nil,
		},
		{
			name:    "skips absolute https URL",
			content: "![remote](https://example.com/img.png)",
			want:    nil,
		},
		{
			name:    "skips protocol-relative URL",
			content: "![cdn](//cdn.example.com/img.png)",
			want:    nil,
		},
		{
			name:    "skips data URI",
			content: "![inline](data:image/png;base64,ABC)",
			want:    nil,
		},
		{
			name:    "skips absolute path",
			content: "![static](/static/img.png)",
			want:    nil,
		},
		{
			name:    "no images",
			content: "# Hello\n\nJust text.",
			want:    nil,
		},
		{
			name:    "mixed relative and absolute",
			content: "![local](diagram.png)\n\n![remote](https://example.com/img.png)\n\n![other](../shared/logo.svg)",
			want:    []string{"diagram.png", "../shared/logo.svg"},
		},
		{
			name:    "empty destination skipped",
			content: "![empty]()",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractImageRefs(tt.content)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCollectAssets(t *testing.T) {
	dir := t.TempDir()

	// Create image files on disk.
	imgDir := filepath.Join(dir, "images")
	require.NoError(t, os.MkdirAll(imgDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(imgDir, "arch.png"), []byte("png-data"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(imgDir, "logo.svg"), []byte("svg-data"), 0o600))

	files := map[string]string{
		"docs/guide.md":  "![arch](../images/arch.png)\n\n![logo](../images/logo.svg)",
		"docs/readme.md": "![arch](../images/arch.png)", // duplicate reference
		"config.yaml":    "name: test",                  // non-markdown, should be skipped
		"api/petstore.yaml": `openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths: {}`,
	}

	assets, err := CollectAssets(dir, files)
	require.NoError(t, err)

	// Should have 2 unique assets (deduplication).
	assert.Len(t, assets, 2)
	assert.Equal(t, []byte("png-data"), assets["images/arch.png"])
	assert.Equal(t, []byte("svg-data"), assets["images/logo.svg"])
}

func TestCollectAssets_SkipsTraversalOutsideRoot(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"guide.md": "![escaped](../../etc/passwd)",
	}

	assets, err := CollectAssets(dir, files)
	require.NoError(t, err)
	assert.Empty(t, assets)
}

func TestCollectAssets_SkipsMissingFiles(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"guide.md": "![missing](nonexistent.png)",
	}

	assets, err := CollectAssets(dir, files)
	require.NoError(t, err)
	assert.Empty(t, assets)
}

func TestCollectAssets_NoMarkdownFiles(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"config.yaml": "name: test",
	}

	assets, err := CollectAssets(dir, files)
	require.NoError(t, err)
	assert.Empty(t, assets)
}

func TestBuildIngestRequest_WithAssets(t *testing.T) {
	files := map[string]string{
		"docs/readme.md": "# Hello\n\n![diagram](images/arch.png)",
	}

	assets := map[string][]byte{
		"images/arch.png": {0x89, 0x50, 0x4E, 0x47},
	}

	req := BuildIngestRequest("owner/repo", "sha", files, assets, true)

	assert.Equal(t, "owner/repo", req.Repo)
	assert.Equal(t, "sha", req.CommitSHA)
	assert.True(t, req.Sync)
	assert.Len(t, req.Documents, 1)
	assert.Len(t, req.Assets, 1)

	assert.Equal(t, "images/arch.png", req.Assets[0].Path)
	assert.Equal(t, "upsert", req.Assets[0].Action)
	assert.NotEmpty(t, req.Assets[0].Content) // base64 encoded
}

func TestBuildIngestRequest_NilAssets(t *testing.T) {
	files := map[string]string{
		"docs/readme.md": "# Hello",
	}

	req := BuildIngestRequest("owner/repo", "sha", files, nil, false)

	assert.Len(t, req.Documents, 1)
	assert.Nil(t, req.Assets)
}

func TestBuildIngestRequest_AssetsSorted(t *testing.T) {
	files := map[string]string{
		"docs/readme.md": "# Hello",
	}

	assets := map[string][]byte{
		"images/c.png": []byte("c"),
		"images/a.png": []byte("a"),
		"images/b.png": []byte("b"),
	}

	req := BuildIngestRequest("owner/repo", "sha", files, assets, false)

	require.Len(t, req.Assets, 3)
	assert.Equal(t, "images/a.png", req.Assets[0].Path)
	assert.Equal(t, "images/b.png", req.Assets[1].Path)
	assert.Equal(t, "images/c.png", req.Assets[2].Path)
}
