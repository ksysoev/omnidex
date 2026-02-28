package search

import (
	"fmt"
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

func TestBleveEngine_SearchDefaultLimit(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "test.bleve")

	engine, err := NewBleve(indexPath)
	require.NoError(t, err)

	defer engine.Close()

	doc := core.Document{
		ID:        "owner/repo/default-limit.md",
		Repo:      "owner/repo",
		Path:      "default-limit.md",
		Title:     "Default Limit Doc",
		UpdatedAt: time.Now(),
	}

	err = engine.Index(t.Context(), doc, "Default limit content for testing")
	require.NoError(t, err)

	// Search with Limit=0 to trigger the default limit branch (opts.Limit <= 0).
	results, err := engine.Search(t.Context(), "default limit", core.SearchOpts{Limit: 0})
	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Greater(t, results.Total, uint64(0))
}

func TestBleveEngine_SearchFieldExtraction(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "test.bleve")

	engine, err := NewBleve(indexPath)
	require.NoError(t, err)

	defer engine.Close()

	doc := core.Document{
		ID:        "myowner/myrepo/mypath.md",
		Repo:      "myowner/myrepo",
		Path:      "mypath.md",
		Title:     "Field Extraction Test",
		UpdatedAt: time.Now(),
	}

	err = engine.Index(t.Context(), doc, "Field extraction test content")
	require.NoError(t, err)

	results, err := engine.Search(t.Context(), "field extraction", core.SearchOpts{Limit: 10})
	require.NoError(t, err)
	require.NotEmpty(t, results.Hits)

	hit := results.Hits[0]
	assert.Equal(t, "myowner/myrepo/mypath.md", hit.ID)
	assert.Equal(t, "myowner/myrepo", hit.Repo)
	assert.Equal(t, "mypath.md", hit.Path)
	assert.Equal(t, "Field Extraction Test", hit.Title)
}

func TestBleveEngine_CloseExplicit(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "test.bleve")

	engine, err := NewBleve(indexPath)
	require.NoError(t, err)

	err = engine.Close()
	require.NoError(t, err)

	// Verify we can reopen after explicit close.
	engine2, err := NewBleve(indexPath)
	require.NoError(t, err)
	assert.NotNil(t, engine2)

	engine2.Close()
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

