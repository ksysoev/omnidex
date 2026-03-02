//go:build !compile

package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRewriteImageURLs(t *testing.T) {
	tests := []struct {
		name    string
		html    string
		repo    string
		docPath string
		want    string
	}{
		{
			name:    "rewrites relative sibling image",
			html:    `<img src="diagram.png" alt="diagram">`,
			repo:    "owner/repo",
			docPath: "docs/guide.md",
			want:    `<img src="/assets/owner/repo/docs/diagram.png" alt="diagram">`,
		},
		{
			name:    "rewrites relative path with subdirectory",
			html:    `<img src="images/arch.png" alt="arch">`,
			repo:    "owner/repo",
			docPath: "docs/guide.md",
			want:    `<img src="/assets/owner/repo/docs/images/arch.png" alt="arch">`,
		},
		{
			name:    "rewrites dot-slash relative path",
			html:    `<img src="./images/arch.png" alt="arch">`,
			repo:    "owner/repo",
			docPath: "docs/guide.md",
			want:    `<img src="/assets/owner/repo/docs/images/arch.png" alt="arch">`,
		},
		{
			name:    "rewrites parent directory relative path",
			html:    `<img src="../shared/logo.png" alt="logo">`,
			repo:    "owner/repo",
			docPath: "docs/guide.md",
			want:    `<img src="/assets/owner/repo/shared/logo.png" alt="logo">`,
		},
		{
			name:    "leaves absolute http URL unchanged",
			html:    `<img src="http://example.com/img.png" alt="remote">`,
			repo:    "owner/repo",
			docPath: "docs/guide.md",
			want:    `<img src="http://example.com/img.png" alt="remote">`,
		},
		{
			name:    "leaves absolute https URL unchanged",
			html:    `<img src="https://example.com/img.png" alt="remote">`,
			repo:    "owner/repo",
			docPath: "docs/guide.md",
			want:    `<img src="https://example.com/img.png" alt="remote">`,
		},
		{
			name:    "leaves protocol-relative URL unchanged",
			html:    `<img src="//cdn.example.com/img.png" alt="cdn">`,
			repo:    "owner/repo",
			docPath: "docs/guide.md",
			want:    `<img src="//cdn.example.com/img.png" alt="cdn">`,
		},
		{
			name:    "leaves data URI unchanged",
			html:    `<img src="data:image/png;base64,ABC" alt="inline">`,
			repo:    "owner/repo",
			docPath: "docs/guide.md",
			want:    `<img src="data:image/png;base64,ABC" alt="inline">`,
		},
		{
			name:    "leaves absolute path unchanged",
			html:    `<img src="/static/img.png" alt="static">`,
			repo:    "owner/repo",
			docPath: "docs/guide.md",
			want:    `<img src="/static/img.png" alt="static">`,
		},
		{
			name:    "prevents directory traversal outside repo root",
			html:    `<img src="../../secret.png" alt="escaped">`,
			repo:    "owner/repo",
			docPath: "docs/guide.md",
			want:    `<img src="../../secret.png" alt="escaped">`,
		},
		{
			name:    "allows path starting with double-dot that is not a traversal",
			html:    `<img src="..images/logo.png" alt="dotdot">`,
			repo:    "owner/repo",
			docPath: "docs/guide.md",
			want:    `<img src="/assets/owner/repo/docs/..images/logo.png" alt="dotdot">`,
		},
		{
			name:    "percent-encodes spaces in path",
			html:    `<img src="my image.png" alt="spaced">`,
			repo:    "owner/repo",
			docPath: "docs/guide.md",
			want:    `<img src="/assets/owner/repo/docs/my%20image.png" alt="spaced">`,
		},
		{
			name:    "percent-encodes hash in path",
			html:    `<img src="img#1.png" alt="hash">`,
			repo:    "owner/repo",
			docPath: "docs/guide.md",
			want:    `<img src="/assets/owner/repo/docs/img%231.png" alt="hash">`,
		},
		{
			name:    "rewrites multiple images",
			html:    `<p><img src="a.png" alt="a"></p><p><img src="b.png" alt="b"></p>`,
			repo:    "owner/repo",
			docPath: "docs/guide.md",
			want:    `<p><img src="/assets/owner/repo/docs/a.png" alt="a"></p><p><img src="/assets/owner/repo/docs/b.png" alt="b"></p>`,
		},
		{
			name:    "no images in html",
			html:    `<p>Hello World</p>`,
			repo:    "owner/repo",
			docPath: "docs/guide.md",
			want:    `<p>Hello World</p>`,
		},
		{
			name:    "root level document",
			html:    `<img src="img.png" alt="root">`,
			repo:    "owner/repo",
			docPath: "readme.md",
			want:    `<img src="/assets/owner/repo/img.png" alt="root">`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RewriteImageURLs([]byte(tt.html), tt.repo, tt.docPath)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestRewriteImageURLs_MalformedSegment(t *testing.T) {
	// An invalid percent-escape sequence (%zz) in the src causes url.JoinPath
	// to return an error. The function must leave the original match unchanged.
	html := `<img src="img%zz.png" alt="bad">`
	got := RewriteImageURLs([]byte(html), "owner/repo", "docs/guide.md")
	assert.Equal(t, html, string(got))
}

func TestIsAbsoluteURL(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want bool
	}{
		{"http URL", "http://example.com/img.png", true},
		{"https URL", "https://example.com/img.png", true},
		{"protocol relative", "//cdn.example.com/img.png", true},
		{"data URI", "data:image/png;base64,ABC", true},
		{"absolute path", "/static/img.png", true},
		{"relative path", "images/arch.png", false},
		{"dot-slash", "./images/arch.png", false},
		{"parent directory", "../shared/logo.png", false},
		{"just filename", "diagram.png", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isAbsoluteURL(tt.src))
		})
	}
}
