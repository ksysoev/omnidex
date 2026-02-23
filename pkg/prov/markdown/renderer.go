// Package markdown provides markdown rendering and processing utilities.
package markdown

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Renderer converts markdown content to HTML, extracts titles, and strips markdown to plain text.
type Renderer struct {
	md goldmark.Markdown
}

// New creates a new Renderer with default goldmark configuration.
func New() *Renderer {
	md := goldmark.New()

	return &Renderer{md: md}
}

// ToHTML converts markdown source to HTML.
func (r *Renderer) ToHTML(src []byte) ([]byte, error) {
	var buf bytes.Buffer

	if err := r.md.Convert(src, &buf); err != nil {
		return nil, fmt.Errorf("failed to convert markdown to HTML: %w", err)
	}

	return buf.Bytes(), nil
}

// ExtractTitle extracts the title from the first H1 heading in the markdown content.
// If no H1 is found, it returns an empty string.
func (r *Renderer) ExtractTitle(src []byte) string {
	reader := text.NewReader(src)
	doc := r.md.Parser().Parse(reader)

	var title string

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		heading, ok := n.(*ast.Heading)
		if !ok || heading.Level != 1 {
			return ast.WalkContinue, nil
		}

		var buf bytes.Buffer

		for child := heading.FirstChild(); child != nil; child = child.NextSibling() {
			if textNode, ok := child.(*ast.Text); ok {
				buf.Write(textNode.Segment.Value(src))
			}
		}

		title = buf.String()

		return ast.WalkStop, nil
	})

	return title
}

// ToPlainText strips markdown formatting and returns plain text content suitable for search indexing.
func (r *Renderer) ToPlainText(src []byte) string {
	reader := text.NewReader(src)
	doc := r.md.Parser().Parse(reader)

	var buf bytes.Buffer

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node := n.(type) {
		case *ast.Text:
			buf.Write(node.Segment.Value(src))

			if node.SoftLineBreak() || node.HardLineBreak() {
				buf.WriteByte('\n')
			}
		case *ast.CodeSpan:
			for child := node.FirstChild(); child != nil; child = child.NextSibling() {
				if textNode, ok := child.(*ast.Text); ok {
					buf.Write(textNode.Segment.Value(src))
				}
			}

			return ast.WalkSkipChildren, nil
		case *ast.FencedCodeBlock:
			lines := node.Lines()
			for i := range lines.Len() {
				line := lines.At(i)
				buf.Write(line.Value(src))
			}

			return ast.WalkSkipChildren, nil
		case *ast.Paragraph, *ast.Heading, *ast.ListItem:
			if buf.Len() > 0 && buf.Bytes()[buf.Len()-1] != '\n' {
				buf.WriteByte('\n')
			}
		}

		return ast.WalkContinue, nil
	})

	return strings.TrimSpace(buf.String())
}
