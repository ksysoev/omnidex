package views

import (
	"bytes"
	"encoding/json"
	"strings"
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
	assert.Contains(t, output, "pl-5")
	assert.Contains(t, output, "pl-8")
}

func TestRenderSearch_FullPage(t *testing.T) {
	r := New()

	results := &core.SearchResults{
		Hits: []core.SearchResult{
			{
				ID:               "org/repo/doc.md",
				Repo:             "org/repo",
				Path:             "doc.md",
				Title:            "My Document",
				ContentFragments: []string{"matched fragment here"},
				Score:            1.5,
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

func TestSafeFragment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
		absent   []string
	}{
		{
			name:     "preserves mark tag",
			input:    "foo <mark>bar</mark> baz",
			contains: []string{"<mark>bar</mark>", "foo", "baz"},
		},
		{
			name:     "strips script tag",
			input:    "text <script>alert('xss')</script> more",
			contains: []string{"text", "more"},
			absent:   []string{"<script>", "alert"},
		},
		{
			name:     "strips attributes from mark tag",
			input:    `<mark onclick="evil()">term</mark>`,
			contains: []string{"<mark>term</mark>"},
			absent:   []string{"onclick"},
		},
		{
			name:     "strips other tags but keeps text",
			input:    "<b>bold</b> and <em>italic</em>",
			contains: []string{"bold", "italic"},
			absent:   []string{"<b>", "<em>"},
		},
		{
			name:     "plain text passes through unchanged",
			input:    "plain text fragment",
			contains: []string{"plain text fragment"},
		},
	}

	r := New()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Render a search result whose ContentFragment contains the test input.
			results := &core.SearchResults{
				Hits: []core.SearchResult{
					{
						ID:               "org/repo/doc.md",
						Repo:             "org/repo",
						Path:             "doc.md",
						Title:            "Doc",
						ContentFragments: []string{tt.input},
						Score:            1.0,
					},
				},
				Total: 1,
			}

			var buf bytes.Buffer

			err := r.RenderSearch(&buf, "q", results, true)
			require.NoError(t, err)

			output := buf.String()

			for _, want := range tt.contains {
				assert.Contains(t, output, want)
			}

			for _, unwanted := range tt.absent {
				assert.NotContains(t, output, unwanted)
			}
		})
	}
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

func TestRenderDoc_OpenAPI_FullPage(t *testing.T) {
	r := New()

	doc := core.Document{
		ID:          "my-org/repo/petstore.yaml",
		Repo:        "my-org/repo",
		Path:        "petstore.yaml",
		Title:       "Petstore API",
		ContentType: core.ContentTypeOpenAPI,
	}

	specJSON := []byte(`{"openapi":"3.0.3","info":{"title":"Petstore API","version":"1.0.0"},"paths":{}}`)

	navDocs := []core.DocumentMeta{
		{ID: "my-org/repo/petstore.yaml", Repo: "my-org/repo", Path: "petstore.yaml", Title: "Petstore API"},
	}

	var buf bytes.Buffer

	err := r.RenderDoc(&buf, doc, specJSON, nil, navDocs, false)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "<!DOCTYPE html>")
	assert.Contains(t, output, "scalar-api-reference")
	assert.Contains(t, output, "Scalar.createApiReference")
	assert.Contains(t, output, "@scalar/api-reference")
	assert.Contains(t, output, "Petstore API")
	assert.NotContains(t, output, "On this page", "OpenAPI docs should not show markdown TOC")
}

func TestRenderDoc_OpenAPI_SpecJSONNotCorrupted(t *testing.T) {
	r := New()

	doc := core.Document{
		ID:          "my-org/repo/petstore.yaml",
		Repo:        "my-org/repo",
		Path:        "petstore.yaml",
		Title:       "Petstore API",
		ContentType: core.ContentTypeOpenAPI,
	}

	specJSON := []byte(`{"openapi":"3.0.3","info":{"title":"Petstore API","version":"1.0.0"},"paths":{"/pets":{"get":{"summary":"List pets","responses":{"200":{"description":"OK"}}}}}}`)

	var buf bytes.Buffer

	err := r.RenderDoc(&buf, doc, specJSON, nil, nil, false)
	require.NoError(t, err)

	output := buf.String()

	// Extract the JSON content between the script tags.
	const startTag = `<script type="application/json" id="openapi-spec">`

	const endTag = `</script>`

	startIdx := strings.Index(output, startTag)
	require.NotEqual(t, -1, startIdx, "expected openapi-spec script tag in output")

	startIdx += len(startTag)

	endIdx := strings.Index(output[startIdx:], endTag)
	require.NotEqual(t, -1, endIdx, "expected closing script tag after openapi-spec")

	embedded := output[startIdx : startIdx+endIdx]

	// The embedded content must be valid JSON — not corrupted by html/template
	// JavaScript-context escaping (e.g., unicode escape sequences or extra quoting).
	assert.True(t, json.Valid([]byte(embedded)), "embedded spec must be valid JSON, got: %s", embedded)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(embedded), &parsed))
	assert.Equal(t, "3.0.3", parsed["openapi"])
}

func TestRenderDoc_OpenAPI_Partial(t *testing.T) {
	r := New()

	doc := core.Document{
		ID:          "my-org/repo/petstore.yaml",
		Repo:        "my-org/repo",
		Path:        "petstore.yaml",
		Title:       "Petstore API",
		ContentType: core.ContentTypeOpenAPI,
	}

	specJSON := []byte(`{"openapi":"3.0.3","info":{"title":"Petstore API","version":"1.0.0"},"paths":{}}`)

	var buf bytes.Buffer

	err := r.RenderDoc(&buf, doc, specJSON, nil, nil, true)
	require.NoError(t, err)

	output := buf.String()
	assert.NotContains(t, output, "<!DOCTYPE html>")
	assert.Contains(t, output, "scalar-api-reference")
	assert.Contains(t, output, "Scalar.createApiReference")
}

func TestRenderDoc_MarkdownDefault_WhenContentTypeEmpty(t *testing.T) {
	r := New()

	doc := core.Document{
		ID:   "my-org/repo/readme.md",
		Repo: "my-org/repo",
		Path: "readme.md",
		// ContentType is empty — should default to markdown template.
	}

	htmlContent := []byte("<h1>README</h1><p>Content</p>")

	var buf bytes.Buffer

	err := r.RenderDoc(&buf, doc, htmlContent, nil, nil, false)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "prose")
	assert.NotContains(t, output, "scalar-api-reference")
}
