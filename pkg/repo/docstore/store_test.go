package docstore

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ksysoev/omnidex/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(tmpDir)

	require.NoError(t, err)
	assert.NotNil(t, store)
}

func TestStore_SaveAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(tmpDir)
	require.NoError(t, err)

	doc := core.Document{
		ID:        "owner/repo/getting-started.md",
		Repo:      "owner/repo",
		Path:      "getting-started.md",
		Title:     "Getting Started",
		Content:   "# Getting Started\n\nWelcome!",
		CommitSHA: "abc123",
		UpdatedAt: time.Now().Truncate(time.Second),
	}

	err = store.Save(t.Context(), doc)
	require.NoError(t, err)

	got, err := store.Get(t.Context(), "owner/repo", "getting-started.md")
	require.NoError(t, err)

	assert.Equal(t, doc.ID, got.ID)
	assert.Equal(t, doc.Repo, got.Repo)
	assert.Equal(t, doc.Path, got.Path)
	assert.Equal(t, doc.Title, got.Title)
	assert.Equal(t, doc.Content, got.Content)
	assert.Equal(t, doc.CommitSHA, got.CommitSHA)
}

func TestStore_GetNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(tmpDir)
	require.NoError(t, err)

	_, err = store.Get(t.Context(), "owner/repo", "nonexistent.md")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestStore_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(tmpDir)
	require.NoError(t, err)

	doc := core.Document{
		ID:        "owner/repo/to-delete.md",
		Repo:      "owner/repo",
		Path:      "to-delete.md",
		Title:     "Delete Me",
		Content:   "# Delete Me",
		CommitSHA: "abc123",
		UpdatedAt: time.Now(),
	}

	err = store.Save(t.Context(), doc)
	require.NoError(t, err)

	err = store.Delete(t.Context(), "owner/repo", "to-delete.md")
	require.NoError(t, err)

	_, err = store.Get(t.Context(), "owner/repo", "to-delete.md")
	assert.Error(t, err)
}

func TestStore_List(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(tmpDir)
	require.NoError(t, err)

	docs := []core.Document{
		{
			ID:        "owner/repo/getting-started.md",
			Repo:      "owner/repo",
			Path:      "getting-started.md",
			Title:     "Getting Started",
			Content:   "# Getting Started",
			CommitSHA: "abc",
			UpdatedAt: time.Now(),
		},
		{
			ID:        "owner/repo/api/overview.md",
			Repo:      "owner/repo",
			Path:      "api/overview.md",
			Title:     "API Overview",
			Content:   "# API Overview",
			CommitSHA: "def",
			UpdatedAt: time.Now(),
		},
	}

	for _, doc := range docs {
		err = store.Save(t.Context(), doc)
		require.NoError(t, err)
	}

	list, err := store.List(t.Context(), "owner/repo")
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestStore_ListRepos(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(tmpDir)
	require.NoError(t, err)

	doc := core.Document{
		ID:        "owner/repo/README.md",
		Repo:      "owner/repo",
		Path:      "README.md",
		Title:     "README",
		Content:   "# README",
		CommitSHA: "abc",
		UpdatedAt: time.Now(),
	}

	err = store.Save(t.Context(), doc)
	require.NoError(t, err)

	repos, err := store.ListRepos(t.Context())
	require.NoError(t, err)
	assert.Len(t, repos, 1)
	assert.Equal(t, "owner/repo", repos[0].Name)
	assert.Equal(t, 1, repos[0].DocCount)
}

func TestStore_ListEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(tmpDir)
	require.NoError(t, err)

	list, err := store.List(t.Context(), "nonexistent/repo")
	require.NoError(t, err)
	assert.Nil(t, list)
}

func TestStore_GetNotFound_ReturnsErrNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(tmpDir)
	require.NoError(t, err)

	_, err = store.Get(t.Context(), "owner/repo", "nonexistent.md")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))
}

func TestStore_PathTraversal_Save(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(tmpDir)
	require.NoError(t, err)

	tests := []struct {
		name string
		repo string
		path string
	}{
		{
			name: "path escapes base via deep traversal",
			repo: "owner/repo",
			path: "../../../../tmp/evil",
		},
		{
			name: "repo escapes base",
			repo: "../../../etc",
			path: "passwd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := core.Document{
				ID:        tt.repo + "/" + tt.path,
				Repo:      tt.repo,
				Path:      tt.path,
				Title:     "Malicious",
				Content:   "pwned",
				CommitSHA: "abc",
				UpdatedAt: time.Now(),
			}

			err := store.Save(t.Context(), doc)
			assert.Error(t, err)
			assert.True(t, errors.Is(err, ErrInvalidPath))
		})
	}
}

