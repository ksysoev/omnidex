//go:build !compile

package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripMarkTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no tags",
			input:    "plain text",
			expected: "plain text",
		},
		{
			name:     "single mark tag pair",
			input:    "<mark>hello</mark> world",
			expected: "hello world",
		},
		{
			name:     "multiple mark tag pairs",
			input:    "<mark>foo</mark> and <mark>bar</mark>",
			expected: "foo and bar",
		},
		{
			name:     "nested-looking but flat",
			input:    "before <mark>term</mark> after",
			expected: "before term after",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only mark tags",
			input:    "<mark></mark>",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, stripMarkTags(tt.input))
		})
	}
}

func TestFindAnchorForFragment(t *testing.T) {
	tests := []struct {
		name      string
		plainText string
		fragment  string
		expected  string
		headings  []Heading
	}{
		{
			name:      "fragment in second section",
			plainText: "Introduction\nSome intro text\nSetup\nHow to set up the tool",
			headings: []Heading{
				{ID: "introduction", Text: "Introduction", Level: 1},
				{ID: "setup", Text: "Setup", Level: 2},
			},
			fragment: "How to set up",
			expected: "setup",
		},
		{
			name:      "fragment in first section",
			plainText: "Introduction\nSome intro text\nSetup\nHow to set up",
			headings: []Heading{
				{ID: "introduction", Text: "Introduction", Level: 1},
				{ID: "setup", Text: "Setup", Level: 2},
			},
			fragment: "intro text",
			expected: "introduction",
		},
		{
			name:      "fragment before first heading (preamble)",
			plainText: "preamble content\nIntroduction\nSection text",
			headings: []Heading{
				{ID: "introduction", Text: "Introduction", Level: 1},
			},
			fragment: "preamble content",
			expected: "",
		},
		{
			name:      "fragment not found in plain text",
			plainText: "Introduction\nSome text",
			headings: []Heading{
				{ID: "introduction", Text: "Introduction", Level: 1},
			},
			fragment: "completely missing",
			expected: "",
		},
		{
			name:      "no headings",
			plainText: "just some content without headings",
			headings:  []Heading{},
			fragment:  "some content",
			expected:  "",
		},
		{
			name:      "case-insensitive fallback",
			plainText: "Overview\nThe API Overview section explains things",
			headings: []Heading{
				{ID: "overview", Text: "Overview", Level: 1},
			},
			fragment: "API OVERVIEW SECTION",
			expected: "overview",
		},
		{
			name:      "fragment in last of three sections",
			plainText: "Alpha\nalpha content\nBeta\nbeta content\nGamma\ngamma content here",
			headings: []Heading{
				{ID: "alpha", Text: "Alpha", Level: 2},
				{ID: "beta", Text: "Beta", Level: 2},
				{ID: "gamma", Text: "Gamma", Level: 2},
			},
			fragment: "gamma content",
			expected: "gamma",
		},
		{
			name:      "heading with empty ID is skipped",
			plainText: "Alpha\nalpha content\nBeta\nbeta content",
			headings: []Heading{
				{ID: "", Text: "Alpha", Level: 1},
				{ID: "beta", Text: "Beta", Level: 2},
			},
			fragment: "alpha content",
			expected: "",
		},
		{
			name:      "duplicate heading texts resolved by document order",
			plainText: "Config\nfirst config section\nConfig\nsecond config section",
			headings: []Heading{
				{ID: "config", Text: "Config", Level: 2},
				{ID: "config-1", Text: "Config", Level: 2},
			},
			fragment: "second config section",
			expected: "config-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findAnchorForFragment(tt.plainText, tt.headings, tt.fragment)
			assert.Equal(t, tt.expected, got)
		})
	}
}