func TestSplitQueryTerms(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []queryTerm
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "whitespace only",
			input:    "   \t  ",
			expected: nil,
		},
		{
			name:  "single word",
			input: "markdown",
			expected: []queryTerm{
				{text: "markdown", phrase: false},
			},
		},
		{
			name:  "multiple words",
			input: "getting started guide",
			expected: []queryTerm{
				{text: "getting", phrase: false},
				{text: "started", phrase: false},
				{text: "guide", phrase: false},
			},
		},
		{
			name:  "quoted phrase",
			input: `"getting started"`,
			expected: []queryTerm{
				{text: "getting started", phrase: true},
			},
		},
		{
			name:  "mixed terms and phrases",
			input: `welcome "getting started" guide`,
			expected: []queryTerm{
				{text: "welcome", phrase: false},
				{text: "getting started", phrase: true},
				{text: "guide", phrase: false},
			},
		},
		{
			name:  "unclosed quote",
			input: `"getting started`,
			expected: []queryTerm{
				{text: "getting started", phrase: true},
			},
		},
		{
			name:     "empty quotes",
			input:    `""`,
			expected: nil,
		},
		{
			name:  "extra whitespace between words",
			input: "  hello   world  ",
			expected: []queryTerm{
				{text: "hello", phrase: false},
				{text: "world", phrase: false},
			},
		},
		{
			name:  "multiple quoted phrases",
			input: `"hello world" "foo bar"`,
			expected: []queryTerm{
				{text: "hello world", phrase: true},
				{text: "foo bar", phrase: true},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := splitQueryTerms(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestBleveEngine_SearchPartialWord(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "test.bleve")

	engine, err := NewBleve(indexPath)
	require.NoError(t, err)

	defer engine.Close()

	doc := core.Document{
		ID:        "owner/repo/markdown-guide.md",
		Repo:      "owner/repo",
		Path:      "markdown-guide.md",
		Title:     "Markdown Guide",
		UpdatedAt: time.Now(),
	}

	err = engine.Index(t.Context(), doc, "This is a comprehensive markdown formatting guide")
	require.NoError(t, err)

	// Searching for "mark" should match "markdown" via prefix query.
	results, err := engine.Search(t.Context(), "mark", core.SearchOpts{Limit: 10})
	require.NoError(t, err)
	assert.Greater(t, results.Total, uint64(0), "partial word 'mark' should match 'markdown'")
	assert.Equal(t, "owner/repo/markdown-guide.md", results.Hits[0].ID)
}

func TestBleveEngine_SearchPartialWordGet(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "test.bleve")

	engine, err := NewBleve(indexPath)
	require.NoError(t, err)

	defer engine.Close()

	doc := core.Document{
		ID:        "owner/repo/getting-started.md",
		Repo:      "owner/repo",
		Path:      "getting-started.md",
		Title:     "Getting Started",
		UpdatedAt: time.Now(),
	}

	err = engine.Index(t.Context(), doc, "Getting started with the project setup and configuration")
	require.NoError(t, err)

	// Searching for "get" should match "getting" via prefix query.
	results, err := engine.Search(t.Context(), "get", core.SearchOpts{Limit: 10})
	require.NoError(t, err)
	assert.Greater(t, results.Total, uint64(0), "partial word 'get' should match 'getting'")
	assert.Equal(t, "owner/repo/getting-started.md", results.Hits[0].ID)
}

func TestBleveEngine_SearchFuzzyTypo(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "test.bleve")

	engine, err := NewBleve(indexPath)
	require.NoError(t, err)

	defer engine.Close()

	doc := core.Document{
		ID:        "owner/repo/markdown-guide.md",
		Repo:      "owner/repo",
		Path:      "markdown-guide.md",
		Title:     "Markdown Guide",
		UpdatedAt: time.Now(),
	}

	err = engine.Index(t.Context(), doc, "This is a comprehensive markdown formatting guide")
	require.NoError(t, err)

	// Searching for "markdwon" (typo) should match "markdown" via fuzzy query.
	results, err := engine.Search(t.Context(), "markdwon", core.SearchOpts{Limit: 10})
	require.NoError(t, err)
	assert.Greater(t, results.Total, uint64(0), "typo 'markdwon' should match 'markdown'")
	assert.Equal(t, "owner/repo/markdown-guide.md", results.Hits[0].ID)
}

func TestBleveEngine_SearchQuotedPhrase(t *testing.T) {
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
		UpdatedAt: time.Now(),
	}

	err = engine.Index(t.Context(), doc, "Getting started with the project setup and configuration")
	require.NoError(t, err)

	// Quoted phrase search should match exact phrase.
	results, err := engine.Search(t.Context(), `"getting started"`, core.SearchOpts{Limit: 10})
	require.NoError(t, err)
	assert.Greater(t, results.Total, uint64(0), "quoted phrase 'getting started' should match")
	assert.Equal(t, "owner/repo/getting-started.md", results.Hits[0].ID)
}

func TestBleveEngine_SearchMultipleTerms(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "test.bleve")

	engine, err := NewBleve(indexPath)
	require.NoError(t, err)

	defer engine.Close()

	matchDoc := core.Document{
		ID:        "owner/repo/markdown-guide.md",
		Repo:      "owner/repo",
		Path:      "markdown-guide.md",
		Title:     "Markdown Formatting Guide",
		UpdatedAt: time.Now(),
	}

	noMatchDoc := core.Document{
		ID:        "owner/repo/intro.md",
		Repo:      "owner/repo",
		Path:      "intro.md",
		Title:     "Introduction",
		UpdatedAt: time.Now(),
	}

	err = engine.Index(t.Context(), matchDoc, "Learn markdown formatting for your documents")
	require.NoError(t, err)

	err = engine.Index(t.Context(), noMatchDoc, "Welcome to the project introduction")
	require.NoError(t, err)

	// Both terms must match -- only the markdown guide has both "markdown" and "formatting".
	results, err := engine.Search(t.Context(), "markdown formatting", core.SearchOpts{Limit: 10})
	require.NoError(t, err)
	require.Greater(t, results.Total, uint64(0))
	assert.Equal(t, "owner/repo/markdown-guide.md", results.Hits[0].ID)

	// "markdown introduction" -- no single document contains both terms.
	results, err = engine.Search(t.Context(), "markdown introduction", core.SearchOpts{Limit: 10})
	require.NoError(t, err)
	// Each document only matches one term, so the conjunction should not match either.
	assert.Equal(t, uint64(0), results.Total, "conjunction of unrelated terms should not match a single document")
}

