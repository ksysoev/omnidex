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
			contains: `<a href="https://go.dev">Go</a>`,
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
