package docstore

import (
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