func TestBleveEngine_SearchBoostRanking(t *testing.T) {
	tests := []struct {
		name        string
		doc1        core.Document
		doc1Content string
		doc2        core.Document
		doc2Content string
		query       string
		expectedID  string
		reason      string
	}{
		{
			name: "exact match ranks higher than prefix match",
			doc1: core.Document{
				ID:        "owner/repo/exact.md",
				Repo:      "owner/repo",
				Path:      "exact.md",
				Title:     "Markdown Reference",
				UpdatedAt: time.Now(),
			},
			doc1Content: "Guide to markdown syntax and features",
			doc2: core.Document{
				ID:        "owner/repo/prefix.md",
				Repo:      "owner/repo",
				Path:      "prefix.md",
				Title:     "Markdownlint Setup",
				UpdatedAt: time.Now(),
			},
			doc2Content: "Guide to markdownlint configuration",
			query:       "markdown",
			expectedID:  "owner/repo/exact.md",
			reason:      "exact match should score higher than prefix-only match",
		},
		{
			name: "title match ranks higher than content match",
			doc1: core.Document{
				ID:        "owner/repo/title.md",
				Repo:      "owner/repo",
				Path:      "title.md",
				Title:     "Markdown Reference",
				UpdatedAt: time.Now(),
			},
			doc1Content: "A general reference document",
			doc2: core.Document{
				ID:        "owner/repo/content.md",
				Repo:      "owner/repo",
				Path:      "content.md",
				Title:     "Reference Guide",
				UpdatedAt: time.Now(),
			},
			doc2Content: "This explains markdown syntax in detail",
			query:       "markdown",
			expectedID:  "owner/repo/title.md",
			reason:      "title match should score higher than content-only match",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			indexPath := filepath.Join(tmpDir, "test.bleve")

			engine, err := NewBleve(indexPath)
			require.NoError(t, err)

			defer engine.Close()

			err = engine.Index(t.Context(), tc.doc1, tc.doc1Content)
			require.NoError(t, err)

			err = engine.Index(t.Context(), tc.doc2, tc.doc2Content)
			require.NoError(t, err)

			results, err := engine.Search(t.Context(), tc.query, core.SearchOpts{Limit: 10})
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(results.Hits), 2, "both documents should match")

			assert.Equal(t, tc.expectedID, results.Hits[0].ID, tc.reason)
		})
	}
}

func TestBleveEngine_SearchEmptyQuery(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "test.bleve")

	engine, err := NewBleve(indexPath)
	require.NoError(t, err)

	defer engine.Close()

	doc := core.Document{
		ID:        "owner/repo/doc.md",
		Repo:      "owner/repo",
		Path:      "doc.md",
		Title:     "Test Doc",
		UpdatedAt: time.Now(),
	}

	err = engine.Index(t.Context(), doc, "Some content here")
	require.NoError(t, err)

	// Empty query should return no results (MatchNoneQuery).
	results, err := engine.Search(t.Context(), "", core.SearchOpts{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, uint64(0), results.Total)

	// Whitespace-only query should also return no results.
	results, err = engine.Search(t.Context(), "   ", core.SearchOpts{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, uint64(0), results.Total)
}

func TestBleveEngine_SearchHighlightingWorks(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "test.bleve")

	engine, err := NewBleve(indexPath)
	require.NoError(t, err)

	defer engine.Close()

	doc := core.Document{
		ID:        "owner/repo/highlighted.md",
		Repo:      "owner/repo",
		Path:      "highlighted.md",
		Title:     "Highlighted Document",
		UpdatedAt: time.Now(),
	}

	err = engine.Index(t.Context(), doc, "This document contains markdown formatting examples")
	require.NoError(t, err)

	results, err := engine.Search(t.Context(), "markdown", core.SearchOpts{Limit: 10})
	require.NoError(t, err)
	require.NotEmpty(t, results.Hits)
	assert.NotEmpty(t, results.Hits[0].Fragments, "search results should include highlight fragments")
}

