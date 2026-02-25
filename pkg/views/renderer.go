// Package views provides HTML template rendering for the documentation portal.
package views

import (
	"fmt"
	"html/template"
	"io"

	"github.com/ksysoev/omnidex/pkg/core"
)

// Renderer renders HTML views for the documentation portal.
type Renderer struct {
	homeFull         *template.Template
	homePartial      *template.Template
	repoIndexFull    *template.Template
	repoIndexPartial *template.Template
	docFull          *template.Template
	docPartial       *template.Template
	searchFull       *template.Template
	searchPartial    *template.Template
	searchResults    *template.Template
	notFoundFull     *template.Template
}

// New creates a new view Renderer with all templates parsed.
func New() *Renderer {
	funcMap := template.FuncMap{
		"html": func(s string) template.HTML {
			return template.HTML(s) //nolint:gosec // trusted content from markdown renderer
		},
	}

	return &Renderer{
		homeFull:         template.Must(template.New("home_full").Funcs(funcMap).Parse(layoutHeader + homeContentBody + layoutFooter)),
		homePartial:      template.Must(template.New("home_partial").Funcs(funcMap).Parse(homeContentBody)),
		repoIndexFull:    template.Must(template.New("repo_index_full").Funcs(funcMap).Parse(layoutHeader + repoIndexContentBody + layoutFooter)),
		repoIndexPartial: template.Must(template.New("repo_index_partial").Funcs(funcMap).Parse(repoIndexContentBody)),
		docFull:          template.Must(template.New("doc_full").Funcs(funcMap).Parse(layoutHeader + docContentBody + layoutFooter)),
		docPartial:       template.Must(template.New("doc_partial").Funcs(funcMap).Parse(docContentBody)),
		searchFull:       template.Must(template.New("search_full").Funcs(funcMap).Parse(layoutHeader + searchContentBody + layoutFooter)),
		searchPartial:    template.Must(template.New("search_partial").Funcs(funcMap).Parse(searchContentBody)),
		searchResults:    template.Must(template.New("search_results").Funcs(funcMap).Parse(searchResultsBody)),
		notFoundFull:     template.Must(template.New("notfound").Funcs(funcMap).Parse(layoutHeader + notFoundBody + layoutFooter)),
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
	Doc     core.Document
	HTML    string
	NavDocs []core.DocumentMeta
}

// RenderDoc renders a document page with sidebar navigation.
func (v *Renderer) RenderDoc(w io.Writer, doc core.Document, html []byte, navDocs []core.DocumentMeta, partial bool) error { //nolint:gocritic // Document is passed by value for immutability
	data := docData{
		Doc:     doc,
		HTML:    string(html),
		NavDocs: navDocs,
	}

	tmpl := v.docFull
	if partial {
		tmpl = v.docPartial
	}

	return execTemplate(w, tmpl, data)
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
