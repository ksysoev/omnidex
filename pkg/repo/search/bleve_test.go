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