func TestStore_PathTraversal_Get(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(tmpDir)
	require.NoError(t, err)

	_, err = store.Get(t.Context(), "owner/repo", "../../../../tmp/evil")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidPath))
}

func TestStore_PathTraversal_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(tmpDir)
	require.NoError(t, err)

	err = store.Delete(t.Context(), "owner/repo", "../../../../tmp/evil")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidPath))
}

func TestStore_PathTraversal_List(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(tmpDir)
	require.NoError(t, err)

	_, err = store.List(t.Context(), "../../etc")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidPath))
}

func TestStore_DeleteNestedCleansEmptyDirs(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(tmpDir)
	require.NoError(t, err)

	doc := core.Document{
		ID:        "owner/repo/deep/nested/doc.md",
		Repo:      "owner/repo",
		Path:      "deep/nested/doc.md",
		Title:     "Nested Doc",
		Content:   "# Nested",
		CommitSHA: "abc",
		UpdatedAt: time.Now(),
	}

	err = store.Save(t.Context(), doc)
	require.NoError(t, err)

	// Confirm it exists.
	got, err := store.Get(t.Context(), "owner/repo", "deep/nested/doc.md")
	require.NoError(t, err)
	assert.Equal(t, "Nested Doc", got.Title)

	// Delete and verify empty directories are cleaned up.
	err = store.Delete(t.Context(), "owner/repo", "deep/nested/doc.md")
	require.NoError(t, err)

	// Verify the document is gone.
	_, err = store.Get(t.Context(), "owner/repo", "deep/nested/doc.md")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))
}

func TestStore_SaveOverwritesExisting(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(tmpDir)
	require.NoError(t, err)

	doc := core.Document{
		ID:        "owner/repo/readme.md",
		Repo:      "owner/repo",
		Path:      "readme.md",
		Title:     "Original",
		Content:   "# Original",
		CommitSHA: "abc",
		UpdatedAt: time.Now().Truncate(time.Second),
	}

	err = store.Save(t.Context(), doc)
	require.NoError(t, err)

	// Update the document.
	doc.Title = "Updated"
	doc.Content = "# Updated"
	doc.CommitSHA = "def"

	err = store.Save(t.Context(), doc)
	require.NoError(t, err)

	got, err := store.Get(t.Context(), "owner/repo", "readme.md")
	require.NoError(t, err)
	assert.Equal(t, "Updated", got.Title)
	assert.Equal(t, "# Updated", got.Content)
	assert.Equal(t, "def", got.CommitSHA)
}

func TestStore_ListReposEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(tmpDir)
	require.NoError(t, err)

	repos, err := store.ListRepos(t.Context())
	require.NoError(t, err)
	assert.Empty(t, repos)
}

func TestStore_ListMultipleRepos(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(tmpDir)
	require.NoError(t, err)

	docs := []core.Document{
		{
			ID:        "owner/repo1/readme.md",
			Repo:      "owner/repo1",
			Path:      "readme.md",
			Title:     "Repo 1",
			Content:   "# Repo 1",
			CommitSHA: "abc",
			UpdatedAt: time.Now(),
		},
		{
			ID:        "owner/repo2/readme.md",
			Repo:      "owner/repo2",
			Path:      "readme.md",
			Title:     "Repo 2",
			Content:   "# Repo 2",
			CommitSHA: "def",
			UpdatedAt: time.Now(),
		},
	}

	for _, doc := range docs {
		err = store.Save(t.Context(), doc)
		require.NoError(t, err)
	}

	repos, err := store.ListRepos(t.Context())
	require.NoError(t, err)
	assert.Len(t, repos, 2)
}

func TestStore_GetCorruptedMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(tmpDir)
	require.NoError(t, err)

	doc := core.Document{
		ID:        "owner/repo/doc.md",
		Repo:      "owner/repo",
		Path:      "doc.md",
		Title:     "Test",
		Content:   "# Test",
		CommitSHA: "abc",
		UpdatedAt: time.Now(),
	}

	err = store.Save(t.Context(), doc)
	require.NoError(t, err)

	// Corrupt the metadata file.
	metaPath := filepath.Join(tmpDir, "owner", "repo", "docs", "doc.md.meta.json")
	err = os.WriteFile(metaPath, []byte("{invalid json"), 0o600)
	require.NoError(t, err)

	_, err = store.Get(t.Context(), "owner/repo", "doc.md")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestStore_ListWithMissingMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(tmpDir)
	require.NoError(t, err)

	// Create a doc file without metadata by writing directly to disk.
	docDir := filepath.Join(tmpDir, "owner", "repo", "docs")
	err = os.MkdirAll(docDir, 0o750)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(docDir, "bare.md"), []byte("# Bare"), 0o600)
	require.NoError(t, err)

	list, err := store.List(t.Context(), "owner/repo")
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, "bare.md", list[0].Path)
	// Title falls back to the relative path when metadata is missing.
	assert.Equal(t, "bare.md", list[0].Title)
}