func TestBleveEngine_ListByRepo(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "test.bleve")

	engine, err := NewBleve(indexPath)
	require.NoError(t, err)

	defer engine.Close()

	// Index documents across two different repos.
	docs := []struct {
		doc     core.Document
		content string
	}{
		{
			doc: core.Document{
				ID:        "owner/repo-a/doc1.md",
				Repo:      "owner/repo-a",
				Path:      "doc1.md",
				Title:     "Doc 1",
				UpdatedAt: time.Now(),
			},
			content: "First document",
		},
		{
			doc: core.Document{
				ID:        "owner/repo-a/doc2.md",
				Repo:      "owner/repo-a",
				Path:      "doc2.md",
				Title:     "Doc 2",
				UpdatedAt: time.Now(),
			},
			content: "Second document",
		},
		{
			doc: core.Document{
				ID:        "owner/repo-b/other.md",
				Repo:      "owner/repo-b",
				Path:      "other.md",
				Title:     "Other",
				UpdatedAt: time.Now(),
			},
			content: "Other repo document",
		},
	}

	for _, d := range docs {
		err = engine.Index(t.Context(), d.doc, d.content)
		require.NoError(t, err)
	}

	// List repo-a — should return exactly 2 doc IDs.
	ids, err := engine.ListByRepo(t.Context(), "owner/repo-a")
	require.NoError(t, err)
	assert.Len(t, ids, 2)
	assert.ElementsMatch(t, []string{"owner/repo-a/doc1.md", "owner/repo-a/doc2.md"}, ids)

	// List repo-b — should return exactly 1 doc ID.
	ids, err = engine.ListByRepo(t.Context(), "owner/repo-b")
	require.NoError(t, err)
	assert.Len(t, ids, 1)
	assert.Equal(t, "owner/repo-b/other.md", ids[0])
}

func TestBleveEngine_ListByRepoEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "test.bleve")

	engine, err := NewBleve(indexPath)
	require.NoError(t, err)

	defer engine.Close()

	// No documents indexed — ListByRepo should return an empty slice.
	ids, err := engine.ListByRepo(t.Context(), "owner/nonexistent")
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestBleveEngine_ListByRepoManyDocuments(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "test.bleve")

	engine, err := NewBleve(indexPath)
	require.NoError(t, err)

	defer engine.Close()

	// Index more documents than a typical single-page fetch to exercise the
	// collection logic in ListByRepo. While we can't easily exceed the
	// listByRepoPageSize constant (10k) in a unit test, we validate that all
	// indexed documents are returned faithfully.
	const docCount = 50

	expected := make([]string, 0, docCount)

	for i := range docCount {
		doc := core.Document{
			ID:        fmt.Sprintf("owner/big-repo/doc-%03d.md", i),
			Repo:      "owner/big-repo",
			Path:      fmt.Sprintf("doc-%03d.md", i),
			Title:     fmt.Sprintf("Doc %d", i),
			UpdatedAt: time.Now(),
		}

		err = engine.Index(t.Context(), doc, fmt.Sprintf("Content of document %d", i))
		require.NoError(t, err)

		expected = append(expected, doc.ID)
	}

	ids, err := engine.ListByRepo(t.Context(), "owner/big-repo")
	require.NoError(t, err)
	assert.Len(t, ids, docCount)
	assert.ElementsMatch(t, expected, ids)
}

func TestBleveEngine_ListByRepoAfterRemove(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "test.bleve")

	engine, err := NewBleve(indexPath)
	require.NoError(t, err)

	defer engine.Close()

	doc := core.Document{
		ID:        "owner/repo/removable.md",
		Repo:      "owner/repo",
		Path:      "removable.md",
		Title:     "Removable",
		UpdatedAt: time.Now(),
	}

	err = engine.Index(t.Context(), doc, "content")
	require.NoError(t, err)

	ids, err := engine.ListByRepo(t.Context(), "owner/repo")
	require.NoError(t, err)
	assert.Len(t, ids, 1)

	// Remove the document and verify it no longer appears.
	err = engine.Remove(t.Context(), "owner/repo/removable.md")
	require.NoError(t, err)

	ids, err = engine.ListByRepo(t.Context(), "owner/repo")
	require.NoError(t, err)
	assert.Empty(t, ids)
}
