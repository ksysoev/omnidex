// Package markdown provides markdown rendering and processing utilities.
package markdown

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/ksysoev/omnidex/pkg/core"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	gmm "go.abhg.dev/goldmark/mermaid"
)

// mermaidClassPattern matches the exact "mermaid" class value for bluemonday sanitization policy.
var mermaidClassPattern = regexp.MustCompile(`^mermaid$`)

// Renderer converts markdown content to HTML, extracts titles, and strips markdown to plain text.
// HTML output is sanitized using bluemonday to prevent XSS attacks from user-submitted markdown.
type Renderer struct {
	md       goldmark.Markdown
	sanitize *bluemonday.Policy
}

// New creates a new Renderer with default goldmark configuration and HTML sanitization.
func New() *Renderer {
	md := goldmark.New(
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithExtensions(
			extension.GFM,
			&gmm.Extender{
				RenderMode: gmm.RenderModeClient,
				NoScript:   true,
			},
		),
	)

	policy := bluemonday.UGCPolicy()
	policy.AllowAttrs("class").Matching(mermaidClassPattern).OnElements("pre")
	policy.AllowAttrs("id").OnElements("h1", "h2", "h3", "h4", "h5", "h6")

	return &Renderer{md: md, sanitize: policy}
}

// ToHTML converts markdown source to sanitized HTML.
// The output is sanitized to prevent XSS from crafted markdown inputs.
func (r *Renderer) ToHTML(src []byte) ([]byte, error) {
	var buf bytes.Buffer

	if err := r.md.Convert(src, &buf); err != nil {
		return nil, fmt.Errorf("failed to convert markdown to HTML: %w", err)
	}

	sanitized := r.sanitize.SanitizeBytes(buf.Bytes())

	return sanitized, nil
}

// extractNodeText recursively walks a node's subtree and collects all plain text
// content, handling inline formatting such as emphasis, strong, links, and code spans.
func extractNodeText(n ast.Node, src []byte) string {
	var buf bytes.Buffer

	_ = ast.Walk(n, func(child ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if child == n {
			return ast.WalkContinue, nil
		}

		if textNode, ok := child.(*ast.Text); ok {
			buf.Write(textNode.Segment.Value(src))
		}

		return ast.WalkContinue, nil
	})

	return buf.String()
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

		title = extractNodeText(heading, src)

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
			if lang := node.Language(src); len(lang) > 0 && string(lang) == "mermaid" {
				return ast.WalkSkipChildren, nil
			}

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
		case *east.Table:
			if buf.Len() > 0 && buf.Bytes()[buf.Len()-1] != '\n' {
				buf.WriteByte('\n')
			}
		case *east.TableRow, *east.TableHeader:
			if buf.Len() > 0 && buf.Bytes()[buf.Len()-1] != '\n' {
				buf.WriteByte('\n')
			}
		case *east.TableCell:
			if node.PreviousSibling() != nil {
				buf.WriteByte('\t')
			}
		}

		return ast.WalkContinue, nil
	})

	return strings.TrimSpace(buf.String())
}

// RenderHTML parses the markdown source once, extracts H1-H3 headings from
// the AST for table of contents rendering, then renders the AST to sanitized HTML.
// This avoids the cost of parsing the same source twice compared to calling ToHTML
// and ExtractHeadings separately.
func (r *Renderer) RenderHTML(src []byte) ([]byte, []core.Heading, error) {
	reader := text.NewReader(src)
	doc := r.md.Parser().Parse(reader)

	headings := collectHeadings(doc, src)

	var buf bytes.Buffer
	if err := r.md.Renderer().Render(&buf, src, doc); err != nil {
		return nil, nil, fmt.Errorf("failed to render markdown to HTML: %w", err)
	}

	sanitized := r.sanitize.SanitizeBytes(buf.Bytes())

	return sanitized, headings, nil
}

// ExtractHeadings walks the Goldmark AST and extracts H1-H3 headings with their
// auto-generated IDs and text content, suitable for table of contents rendering.
func (r *Renderer) ExtractHeadings(src []byte) []core.Heading {
	reader := text.NewReader(src)
	doc := r.md.Parser().Parse(reader)

	return collectHeadings(doc, src)
}

// collectHeadings walks a parsed AST and extracts H1-H3 headings with their
// auto-generated IDs and text content.
func collectHeadings(doc ast.Node, src []byte) []core.Heading {
	var headings []core.Heading

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		heading, ok := n.(*ast.Heading)
		if !ok || heading.Level > 3 {
			return ast.WalkContinue, nil
		}

		idAttr, ok := heading.AttributeString("id")
		if !ok {
			return ast.WalkContinue, nil
		}

		idBytes, ok := idAttr.([]byte)
		if !ok {
			return ast.WalkContinue, nil
		}

		headings = append(headings, core.Heading{
			Level: heading.Level,
			ID:    string(idBytes),
			Text:  extractNodeText(heading, src),
		})

		return ast.WalkContinue, nil
	})

	return headings
}