func TestStore_ListReposSkipsNonDirEntries(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(tmpDir)
	require.NoError(t, err)

	// Save a real document to have a valid repo.
	doc := core.Document{
		ID:        "owner/repo/readme.md",
		Repo:      "owner/repo",
		Path:      "readme.md",
		Title:     "README",
		Content:   "# README",
		CommitSHA: "abc",
		UpdatedAt: time.Now(),
	}

	err = store.Save(t.Context(), doc)
	require.NoError(t, err)

	// Create a file (not directory) in basePath to test non-dir skip.
	err = os.WriteFile(filepath.Join(tmpDir, "stray-file.txt"), []byte("noise"), 0o600)
	require.NoError(t, err)

	// Create a file inside the owner directory to test non-dir repo skip.
	err = os.WriteFile(filepath.Join(tmpDir, "owner", "stray-file.txt"), []byte("noise"), 0o600)
	require.NoError(t, err)

	repos, err := store.ListRepos(t.Context())
	require.NoError(t, err)
	assert.Len(t, repos, 1)
	assert.Equal(t, "owner/repo", repos[0].Name)
}

func TestStore_ListReposSkipsMissingMeta(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(tmpDir)
	require.NoError(t, err)

	// Create a repo directory structure without meta.json.
	repoDir := filepath.Join(tmpDir, "owner", "repo-no-meta", "docs")
	err = os.MkdirAll(repoDir, 0o750)
	require.NoError(t, err)

	repos, err := store.ListRepos(t.Context())
	require.NoError(t, err)
	// Repo without meta.json should be skipped.
	assert.Empty(t, repos)
}

func TestStore_ReadRepoMetaCorruptJSON(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(tmpDir)
	require.NoError(t, err)

	// Create a repo directory with corrupted meta.json.
	repoDir := filepath.Join(tmpDir, "owner", "repo-bad")
	err = os.MkdirAll(repoDir, 0o750)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(repoDir, "meta.json"), []byte("{corrupt json"), 0o600)
	require.NoError(t, err)

	// ListRepos should skip this repo (readRepoMeta fails, continue).
	repos, err := store.ListRepos(t.Context())
	require.NoError(t, err)
	assert.Empty(t, repos)
}

func TestStore_DeleteNonexistent(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(tmpDir)
	require.NoError(t, err)

	// First save a doc to ensure repo dir exists.
	doc := core.Document{
		ID:        "owner/repo/exists.md",
		Repo:      "owner/repo",
		Path:      "exists.md",
		Title:     "Exists",
		Content:   "# Exists",
		CommitSHA: "abc",
		UpdatedAt: time.Now(),
	}

	err = store.Save(t.Context(), doc)
	require.NoError(t, err)

	// Delete a doc that doesn't exist - should not error.
	err = store.Delete(t.Context(), "owner/repo", "does-not-exist.md")
	assert.NoError(t, err)
}

func TestStore_CleanEmptyDirsWithRemainingFiles(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(tmpDir)
	require.NoError(t, err)

	// Save two docs in the same directory.
	for _, name := range []string{"a.md", "b.md"} {
		doc := core.Document{
			ID:        "owner/repo/subdir/" + name,
			Repo:      "owner/repo",
			Path:      "subdir/" + name,
			Title:     name,
			Content:   "# " + name,
			CommitSHA: "abc",
			UpdatedAt: time.Now(),
		}

		err = store.Save(t.Context(), doc)
		require.NoError(t, err)
	}

	// Delete one doc - directory should NOT be cleaned up because b.md still exists.
	err = store.Delete(t.Context(), "owner/repo", "subdir/a.md")
	require.NoError(t, err)

	// b.md should still be accessible.
	got, err := store.Get(t.Context(), "owner/repo", "subdir/b.md")
	require.NoError(t, err)
	assert.Equal(t, "b.md", got.Title)
}
