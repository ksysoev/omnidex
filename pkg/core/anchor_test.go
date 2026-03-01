package core

import (
	"strings"
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

func TestFindAnchorAtPosition(t *testing.T) {
	tests := []struct {
		name      string
		plainText string
		expected  string
		headings  []Heading
		fragIdx   int
	}{
		{
			name:      "fragment in second section",
			plainText: "Introduction\nSome intro text\nSetup\nHow to set up the tool",
			headings: []Heading{
				{ID: "introduction", Text: "Introduction", Level: 1},
				{ID: "setup", Text: "Setup", Level: 2},
			},
			fragIdx:  35,
			expected: "setup",
		},
		{
			name:      "fragment in first section",
			plainText: "Introduction\nSome intro text\nSetup\nHow to set up",
			headings: []Heading{
				{ID: "introduction", Text: "Introduction", Level: 1},
				{ID: "setup", Text: "Setup", Level: 2},
			},
			fragIdx:  18,
			expected: "introduction",
		},
		{
			name:      "fragment before first heading (preamble)",
			plainText: "preamble content\nIntroduction\nSection text",
			headings: []Heading{
				{ID: "introduction", Text: "Introduction", Level: 1},
			},
			fragIdx:  0,
			expected: "",
		},
		{
			name:      "no headings",
			plainText: "just some content without headings",
			headings:  []Heading{},
			fragIdx:   5,
			expected:  "",
		},
		{
			name:      "fragment in last of three sections",
			plainText: "Alpha\nalpha content\nBeta\nbeta content\nGamma\ngamma content here",
			headings: []Heading{
				{ID: "alpha", Text: "Alpha", Level: 2},
				{ID: "beta", Text: "Beta", Level: 2},
				{ID: "gamma", Text: "Gamma", Level: 2},
			},
			fragIdx:  44,
			expected: "gamma",
		},
		{
			name:      "heading with empty ID is skipped",
			plainText: "Alpha\nalpha content\nBeta\nbeta content",
			headings: []Heading{
				{ID: "", Text: "Alpha", Level: 1},
				{ID: "beta", Text: "Beta", Level: 2},
			},
			fragIdx:  6,
			expected: "",
		},
		{
			name:      "duplicate heading texts resolved by document order",
			plainText: "Config\nfirst config section\nConfig\nsecond config section",
			headings: []Heading{
				{ID: "config", Text: "Config", Level: 2},
				{ID: "config-1", Text: "Config", Level: 2},
			},
			fragIdx:  35,
			expected: "config-1",
		},
		{
			// Heading word "Setup" also appears in body text before the actual
			// heading line; whole-line matching must skip the body occurrence.
			name:      "heading text appears in body paragraph before actual heading",
			plainText: "Introduction\nTo set up the tool follow the Setup guide.\nSetup\nActual setup instructions here.",
			headings: []Heading{
				{ID: "introduction", Text: "Introduction", Level: 1},
				{ID: "setup", Text: "Setup", Level: 2},
			},
			// fragIdx points into "Actual setup instructions" which is after the
			// "Setup" heading line (offset 57).
			fragIdx:  58,
			expected: "setup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findAnchorAtPosition(tt.plainText, tt.headings, tt.fragIdx)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestSkipPartialLeadingWord(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ntroduction\nSome content", "Some content"},
		{"Introduction\nSome content", "Introduction\nSome content"},
		{"\nSome content", "\nSome content"},
		{"word", "word"},
		{"", ""},
		{"partial word rest", "word rest"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, skipPartialLeadingWord(tt.input))
		})
	}
}

func TestFragmentMatchIndex(t *testing.T) {
	// plainText mirrors what ToPlainText produces for a markdown document with three sections.
	plainText := "Introduction\nThis is the introduction section with some content.\nSetup\nFollow these steps to set up the tool. Installation is straightforward.\nUsage\nAfter setup you can start using the tool immediately."

	tests := []struct {
		name     string
		rawFrag  string
		wantDesc string
		wantIdx  int
	}{
		{
			name:     "mark at document start, no ellipsis",
			rawFrag:  "<mark>Introduction</mark>\nThis is the introduction section",
			wantIdx:  0, // points at "Introduction"
			wantDesc: "should resolve to start of Introduction heading",
		},
		{
			name:     "bleve ellipsis with partial leading word, mark in setup section",
			rawFrag:  "…ntroduction\nThis is the introduction section with some content.\nSetup\nFollow these steps to set up the tool. <mark>Installation</mark> is straightforward.\nUsage\nAfter setup you can start using the tool immediately…",
			wantIdx:  110, // "Installation" offset in plainText
			wantDesc: "should point at Installation (in Setup section)",
		},
		{
			name:     "bleve ellipsis with partial leading word, mark is section heading",
			rawFrag:  "…ntroduction\nThis is the introduction section with some content.\n<mark>Setup</mark>\nFollow these steps",
			wantIdx:  65, // "Setup" offset in plainText
			wantDesc: "should point at Setup heading",
		},
		{
			name:     "bleve ellipsis, mark in usage section",
			rawFrag:  "…ollow these steps to set up the tool. Installation is straightforward.\nUsage\nAfter setup you can start <mark>using</mark> the tool immediately…",
			wantIdx:  175, // "using" offset
			wantDesc: "should point at 'using' in Usage section",
		},
		{
			name:     "no mark in fragment, strip ellipsis",
			rawFrag:  "…some content.\nSetup\nFollow",
			wantIdx:  strings.Index(plainText, "Setup\nFollow"),
			wantDesc: "falls back to cleaned fragment start",
		},
		{
			name:     "empty fragment",
			rawFrag:  "",
			wantIdx:  -1,
			wantDesc: "empty fragment returns -1",
		},
		{
			name:     "mark not found in plain text",
			rawFrag:  "<mark>completely missing term</mark> rest of context",
			wantIdx:  -1,
			wantDesc: "returns -1 when term not in plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fragmentMatchIndex(tt.rawFrag, plainText)
			assert.Equal(t, tt.wantIdx, got, tt.wantDesc)
		})
	}
}

