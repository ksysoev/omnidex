package markdown

import (
	"testing"

	"github.com/ksysoev/omnidex/pkg/core"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	r := New()
	assert.NotNil(t, r)
}

func TestRenderer_ToHTML(t *testing.T) {
	r := New()

	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "heading",
			input:    "# Hello World",
			contains: `<h1 id="hello-world">Hello World</h1>`,
		},
		{
			name:     "paragraph",
			input:    "This is a paragraph.",
			contains: "<p>This is a paragraph.</p>",
		},
		{
			name:     "bold text",
			input:    "This is **bold** text.",
			contains: "<strong>bold</strong>",
		},
		{
			name:     "code block",
			input:    "```go\nfmt.Println(\"hello\")\n```",
			contains: "<code",
		},
		{
			name:     "link",
			input:    "[Go](https://go.dev)",
			contains: `<a href="https://go.dev" rel="nofollow">Go</a>`,
		},
		{
			name:     "GFM table",
			input:    "| Header 1 | Header 2 |\n|----------|----------|\n| Cell 1   | Cell 2   |",
			contains: "<table>",
		},
		{
			name:     "GFM table header cells",
			input:    "| Header 1 | Header 2 |\n|----------|----------|\n| Cell 1   | Cell 2   |",
			contains: "<th>Header 1</th>",
		},
		{
			name:     "GFM table data cells",
			input:    "| Header 1 | Header 2 |\n|----------|----------|\n| Cell 1   | Cell 2   |",
			contains: "<td>Cell 1</td>",
		},
		{
			name:     "GFM strikethrough",
			input:    "This is ~~deleted~~ text.",
			contains: "<del>deleted</del>",
		},
		{
			name:     "GFM autolink",
			input:    "Visit https://go.dev for more.",
			contains: `<a href="https://go.dev"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.ToHTML([]byte(tt.input))
			assert.NoError(t, err)
			assert.Contains(t, string(result), tt.contains)
		})
	}
}

func TestRenderer_ToHTML_TaskListSanitized(t *testing.T) {
	r := New()

	// GFM task lists produce <input type="checkbox"> elements, but
	// bluemonday.UGCPolicy() strips them for security. Verify that
	// the text content is preserved and checkboxes are removed.
	result, err := r.ToHTML([]byte("- [x] Done\n- [ ] Todo"))
	assert.NoError(t, err)

	html := string(result)
	assert.Contains(t, html, "Done")
	assert.Contains(t, html, "Todo")
	assert.NotContains(t, html, "<input")
}

func TestRenderer_ExtractTitle(t *testing.T) {
	r := New()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "has H1",
			input: "# Getting Started\n\nSome content here.",
			want:  "Getting Started",
		},
		{
			name:  "no H1",
			input: "## Second level\n\nContent without H1.",
			want:  "",
		},
		{
			name:  "H1 after content",
			input: "Some intro\n\n# Title Here\n\nMore content.",
			want:  "Title Here",
		},
		{
			name:  "empty input",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.ExtractTitle([]byte(tt.input))
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestRenderer_ToHTML_Sanitization(t *testing.T) {
	r := New()

	tests := []struct {
		name     string
		input    string
		excludes string
	}{
		{
			name:     "strips javascript links",
			input:    `[click me](javascript:alert('xss'))`,
			excludes: "javascript:",
		},
		{
			name:     "strips script tags",
			input:    `<script>alert('xss')</script>`,
			excludes: "<script>",
		},
		{
			name:     "strips onerror attributes",
			input:    `<img src=x onerror="alert('xss')">`,
			excludes: "onerror",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.ToHTML([]byte(tt.input))
			assert.NoError(t, err)
			assert.NotContains(t, string(result), tt.excludes)
		})
	}
}

func TestRenderer_ToPlainText(t *testing.T) {
	r := New()

	tests := []struct {
		name     string
		input    string
		contains string
		excludes string
	}{
		{
			name:     "strips headings",
			input:    "# Hello World\n\nParagraph content.",
			contains: "Hello World",
			excludes: "#",
		},
		{
			name:     "strips bold",
			input:    "This is **bold** text.",
			contains: "bold",
			excludes: "**",
		},
		{
			name:     "preserves code content",
			input:    "Use `fmt.Println` for output.",
			contains: "fmt.Println",
		},
		{
			name:     "preserves fenced code block",
			input:    "```\nhello world\n```",
			contains: "hello world",
		},
		{
			name:     "preserves table cell text",
			input:    "| Header 1 | Header 2 |\n|----------|----------|\n| Cell 1   | Cell 2   |",
			contains: "Header 1",
			excludes: "|",
		},
		{
			name:     "preserves all table cells",
			input:    "| Header 1 | Header 2 |\n|----------|----------|\n| Cell 1   | Cell 2   |",
			contains: "Cell 2",
			excludes: "---",
		},
		{
			name:     "preserves strikethrough text",
			input:    "This is ~~deleted~~ text.",
			contains: "deleted",
			excludes: "~~",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.ToPlainText([]byte(tt.input))
			assert.Contains(t, result, tt.contains)

			if tt.excludes != "" {
				assert.NotContains(t, result, tt.excludes)
			}
		})
	}
}

func TestRenderer_ToPlainText_MultipleBlocks(t *testing.T) {
	r := New()

	input := "# Title\n\nFirst paragraph.\n\n## Subtitle\n\nSecond paragraph with **bold** and *italic*.\n\n- Item one\n- Item two\n\n```go\nfmt.Println(\"hello\")\n```"

	result := r.ToPlainText([]byte(input))

	assert.Contains(t, result, "Title")
	assert.Contains(t, result, "First paragraph.")
	assert.Contains(t, result, "Subtitle")
	assert.Contains(t, result, "Second paragraph with bold and italic.")
	assert.Contains(t, result, "Item one")
	assert.Contains(t, result, "Item two")
	assert.Contains(t, result, "fmt.Println")
	assert.NotContains(t, result, "**")
	assert.NotContains(t, result, "```")
}

func TestRenderer_ToPlainText_Table(t *testing.T) {
	r := New()

	input := "# Title\n\n| Name | Age |\n|------|-----|\n| Alice | 30 |\n| Bob | 25 |\n\nAfter table."

	result := r.ToPlainText([]byte(input))

	assert.Contains(t, result, "Title")
	assert.Contains(t, result, "Name")
	assert.Contains(t, result, "Age")
	assert.Contains(t, result, "Alice")
	assert.Contains(t, result, "30")
	assert.Contains(t, result, "Bob")
	assert.Contains(t, result, "25")
	assert.Contains(t, result, "After table.")
	assert.NotContains(t, result, "|")
	assert.NotContains(t, result, "---")
}

func TestRenderer_ExtractTitle_FormattedH1(t *testing.T) {
	r := New()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "H1 with bold extracts text from inline formatting",
			input: "# **Bold Title**\n\nContent.",
			want:  "Bold Title",
		},
		{
			name:  "multiple H1 returns first",
			input: "# First Title\n\n# Second Title",
			want:  "First Title",
		},
		{
			name:  "H1 with only whitespace",
			input: "#   Spaced Title  \n\nContent.",
			want:  "Spaced Title",
		},
		{
			name:  "H1 with inline code",
			input: "# Install `foo`\n\nContent.",
			want:  "Install foo",
		},
		{
			name:  "H1 with link",
			input: "# The [Go](https://go.dev) Language\n\nContent.",
			want:  "The Go Language",
		},
		{
			name:  "H1 with emphasis",
			input: "# This is *important*\n\nContent.",
			want:  "This is important",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.ExtractTitle([]byte(tt.input))
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestRenderer_ToHTML_MermaidBlock(t *testing.T) {
	r := New()

	input := "```mermaid\ngraph TD;\n    A-->B;\n```"

	result, err := r.ToHTML([]byte(input))
	assert.NoError(t, err)

	html := string(result)
	assert.Contains(t, html, `<pre class="mermaid">`)
	assert.Contains(t, html, "A--&gt;B;")
	assert.NotContains(t, html, "<code")
}

func TestRenderer_ToHTML_MermaidClassSurvivesSanitization(t *testing.T) {
	r := New()

	input := "```mermaid\ngraph LR;\n    Start-->End;\n```"

	result, err := r.ToHTML([]byte(input))
	assert.NoError(t, err)

	html := string(result)
	// The class="mermaid" attribute must survive bluemonday sanitization.
	assert.Contains(t, html, `class="mermaid"`)
}

func TestRenderer_ToPlainText_MermaidExcluded(t *testing.T) {
	r := New()

	input := "# Title\n\nSome text.\n\n```mermaid\ngraph TD;\n    A-->B;\n    C-->D;\n```\n\nAfter diagram."

	result := r.ToPlainText([]byte(input))

	assert.Contains(t, result, "Title")
	assert.Contains(t, result, "Some text.")
	assert.Contains(t, result, "After diagram.")
	assert.NotContains(t, result, "graph TD")
	assert.NotContains(t, result, "A-->B")
}

func TestRenderer_ToHTML_MermaidComplexSyntaxSurvivesSanitization(t *testing.T) {
	r := New()

	input := "```mermaid\ngraph TD;\n    A[\"Label with <b>HTML</b> & more\"]-->B;\n```"

	result, err := r.ToHTML([]byte(input))
	assert.NoError(t, err)

	html := string(result)
	// Ensure the mermaid block wrapper is preserved.
	assert.Contains(t, html, `<pre class="mermaid">`)
	// Ensure complex mermaid syntax with HTML-like characters survives sanitization
	// without depending on the exact HTML entity used for quotes.
	assert.Contains(t, html, `A[`)
	assert.Contains(t, html, `Label with`)
	assert.Contains(t, html, `&lt;b&gt;HTML&lt;/b&gt;`)
	assert.Contains(t, html, `&amp; more`)
	assert.Contains(t, html, `]--&gt;B;`)
}

func TestRenderer_ToPlainText_NonMermaidCodePreserved(t *testing.T) {
	r := New()

	input := "```go\nfmt.Println(\"hello\")\n```\n\n```mermaid\ngraph TD;\n    A-->B;\n```"

	result := r.ToPlainText([]byte(input))

	// Regular code blocks should still be indexed.
	assert.Contains(t, result, "fmt.Println")
	// Mermaid blocks should be excluded.
	assert.NotContains(t, result, "graph TD")
}

func TestRenderer_ExtractHeadings(t *testing.T) {
	r := New()

	tests := []struct {
		name  string
		input string
		want  []core.Heading
	}{
		{
			name:  "multiple headings H1 H2 H3",
			input: "# Introduction\n\n## Getting Started\n\n### Installation\n",
			want: []core.Heading{
				{Level: 1, ID: "introduction", Text: "Introduction"},
				{Level: 2, ID: "getting-started", Text: "Getting Started"},
				{Level: 3, ID: "installation", Text: "Installation"},
			},
		},
		{
			name:  "H4 and above are excluded",
			input: "# Title\n\n## Section\n\n### Subsection\n\n#### Deep\n\n##### Deeper\n",
			want: []core.Heading{
				{Level: 1, ID: "title", Text: "Title"},
				{Level: 2, ID: "section", Text: "Section"},
				{Level: 3, ID: "subsection", Text: "Subsection"},
			},
		},
		{
			name:  "empty document returns nil",
			input: "",
			want:  nil,
		},
		{
			name:  "duplicate heading text gets unique IDs",
			input: "## Section\n\nContent.\n\n## Section\n",
			want: []core.Heading{
				{Level: 2, ID: "section", Text: "Section"},
				{Level: 2, ID: "section-1", Text: "Section"},
			},
		},
		{
			name:  "single heading returns slice with one element",
			input: "# Only Heading\n\nSome content.\n",
			want: []core.Heading{
				{Level: 1, ID: "only-heading", Text: "Only Heading"},
			},
		},
		{
			name:  "headings with inline formatting",
			input: "# **Bold Title**\n\n## Install `foo`\n\n### The [Link](https://example.com) Section\n",
			want: []core.Heading{
				{Level: 1, ID: "bold-title", Text: "Bold Title"},
				{Level: 2, ID: "install-foo", Text: "Install foo"},
				{Level: 3, ID: "the-linkhttpsexamplecom-section", Text: "The Link Section"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.ExtractHeadings([]byte(tt.input))
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestRenderer_ToHTML_HeadingIDSurvivesSanitization(t *testing.T) {
	r := New()

	input := "# Hello World\n\n## Getting Started\n"

	result, err := r.ToHTML([]byte(input))
	assert.NoError(t, err)

	html := string(result)
	assert.Contains(t, html, `<h1 id="hello-world">`)
	assert.Contains(t, html, `<h2 id="getting-started">`)
}

func TestRenderer_RenderHTML(t *testing.T) {
	r := New()

	input := "# Introduction\n\nSome content.\n\n## Getting Started\n\n### Installation\n\nMore content.\n"

	html, headings, err := r.RenderHTML([]byte(input))
	assert.NoError(t, err)

	// Verify HTML output matches ToHTML
	expectedHTML, err := r.ToHTML([]byte(input))
	assert.NoError(t, err)
	assert.Equal(t, string(expectedHTML), string(html))

	// Verify headings match ExtractHeadings
	expectedHeadings := r.ExtractHeadings([]byte(input))
	assert.Equal(t, expectedHeadings, headings)
}

func TestRenderer_RenderHTML_EmptyInput(t *testing.T) {
	r := New()

	html, headings, err := r.RenderHTML([]byte(""))
	assert.NoError(t, err)
	assert.Empty(t, string(html))
	assert.Nil(t, headings)
}

func TestRenderer_RenderHTML_NoHeadings(t *testing.T) {
	r := New()

	html, headings, err := r.RenderHTML([]byte("Just a paragraph.\n"))
	assert.NoError(t, err)
	assert.Contains(t, string(html), "Just a paragraph.")
	assert.Nil(t, headings)
}

func TestRenderer_RenderHTML_InlineFormatting(t *testing.T) {
	r := New()

	input := "# **Bold Title**\n\n## Install `foo`\n\n### The [Link](https://example.com) Section\n"

	html, headings, err := r.RenderHTML([]byte(input))
	assert.NoError(t, err)

	assert.Contains(t, string(html), `<h1 id="bold-title">`)
	assert.Equal(t, []core.Heading{
		{Level: 1, ID: "bold-title", Text: "Bold Title"},
		{Level: 2, ID: "install-foo", Text: "Install foo"},
		{Level: 3, ID: "the-linkhttpsexamplecom-section", Text: "The Link Section"},
	}, headings)
}

func TestRenderer_RenderHTML_SanitizesOutput(t *testing.T) {
	r := New()

	input := "# Title\n\n<script>alert('xss')</script>\n"

	html, headings, err := r.RenderHTML([]byte(input))
	assert.NoError(t, err)
	assert.NotContains(t, string(html), "<script>")
	assert.Equal(t, []core.Heading{
		{Level: 1, ID: "title", Text: "Title"},
	}, headings)
}
