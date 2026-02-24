package cmd

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

	files, err := collectFiles(dir, "**/*.md")
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

	files, err := collectFiles(dir, "**/*.md")
	require.NoError(t, err)
	assert.Len(t, files, 3)
	assert.Equal(t, "top", files["top.md"])
	assert.Equal(t, "mid", files["sub/mid.md"])
	assert.Equal(t, "bottom", files["sub/deep/bottom.md"])
}

func TestCollectFiles_NoMatches(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "data.json"), []byte("{}"), 0o600))

	files, err := collectFiles(dir, "**/*.md")
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestCollectFiles_NonExistentDirectory(t *testing.T) {
	files, err := collectFiles("/nonexistent/path/12345", "**/*.md")
	assert.Error(t, err)
	assert.Nil(t, files)
}

func TestCollectFiles_CustomPattern(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(dir, "docs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "root.md"), []byte("root"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "docs", "guide.md"), []byte("guide"), 0o600))

	files, err := collectFiles(dir, "docs/*.md")
	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, "guide", files["docs/guide.md"])
}

func TestBuildIngestRequest(t *testing.T) {
	files := map[string]string{
		"guide.md":      "# Guide",
		"api/readme.md": "# API",
	}

	req := buildIngestRequest("owner/repo", "abc123", files)

	assert.Equal(t, "owner/repo", req.Repo)
	assert.Equal(t, "abc123", req.CommitSHA)
	assert.Len(t, req.Documents, 2)

	// Documents should be sorted by path.
	assert.Equal(t, "api/readme.md", req.Documents[0].Path)
	assert.Equal(t, "# API", req.Documents[0].Content)
	assert.Equal(t, "upsert", req.Documents[0].Action)

	assert.Equal(t, "guide.md", req.Documents[1].Path)
	assert.Equal(t, "# Guide", req.Documents[1].Content)
	assert.Equal(t, "upsert", req.Documents[1].Action)
}

func TestBuildIngestRequest_Empty(t *testing.T) {
	req := buildIngestRequest("owner/repo", "sha", map[string]string{})

	assert.Equal(t, "owner/repo", req.Repo)
	assert.Empty(t, req.Documents)
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
		assert.Len(t, ingestReq.Documents, 1)

		resp := core.IngestResponse{Indexed: 1, Deleted: 0}

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	defer srv.Close()

	req := core.IngestRequest{
		Repo:      "owner/repo",
		CommitSHA: "sha123",
		Documents: []core.IngestDocument{
			{Path: "doc.md", Content: "# Doc", Action: "upsert"},
		},
	}

	resp, err := sendIngestRequest(t.Context(), srv.URL, "test-key", req)
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

	req := core.IngestRequest{Repo: "owner/repo"}

	resp, err := sendIngestRequest(t.Context(), srv.URL, "bad-key", req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "server returned HTTP 401")
}

func TestSendIngestRequest_ServerDown(t *testing.T) {
	req := core.IngestRequest{Repo: "owner/repo"}

	resp, err := sendIngestRequest(t.Context(), "http://localhost:1", "key", req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "HTTP request failed")
}

func TestSendIngestRequest_InvalidURL(t *testing.T) {
	req := core.IngestRequest{Repo: "owner/repo"}

	resp, err := sendIngestRequest(t.Context(), "://invalid", "key", req)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestSendIngestRequest_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	req := core.IngestRequest{Repo: "owner/repo"}

	resp, err := sendIngestRequest(ctx, "http://localhost:8080", "key", req)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestSendIngestRequest_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not-json"))
	}))
	defer srv.Close()

	req := core.IngestRequest{Repo: "owner/repo"}

	resp, err := sendIngestRequest(t.Context(), srv.URL, "key", req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to parse response")
}

func TestRunPublish_EndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer secret", r.Header.Get("Authorization"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var ingestReq core.IngestRequest
		require.NoError(t, json.Unmarshal(body, &ingestReq))
		assert.Equal(t, "owner/repo", ingestReq.Repo)
		assert.Equal(t, "abc123", ingestReq.CommitSHA)
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

	cmdFlags := &cmdFlags{
		LogLevel:   "error",
		TextFormat: true,
		appName:    "test",
		version:    "0.0.0",
	}

	pubFlags := &publishFlags{
		URL:         srv.URL,
		APIKey:      "secret",
		DocsPath:    dir,
		FilePattern: "**/*.md",
		Repo:        "owner/repo",
		CommitSHA:   "abc123",
	}

	err := runPublish(t.Context(), cmdFlags, pubFlags)
	assert.NoError(t, err)
}

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

func TestRunPublish_NoFiles(t *testing.T) {
	dir := t.TempDir()

	cmdFlags := &cmdFlags{LogLevel: "error", TextFormat: true}
	pubFlags := &publishFlags{
		URL:         "http://localhost",
		APIKey:      "key",
		DocsPath:    dir,
		FilePattern: "**/*.md",
		Repo:        "owner/repo",
	}

	err := runPublish(t.Context(), cmdFlags, pubFlags)
	assert.NoError(t, err)
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

func TestRunPublish_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "doc.md"), []byte("# Doc"), 0o600))

	cmdFlags := &cmdFlags{
		LogLevel:   "error",
		TextFormat: true,
		appName:    "test",
		version:    "0.0.0",
	}

	pubFlags := &publishFlags{
		URL:         srv.URL,
		APIKey:      "key",
		DocsPath:    dir,
		FilePattern: "**/*.md",
		Repo:        "owner/repo",
		CommitSHA:   "sha",
	}

	err := runPublish(t.Context(), cmdFlags, pubFlags)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to publish documentation")
}