func TestFindHeadingLine(t *testing.T) {
	tests := []struct {
		name      string
		plainText string
		heading   string
		fromByte  int
		want      int
	}{
		{
			name:      "heading at start of string",
			plainText: "Setup\nInstallation steps",
			heading:   "Setup",
			fromByte:  0,
			want:      0,
		},
		{
			name:      "heading on second line",
			plainText: "Introduction\nSetup\nContent",
			heading:   "Setup",
			fromByte:  0,
			want:      13,
		},
		{
			name:      "heading at end of string (no trailing newline)",
			plainText: "Introduction\nSetup",
			heading:   "Setup",
			fromByte:  0,
			want:      13,
		},
		{
			name:      "word in body paragraph skipped, real heading matched",
			plainText: "Introduction\nSetup guide here.\nSetup\nActual content",
			heading:   "Setup",
			fromByte:  0,
			want:      31,
		},
		{
			name:      "heading not present",
			plainText: "Introduction\nContent here",
			heading:   "Setup",
			fromByte:  0,
			want:      -1,
		},
		{
			name:      "fromByte skips earlier occurrence",
			plainText: "Setup\nFirst section\nSetup\nSecond section",
			heading:   "Setup",
			fromByte:  6,
			want:      20,
		},
		{
			name:      "substring match that is not a whole line is skipped",
			plainText: "SetupGuide\nSetup\nContent",
			heading:   "Setup",
			fromByte:  0,
			want:      11,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findHeadingLine(tt.plainText, tt.heading, tt.fromByte)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCaseInsensitiveIndex(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		substr string
		want   int
	}{
		{
			name:   "basic ASCII match",
			s:      "Hello World",
			substr: "world",
			want:   6,
		},
		{
			name:   "exact match",
			s:      "Hello World",
			substr: "World",
			want:   6,
		},
		{
			name:   "all uppercase substr",
			s:      "Hello World",
			substr: "HELLO",
			want:   0,
		},
		{
			name:   "multi-byte rune in s - café matching CAFÉ",
			s:      "visit café today",
			substr: "CAFÉ",
			want:   6,
		},
		{
			name:   "multi-byte rune in substr - CAFÉ matching café",
			s:      "visit CAFÉ today",
			substr: "café",
			want:   6,
		},
		{
			// "résumé" contains 's','u','m' at byte offset 3 — the function
			// correctly finds that 3-rune window.
			name:   "multi-byte rune in s - rune window does not split mid-rune",
			s:      "résumé",
			substr: "SUM",
			want:   3,
		},
		{
			name:   "substr at end of s",
			s:      "Hello World",
			substr: "WORLD",
			want:   6,
		},
		{
			name:   "empty substr returns -1",
			s:      "Hello",
			substr: "",
			want:   -1,
		},
		{
			name:   "s shorter than substr returns -1",
			s:      "Hi",
			substr: "Hello",
			want:   -1,
		},
		{
			name:   "no match returns -1",
			s:      "Hello World",
			substr: "xyz",
			want:   -1,
		},
		{
			// "ß" (1 rune) case-folds to "ss" (2 runes). Rune-count windowing
			// cannot bridge this cross-rune-count folding; -1 is correct for our
			// implementation's scope. The function targets offset-safe matching
			// for common Unicode, not full Unicode case-folding equivalence.
			name:   "unicode: german sharp-s cross-rune-count folding not matched",
			s:      "straße",
			substr: "STRASSE",
			want:   -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := caseInsensitiveIndex(tt.s, tt.substr)
			assert.Equal(t, tt.want, got)
		})
	}
}
