package markdown

import (
	"testing"

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
			contains: "<h1>Hello World</h1>",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.ToHTML([]byte(tt.input))
			assert.NoError(t, err)
			assert.Contains(t, string(result), tt.contains)
		})
	}
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

func TestRenderer_ExtractTitle_FormattedH1(t *testing.T) {
	r := New()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "H1 with bold extracts empty (only plain text nodes)",
			input: "# **Bold Title**\n\nContent.",
			want:  "",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.ExtractTitle([]byte(tt.input))
			assert.Equal(t, tt.want, result)
		})
	}
}
