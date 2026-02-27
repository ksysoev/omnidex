package views

import (
	"bytes"
	"testing"
	"time"

	"github.com/ksysoev/omnidex/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	require.NotPanics(t, func() {
		r := New()
		assert.NotNil(t, r)
	})
}

func TestRenderHome_FullPage(t *testing.T) {
	r := New()

	repos := []core.RepoInfo{
		{Name: "my-org/repo-alpha", DocCount: 5, LastUpdated: time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)},
		{Name: "my-org/repo-beta", DocCount: 12, LastUpdated: time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)},
	}

	var buf bytes.Buffer

	err := r.RenderHome(&buf, repos, false)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "<!DOCTYPE html>")
	assert.Contains(t, output, "<nav")
	assert.Contains(t, output, "my-org/repo-alpha")
	assert.Contains(t, output, "my-org/repo-beta")
	assert.Contains(t, output, "5 documents")
	assert.Contains(t, output, "12 documents")
}

func TestRenderHome_Partial(t *testing.T) {
	r := New()

	repos := []core.RepoInfo{
		{Name: "my-org/repo-alpha", DocCount: 3, LastUpdated: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
	}

	var buf bytes.Buffer

	err := r.RenderHome(&buf, repos, true)
	require.NoError(t, err)

	output := buf.String()
	assert.NotContains(t, output, "<!DOCTYPE html>")
	assert.Contains(t, output, "my-org/repo-alpha")
	assert.Contains(t, output, "3 documents")
}

func TestRenderHome_EmptyRepos(t *testing.T) {
	r := New()

	var buf bytes.Buffer

	err := r.RenderHome(&buf, nil, false)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "No repositories indexed yet.")
}

func TestRenderRepoIndex_FullPage(t *testing.T) {
	r := New()

	docs := []core.DocumentMeta{
		{ID: "my-org/repo/getting-started.md", Repo: "my-org/repo", Path: "getting-started.md", Title: "Getting Started", UpdatedAt: time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)},
		{ID: "my-org/repo/advanced.md", Repo: "my-org/repo", Path: "advanced.md", Title: "Advanced Usage", UpdatedAt: time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)},
	}

	var buf bytes.Buffer

	err := r.RenderRepoIndex(&buf, "my-org/repo", docs, false)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "<!DOCTYPE html>")
	assert.Contains(t, output, "<nav")
	assert.Contains(t, output, "my-org/repo")
	assert.Contains(t, output, "Getting Started")
	assert.Contains(t, output, "Advanced Usage")
	assert.Contains(t, output, "getting-started.md")
	assert.Contains(t, output, "advanced.md")
}

func TestRenderRepoIndex_Partial(t *testing.T) {
	r := New()

	docs := []core.DocumentMeta{
		{ID: "my-org/repo/readme.md", Repo: "my-org/repo", Path: "readme.md", Title: "README", UpdatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
	}

	var buf bytes.Buffer

	err := r.RenderRepoIndex(&buf, "my-org/repo", docs, true)
	require.NoError(t, err)

	output := buf.String()
	assert.NotContains(t, output, "<!DOCTYPE html>")
	assert.Contains(t, output, "my-org/repo")
	assert.Contains(t, output, "README")
}

func TestRenderRepoIndex_EmptyDocs(t *testing.T) {
	r := New()

	var buf bytes.Buffer

	err := r.RenderRepoIndex(&buf, "my-org/repo", nil, false)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "No documents in this repository yet.")
}

func TestRenderDoc_FullPage(t *testing.T) {
	r := New()

	doc := core.Document{
		ID:        "my-org/repo/getting-started.md",
		Repo:      "my-org/repo",
		Path:      "getting-started.md",
		Title:     "Getting Started",
		Content:   "# Getting Started\nWelcome!",
		CommitSHA: "abc123",
		UpdatedAt: time.Date(2025, 5, 10, 0, 0, 0, 0, time.UTC),
	}

	htmlContent := []byte("<h1>Getting Started</h1><p>Welcome!</p>")

	navDocs := []core.DocumentMeta{
		{ID: "my-org/repo/getting-started.md", Repo: "my-org/repo", Path: "getting-started.md", Title: "Getting Started"},
		{ID: "my-org/repo/advanced.md", Repo: "my-org/repo", Path: "advanced.md", Title: "Advanced Usage"},
	}

	headings := []core.Heading{
		{Level: 1, ID: "getting-started", Text: "Getting Started"},
		{Level: 2, ID: "installation", Text: "Installation"},
	}

	var buf bytes.Buffer

	err := r.RenderDoc(&buf, doc, htmlContent, headings, navDocs, false)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "<!DOCTYPE html>")
	assert.Contains(t, output, "<nav")
	assert.Contains(t, output, "<h1>Getting Started</h1><p>Welcome!</p>")
	assert.Contains(t, output, "Advanced Usage")
	assert.Contains(t, output, "getting-started.md")
	assert.Contains(t, output, "On this page")
	assert.Contains(t, output, "Installation")
	assert.Contains(t, output, "data-toc-link")
}

