package s3store

import (
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ksysoev/omnidex/pkg/core"
)

const (
	testBucket = "test-bucket"
	testRegion = "us-east-1"
)

// newTestStore creates a Store backed by an in-process gofakes3 server.
// It also creates the test bucket so the store is ready to use immediately.
func newTestStore(t *testing.T) *Store {
	t.Helper()

	backend := s3mem.New()

	// Create the bucket directly on the backend before starting the HTTP server.
	require.NoError(t, backend.CreateBucket(testBucket))

	fake := gofakes3.New(backend)
	srv := httptest.NewServer(fake.Server())

	t.Cleanup(srv.Close)

	cfg := Config{
		Bucket:         testBucket,
		Region:         testRegion,
		Endpoint:       srv.URL,
		ForcePathStyle: true,
	}

	store, err := newWithStaticCreds(
		t.Context(),
		cfg,
		"test-access-key",
		"test-secret-key",
	)
	require.NoError(t, err)

	return store
}

func TestNew_ValidConfig(t *testing.T) {
	store := newTestStore(t)
	assert.NotNil(t, store)
}

func TestStore_SaveAndGet(t *testing.T) {
	store := newTestStore(t)

	doc := core.Document{
		ID:          "owner/repo/getting-started.md",
		Repo:        "owner/repo",
		Path:        "getting-started.md",
		Title:       "Getting Started",
		Content:     "# Getting Started\n\nWelcome!",
		CommitSHA:   "abc123",
		UpdatedAt:   time.Now().UTC().Truncate(time.Second),
		ContentType: core.ContentTypeMarkdown,
	}

	err := store.Save(t.Context(), doc)
	require.NoError(t, err)

	got, err := store.Get(t.Context(), "owner/repo", "getting-started.md")
	require.NoError(t, err)

	assert.Equal(t, doc.ID, got.ID)
	assert.Equal(t, doc.Repo, got.Repo)
	assert.Equal(t, doc.Path, got.Path)
	assert.Equal(t, doc.Title, got.Title)
	assert.Equal(t, doc.Content, got.Content)
	assert.Equal(t, doc.CommitSHA, got.CommitSHA)
	assert.Equal(t, doc.ContentType, got.ContentType)
	assert.Equal(t, doc.UpdatedAt, got.UpdatedAt)
}

func TestStore_GetNotFound(t *testing.T) {
	store := newTestStore(t)

	_, err := store.Get(t.Context(), "owner/repo", "nonexistent.md")
	require.Error(t, err)
	assert.True(t, errors.Is(err, core.ErrNotFound))
}

func TestStore_Delete(t *testing.T) {
	store := newTestStore(t)

	doc := core.Document{
		ID:        "owner/repo/to-delete.md",
		Repo:      "owner/repo",
		Path:      "to-delete.md",
		Title:     "Delete Me",
		Content:   "# Delete Me",
		CommitSHA: "abc123",
		UpdatedAt: time.Now().UTC(),
	}

	err := store.Save(t.Context(), doc)
	require.NoError(t, err)

	err = store.Delete(t.Context(), "owner/repo", "to-delete.md")
	require.NoError(t, err)

	_, err = store.Get(t.Context(), "owner/repo", "to-delete.md")
	require.Error(t, err)
	assert.True(t, errors.Is(err, core.ErrNotFound))
}

func TestStore_DeleteNonexistent(t *testing.T) {
	store := newTestStore(t)

	// Deleting a non-existent object must be idempotent (no error).
	err := store.Delete(t.Context(), "owner/repo", "does-not-exist.md")
	assert.NoError(t, err)
}

func TestStore_List(t *testing.T) {
	store := newTestStore(t)

	docs := []core.Document{
		{
			ID:        "owner/repo/getting-started.md",
			Repo:      "owner/repo",
			Path:      "getting-started.md",
			Title:     "Getting Started",
			Content:   "# Getting Started",
			CommitSHA: "abc",
			UpdatedAt: time.Now().UTC().Truncate(time.Second),
		},
		{
			ID:        "owner/repo/api/overview.md",
			Repo:      "owner/repo",
			Path:      "api/overview.md",
			Title:     "API Overview",
			Content:   "# API Overview",
			CommitSHA: "def",
			UpdatedAt: time.Now().UTC().Truncate(time.Second),
		},
	}

	for _, doc := range docs {
		require.NoError(t, store.Save(t.Context(), doc))
	}

	list, err := store.List(t.Context(), "owner/repo")
	require.NoError(t, err)
	assert.Len(t, list, 2)

	// Results should be sorted by path.
	assert.Equal(t, "api/overview.md", list[0].Path)
	assert.Equal(t, "getting-started.md", list[1].Path)
}

