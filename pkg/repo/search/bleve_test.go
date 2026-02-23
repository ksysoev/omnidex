package search

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ksysoev/omnidex/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBleve(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "test.bleve")

	engine, err := NewBleve(indexPath)
	require.NoError(t, err)
	assert.NotNil(t, engine)

	engine.Close()
}

func TestBleveEngine_IndexAndSearch(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "test.bleve")

	engine, err := NewBleve(indexPath)
	require.NoError(t, err)

	defer engine.Close()

	doc := core.Document{
		ID:        "owner/repo/getting-started.md",
		Repo:      "owner/repo",
		Path:      "getting-started.md",
		Title:     "Getting Started Guide",
		Content:   "# Getting Started\n\nWelcome to the project.",
		CommitSHA: "abc123",
		UpdatedAt: time.Now(),
	}

	err = engine.Index(t.Context(), doc, "Getting Started Guide Welcome to the project")
	require.NoError(t, err)

	// Search for the document.
	results, err := engine.Search(t.Context(), "getting started", core.SearchOpts{Limit: 10})
	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Greater(t, results.Total, uint64(0))
	assert.NotEmpty(t, results.Hits)
	assert.Equal(t, "owner/repo/getting-started.md", results.Hits[0].ID)
}

func TestBleveEngine_Remove(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "test.bleve")

	engine, err := NewBleve(indexPath)
	require.NoError(t, err)

	defer engine.Close()

	doc := core.Document{
		ID:        "owner/repo/to-remove.md",
		Repo:      "owner/repo",
		Path:      "to-remove.md",
		Title:     "To Remove",
		Content:   "# To Remove",
		CommitSHA: "abc",
		UpdatedAt: time.Now(),
	}

	err = engine.Index(t.Context(), doc, "To Remove content")
	require.NoError(t, err)

	err = engine.Remove(t.Context(), "owner/repo/to-remove.md")
	require.NoError(t, err)

	results, err := engine.Search(t.Context(), "remove", core.SearchOpts{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, uint64(0), results.Total)
}

func TestBleveEngine_DocCount(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "test.bleve")

	engine, err := NewBleve(indexPath)
	require.NoError(t, err)

	defer engine.Close()

	count, err := engine.DocCount()
	require.NoError(t, err)
	assert.Equal(t, uint64(0), count)

	doc := core.Document{
		ID:        "owner/repo/doc.md",
		Repo:      "owner/repo",
		Path:      "doc.md",
		Title:     "Test Doc",
		UpdatedAt: time.Now(),
	}

	err = engine.Index(t.Context(), doc, "Test document content")
	require.NoError(t, err)

	count, err = engine.DocCount()
	require.NoError(t, err)
	assert.Equal(t, uint64(1), count)
}

func TestBleveEngine_SearchEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "test.bleve")

	engine, err := NewBleve(indexPath)
	require.NoError(t, err)

	defer engine.Close()

	results, err := engine.Search(t.Context(), "nonexistent", core.SearchOpts{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, uint64(0), results.Total)
	assert.Empty(t, results.Hits)
}

func TestBleveEngine_ReopenIndex(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "test.bleve")

	// Create and populate.
	engine, err := NewBleve(indexPath)
	require.NoError(t, err)

	doc := core.Document{
		ID:        "owner/repo/persistent.md",
		Repo:      "owner/repo",
		Path:      "persistent.md",
		Title:     "Persistent Doc",
		UpdatedAt: time.Now(),
	}

	err = engine.Index(t.Context(), doc, "Persistent document content")
	require.NoError(t, err)

	err = engine.Close()
	require.NoError(t, err)

	// Reopen and verify.
	engine2, err := NewBleve(indexPath)
	require.NoError(t, err)

	defer engine2.Close()

	count, err := engine2.DocCount()
	require.NoError(t, err)
	assert.Equal(t, uint64(1), count)
}
