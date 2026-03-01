// Package views provides HTML template rendering for the documentation portal.
package views

import (
	"fmt"
	"html/template"
	"io"
	"net/url"
	"strings"

	"github.com/microcosm-cc/bluemonday"

	"github.com/ksysoev/omnidex/pkg/core"
)

// githubBlobURL constructs a GitHub blob URL for viewing a file at a specific commit.
// If commitSHA is empty, it falls back to the "main" branch.
// Each segment of path is percent-encoded to handle spaces and reserved characters
// (e.g. '#', '?') while preserving the slash separators.
func githubBlobURL(repo, path, commitSHA string) string {
	ref := commitSHA
	if ref == "" {
		ref = "main"
	}

	segments := strings.Split(path, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}

	return "https://github.com/" + repo + "/blob/" + ref + "/" + strings.Join(segments, "/")
}

// fragmentPolicy is a bluemonday policy that allows only <mark> tags in search fragments.
// This lets Bleve's highlight markers render as real HTML while stripping any other markup.
var fragmentPolicy = func() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements("mark")

	return p
}()

// Renderer renders HTML views for the documentation portal.
type Renderer struct {
	homeFull          *template.Template
	homePartial       *template.Template
	repoIndexFull     *template.Template
	repoIndexPartial  *template.Template
	docFull           *template.Template
	docPartial        *template.Template
	openapiDocFull    *template.Template
	openapiDocPartial *template.Template
	searchFull        *template.Template
	searchPartial     *template.Template
	searchResults     *template.Template
	notFoundFull      *template.Template
}

// New creates a new view Renderer with all templates parsed.
func New() *Renderer {
	const tocIndentDefault = "pl-3"

	funcMap := template.FuncMap{
		"html": func(s string) template.HTML {
			return template.HTML(s) //nolint:gosec // trusted content from markdown renderer
		},
		"safeJS": func(s string) template.JS {
			return template.JS(s) //nolint:gosec // trusted JSON from OpenAPI processor
		},
		// safeFragment sanitizes a Bleve highlight fragment, allowing only <mark> tags so
		// matched terms are highlighted in the browser without XSS risk.
		"safeFragment": func(s string) template.HTML {
			return template.HTML(fragmentPolicy.Sanitize(s)) //nolint:gosec // sanitized by bluemonday
		},
		"tocIndent": func(level int) string {
			switch level {
			case 2:
				return "pl-5"
			case 3:
				return "pl-8"
			default:
				return tocIndentDefault
			}
		},
		"githubURL": githubBlobURL,
	}

	return &Renderer{
		homeFull:          template.Must(template.New("home_full").Funcs(funcMap).Parse(layoutHeader + homeContentBody + layoutFooter)),
		homePartial:       template.Must(template.New("home_partial").Funcs(funcMap).Parse(homeContentBody)),
		repoIndexFull:     template.Must(template.New("repo_index_full").Funcs(funcMap).Parse(layoutHeader + repoIndexContentBody + layoutFooter)),
		repoIndexPartial:  template.Must(template.New("repo_index_partial").Funcs(funcMap).Parse(repoIndexContentBody)),
		docFull:           template.Must(template.New("doc_full").Funcs(funcMap).Parse(layoutHeader + docContentBody + layoutFooter)),
		docPartial:        template.Must(template.New("doc_partial").Funcs(funcMap).Parse(docContentBody)),
		openapiDocFull:    template.Must(template.New("openapi_doc_full").Funcs(funcMap).Parse(layoutHeader + openapiDocContentBody + layoutFooter)),
		openapiDocPartial: template.Must(template.New("openapi_doc_partial").Funcs(funcMap).Parse(openapiDocContentBody)),
		searchFull:        template.Must(template.New("search_full").Funcs(funcMap).Parse(layoutHeader + searchContentBody + layoutFooter)),
		searchPartial:     template.Must(template.New("search_partial").Funcs(funcMap).Parse(searchContentBody)),
		searchResults:     template.Must(template.New("search_results").Funcs(funcMap).Parse(searchResultsBody)),
		notFoundFull:      template.Must(template.New("notfound").Funcs(funcMap).Parse(layoutHeader + notFoundBody + layoutFooter)),
	}
}

// homeData is the data passed to the home page template.
type homeData struct {
	Repos []core.RepoInfo
}

// RenderHome renders the home page with repository listing.
func (v *Renderer) RenderHome(w io.Writer, repos []core.RepoInfo, partial bool) error {
	data := homeData{Repos: repos}

	tmpl := v.homeFull
	if partial {
		tmpl = v.homePartial
	}

	return execTemplate(w, tmpl, data)
}

// repoIndexData is the data passed to the repo index page template.
type repoIndexData struct {
	Repo string
	Docs []core.DocumentMeta
}

// RenderRepoIndex renders the repository index page with a list of documents.
func (v *Renderer) RenderRepoIndex(w io.Writer, repo string, docs []core.DocumentMeta, partial bool) error {
	data := repoIndexData{Repo: repo, Docs: docs}

	tmpl := v.repoIndexFull
	if partial {
		tmpl = v.repoIndexPartial
	}

	return execTemplate(w, tmpl, data)
}

// docData is the data passed to the document page template.
type docData struct {
	Doc      core.Document
	HTML     string
	Headings []core.Heading
	NavDocs  []core.DocumentMeta
}

// RenderDoc renders a document page with sidebar navigation and table of contents.
// For OpenAPI documents, it renders the Scalar API Reference template instead of the markdown prose template.
func (v *Renderer) RenderDoc(w io.Writer, doc core.Document, html []byte, headings []core.Heading, navDocs []core.DocumentMeta, partial bool) error { //nolint:gocritic // Document is passed by value for immutability
	data := docData{
		Doc:      doc,
		HTML:     string(html),
		Headings: headings,
		NavDocs:  navDocs,
	}

	tmpl := v.selectDocTemplate(doc.ContentType, partial)

	return execTemplate(w, tmpl, data)
}

// selectDocTemplate returns the appropriate template based on content type and partial flag.
func (v *Renderer) selectDocTemplate(ct core.ContentType, partial bool) *template.Template {
	if ct == core.ContentTypeOpenAPI {
		if partial {
			return v.openapiDocPartial
		}

		return v.openapiDocFull
	}

	if partial {
		return v.docPartial
	}

	return v.docFull
}

// searchData is the data passed to the search page template.
type searchData struct {
	Results *core.SearchResults
	Query   string
}

// RenderSearch renders the search page with results.
func (v *Renderer) RenderSearch(w io.Writer, query string, results *core.SearchResults, partial bool) error {
	data := searchData{
		Query:   query,
		Results: results,
	}

	tmpl := v.searchFull
	if partial {
		tmpl = v.searchResults
	}

	return execTemplate(w, tmpl, data)
}

// RenderNotFound renders the 404 not found page.
func (v *Renderer) RenderNotFound(w io.Writer) error {
	return execTemplate(w, v.notFoundFull, nil)
}

func execTemplate(w io.Writer, tmpl *template.Template, data any) error {
	if err := tmpl.Execute(w, data); err != nil {
		return fmt.Errorf("failed to render template %s: %w", tmpl.Name(), err)
	}

	return nil
}