func TestRenderDoc_Partial(t *testing.T) {
	r := New()

	doc := core.Document{
		ID:   "my-org/repo/readme.md",
		Repo: "my-org/repo",
		Path: "readme.md",
	}

	htmlContent := []byte("<p>Partial doc content</p>")

	var buf bytes.Buffer

	err := r.RenderDoc(&buf, doc, htmlContent, nil, nil, true)
	require.NoError(t, err)

	output := buf.String()
	assert.NotContains(t, output, "<!DOCTYPE html>")
	assert.Contains(t, output, "<p>Partial doc content</p>")
}

func TestRenderDoc_TOCHiddenWithFewHeadings(t *testing.T) {
	r := New()

	doc := core.Document{
		ID:   "my-org/repo/guide.md",
		Repo: "my-org/repo",
		Path: "guide.md",
	}

	htmlContent := []byte("<h1>Guide</h1><p>Content</p>")

	tests := []struct {
		name     string
		headings []core.Heading
	}{
		{name: "no headings", headings: nil},
		{name: "single heading", headings: []core.Heading{{Level: 1, ID: "guide", Text: "Guide"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			err := r.RenderDoc(&buf, doc, htmlContent, tt.headings, nil, false)
			require.NoError(t, err)

			output := buf.String()
			assert.NotContains(t, output, "On this page")
		})
	}
}

func TestRenderDoc_TOCRenderedWithMultipleHeadings(t *testing.T) {
	r := New()

	doc := core.Document{
		ID:   "my-org/repo/guide.md",
		Repo: "my-org/repo",
		Path: "guide.md",
	}

	htmlContent := []byte("<h1>Guide</h1><h2>Setup</h2><h3>Details</h3>")

	headings := []core.Heading{
		{Level: 1, ID: "guide", Text: "Guide"},
		{Level: 2, ID: "setup", Text: "Setup"},
		{Level: 3, ID: "details", Text: "Details"},
	}

	var buf bytes.Buffer

	err := r.RenderDoc(&buf, doc, htmlContent, headings, nil, false)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "On this page")
	assert.Contains(t, output, "Guide")
	assert.Contains(t, output, "Setup")
	assert.Contains(t, output, "Details")
	assert.Contains(t, output, "data-toc-link")
	assert.Contains(t, output, "pl-3")
	assert.Contains(t, output, "pl-6")
}

func TestRenderSearch_FullPage(t *testing.T) {
	r := New()

	results := &core.SearchResults{
		Hits: []core.SearchResult{
			{
				ID:        "org/repo/doc.md",
				Repo:      "org/repo",
				Path:      "doc.md",
				Title:     "My Document",
				Fragments: []string{"matched fragment here"},
				Score:     1.5,
			},
		},
		Total:    1,
		Duration: 50 * time.Millisecond,
	}

	var buf bytes.Buffer

	err := r.RenderSearch(&buf, "test query", results, false)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "<!DOCTYPE html>")
	assert.Contains(t, output, "Search Documentation")
	assert.Contains(t, output, "My Document")
	assert.Contains(t, output, "matched fragment here")
	assert.Contains(t, output, "1 results found")
}

func TestRenderSearch_Partial(t *testing.T) {
	r := New()

	results := &core.SearchResults{
		Hits: []core.SearchResult{
			{
				ID:    "org/repo/guide.md",
				Repo:  "org/repo",
				Path:  "guide.md",
				Title: "User Guide",
				Score: 2.0,
			},
		},
		Total:    1,
		Duration: 10 * time.Millisecond,
	}

	var buf bytes.Buffer

	err := r.RenderSearch(&buf, "guide", results, true)
	require.NoError(t, err)

	output := buf.String()
	assert.NotContains(t, output, "<!DOCTYPE html>")
	assert.NotContains(t, output, "Search Documentation")
	assert.Contains(t, output, "User Guide")
	assert.Contains(t, output, "1 results found")
}

func TestRenderSearch_EmptyQuery(t *testing.T) {
	r := New()

	var buf bytes.Buffer

	err := r.RenderSearch(&buf, "", nil, false)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Enter a search query above to find documentation.")
}

func TestRenderSearch_NoResults(t *testing.T) {
	r := New()

	results := &core.SearchResults{
		Hits:     nil,
		Total:    0,
		Duration: 5 * time.Millisecond,
	}

	var buf bytes.Buffer

	err := r.RenderSearch(&buf, "nonexistent", results, false)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "No results found")
	assert.Contains(t, output, "nonexistent")
}

func TestRenderNotFound(t *testing.T) {
	r := New()

	var buf bytes.Buffer

	err := r.RenderNotFound(&buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "404")
	assert.Contains(t, output, "Not Found")
	assert.Contains(t, output, "<!DOCTYPE html>")
}