func TestStore_ListEmpty(t *testing.T) {
	store := newTestStore(t)

	list, err := store.List(t.Context(), "owner/nonexistent-repo")
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestStore_ListRepos(t *testing.T) {
	store := newTestStore(t)

	doc := core.Document{
		ID:        "owner/repo/README.md",
		Repo:      "owner/repo",
		Path:      "README.md",
		Title:     "README",
		Content:   "# README",
		CommitSHA: "abc",
		UpdatedAt: time.Now().UTC(),
	}

	require.NoError(t, store.Save(t.Context(), doc))

	repos, err := store.ListRepos(t.Context())
	require.NoError(t, err)
	require.Len(t, repos, 1)
	assert.Equal(t, "owner/repo", repos[0].Name)
	assert.Equal(t, 1, repos[0].DocCount)
}

func TestStore_ListReposEmpty(t *testing.T) {
	store := newTestStore(t)

	repos, err := store.ListRepos(t.Context())
	require.NoError(t, err)
	assert.Empty(t, repos)
}

func TestStore_ListMultipleRepos(t *testing.T) {
	store := newTestStore(t)

	for _, repo := range []string{"owner/repo1", "owner/repo2"} {
		doc := core.Document{
			ID:        repo + "/readme.md",
			Repo:      repo,
			Path:      "readme.md",
			Title:     repo,
			Content:   "# " + repo,
			CommitSHA: "abc",
			UpdatedAt: time.Now().UTC(),
		}
		require.NoError(t, store.Save(t.Context(), doc))
	}

	repos, err := store.ListRepos(t.Context())
	require.NoError(t, err)
	assert.Len(t, repos, 2)
	// Sorted alphabetically.
	assert.Equal(t, "owner/repo1", repos[0].Name)
	assert.Equal(t, "owner/repo2", repos[1].Name)
}

func TestStore_SaveOverwritesExisting(t *testing.T) {
	store := newTestStore(t)

	doc := core.Document{
		ID:        "owner/repo/readme.md",
		Repo:      "owner/repo",
		Path:      "readme.md",
		Title:     "Original",
		Content:   "# Original",
		CommitSHA: "abc",
		UpdatedAt: time.Now().UTC().Truncate(time.Second),
	}

	require.NoError(t, store.Save(t.Context(), doc))

	doc.Title = "Updated"
	doc.Content = "# Updated"
	doc.CommitSHA = "def"

	require.NoError(t, store.Save(t.Context(), doc))

	got, err := store.Get(t.Context(), "owner/repo", "readme.md")
	require.NoError(t, err)
	assert.Equal(t, "Updated", got.Title)
	assert.Equal(t, "# Updated", got.Content)
	assert.Equal(t, "def", got.CommitSHA)
}

func TestStore_SaveAndGetAsset(t *testing.T) {
	store := newTestStore(t)

	data := []byte{0x89, 0x50, 0x4E, 0x47} // PNG magic bytes

	err := store.SaveAsset(t.Context(), "owner/repo", "images/arch.png", data)
	require.NoError(t, err)

	got, err := store.GetAsset(t.Context(), "owner/repo", "images/arch.png")
	require.NoError(t, err)
	assert.Equal(t, data, got)
}

func TestStore_GetAssetNotFound(t *testing.T) {
	store := newTestStore(t)

	_, err := store.GetAsset(t.Context(), "owner/repo", "nonexistent.png")
	require.Error(t, err)
	assert.True(t, errors.Is(err, core.ErrNotFound))
}

func TestStore_DeleteAsset(t *testing.T) {
	store := newTestStore(t)

	data := []byte("fake image data")

	err := store.SaveAsset(t.Context(), "owner/repo", "img.png", data)
	require.NoError(t, err)

	err = store.DeleteAsset(t.Context(), "owner/repo", "img.png")
	require.NoError(t, err)

	_, err = store.GetAsset(t.Context(), "owner/repo", "img.png")
	require.Error(t, err)
	assert.True(t, errors.Is(err, core.ErrNotFound))
}

func TestStore_DeleteAssetNonexistent(t *testing.T) {
	store := newTestStore(t)

	// Deleting a non-existent asset must be idempotent (no error).
	err := store.DeleteAsset(t.Context(), "owner/repo", "nope.png")
	assert.NoError(t, err)
}

func TestStore_ListAssets(t *testing.T) {
	store := newTestStore(t)

	// No assets yet — should return an empty result.
	paths, err := store.ListAssets(t.Context(), "owner/repo")
	require.NoError(t, err)
	assert.Empty(t, paths)

	// Save some assets.
	require.NoError(t, store.SaveAsset(t.Context(), "owner/repo", "images/b.png", []byte("b")))
	require.NoError(t, store.SaveAsset(t.Context(), "owner/repo", "images/a.png", []byte("a")))
	require.NoError(t, store.SaveAsset(t.Context(), "owner/repo", "diagrams/arch.svg", []byte("svg")))

	paths, err = store.ListAssets(t.Context(), "owner/repo")
	require.NoError(t, err)

	// Results should be sorted alphabetically.
	assert.Equal(t, []string{
		"diagrams/arch.svg",
		"images/a.png",
		"images/b.png",
	}, paths)
}

func TestStore_SaveAssetOverwrite(t *testing.T) {
	store := newTestStore(t)

	require.NoError(t, store.SaveAsset(t.Context(), "owner/repo", "img.png", []byte("version1")))
	require.NoError(t, store.SaveAsset(t.Context(), "owner/repo", "img.png", []byte("version2")))

	got, err := store.GetAsset(t.Context(), "owner/repo", "img.png")
	require.NoError(t, err)
	assert.Equal(t, []byte("version2"), got)
}

func TestStore_ContentTypeRoundTrip(t *testing.T) {
	store := newTestStore(t)

	doc := core.Document{
		ID:          "owner/repo/api.yaml",
		Repo:        "owner/repo",
		Path:        "api.yaml",
		Title:       "API Spec",
		Content:     "openapi: 3.0.0",
		CommitSHA:   "abc",
		UpdatedAt:   time.Now().UTC().Truncate(time.Second),
		ContentType: core.ContentTypeOpenAPI,
	}

	require.NoError(t, store.Save(t.Context(), doc))

	got, err := store.Get(t.Context(), "owner/repo", "api.yaml")
	require.NoError(t, err)
	assert.Equal(t, core.ContentTypeOpenAPI, got.ContentType)
}

func TestStore_GetDefaultsToMarkdownContentType(t *testing.T) {
	store := newTestStore(t)

	// Save a document with an empty content type; Get should default to markdown.
	doc := core.Document{
		ID:          "owner/repo/doc.md",
		Repo:        "owner/repo",
		Path:        "doc.md",
		Title:       "Doc",
		Content:     "# Doc",
		CommitSHA:   "abc",
		UpdatedAt:   time.Now().UTC().Truncate(time.Second),
		ContentType: "",
	}

	require.NoError(t, store.Save(t.Context(), doc))

	got, err := store.Get(t.Context(), "owner/repo", "doc.md")
	require.NoError(t, err)
	assert.Equal(t, core.ContentTypeMarkdown, got.ContentType)
}

func TestStore_InvalidPathRejectsTraversal(t *testing.T) {
	store := newTestStore(t)

	traversalPaths := []string{
		"../secret",
		"../../etc/passwd",
		"",
		"/absolute/path",
	}

	for _, p := range traversalPaths {
		doc := core.Document{
			Repo:    "owner/repo",
			Path:    p,
			Title:   "Bad",
			Content: "bad",
		}

		err := store.Save(t.Context(), doc)
		require.Error(t, err, "expected error for path %q", p)
		assert.True(t, errors.Is(err, core.ErrInvalidPath), "expected ErrInvalidPath for path %q, got %v", p, err)

		_, err = store.Get(t.Context(), "owner/repo", p)
		require.Error(t, err, "expected error for Get path %q", p)
		assert.True(t, errors.Is(err, core.ErrInvalidPath), "expected ErrInvalidPath for Get path %q, got %v", p, err)
	}
}

func TestStore_InvalidAssetPathRejectsTraversal(t *testing.T) {
	store := newTestStore(t)

	traversalPaths := []string{
		"../secret.png",
		"",
		"/absolute.png",
	}

	for _, p := range traversalPaths {
		err := store.SaveAsset(t.Context(), "owner/repo", p, []byte("data"))
		require.Error(t, err, "expected error for asset path %q", p)
		assert.True(t, errors.Is(err, core.ErrInvalidPath), "expected ErrInvalidPath for asset path %q, got %v", p, err)

		_, err = store.GetAsset(t.Context(), "owner/repo", p)
		require.Error(t, err, "expected error for GetAsset path %q", p)
		assert.True(t, errors.Is(err, core.ErrInvalidPath), "expected ErrInvalidPath for GetAsset path %q, got %v", p, err)
	}
}
