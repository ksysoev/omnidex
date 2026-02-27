package api

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/ksysoev/omnidex/pkg/core"
	"github.com/ksysoev/omnidex/pkg/repo/docstore"
)

// isHTMXRequest checks if the request was made by HTMX.
func isHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// homePage handles GET / - renders the home page with repository listing.
func (a *API) homePage(w http.ResponseWriter, r *http.Request) {
	repos, err := a.svc.ListRepos(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to list repos", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := a.views.RenderHome(w, repos, isHTMXRequest(r)); err != nil {
		slog.ErrorContext(r.Context(), "Failed to render home page", "error", err)
	}
}

// repoIndexPage handles GET /docs/{owner}/{repo}/ - renders the document list for a repository.
func (a *API) repoIndexPage(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")

	if owner == "" || repo == "" {
		http.NotFound(w, r)
		return
	}

	fullRepo := owner + "/" + repo

	docs, err := a.svc.ListDocuments(r.Context(), fullRepo)
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to list documents", "error", err, "repo", fullRepo)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := a.views.RenderRepoIndex(w, fullRepo, docs, isHTMXRequest(r)); err != nil {
		slog.ErrorContext(r.Context(), "Failed to render repo index page", "error", err)
	}
}

// docPage handles GET /docs/{owner}/{repo}/{path...} - renders a document or repo index.
func (a *API) docPage(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	path := r.PathValue("path")

	if owner == "" || repo == "" {
		http.NotFound(w, r)
		return
	}

	if path == "" {
		a.repoIndexPage(w, r)
		return
	}

	fullRepo := owner + "/" + repo

	doc, html, headings, err := a.svc.GetDocument(r.Context(), fullRepo, path)
	if err != nil {
		if errors.Is(err, docstore.ErrNotFound) {
			http.NotFound(w, r)
			return
		}

		slog.ErrorContext(r.Context(), "Failed to get document", "error", err, "repo", fullRepo, "path", path)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)

		return
	}

	// Get nav items for the sidebar.
	docs, err := a.svc.ListDocuments(r.Context(), fullRepo)
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to list documents for nav", "error", err)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := a.views.RenderDoc(w, doc, html, headings, docs, isHTMXRequest(r)); err != nil {
		slog.ErrorContext(r.Context(), "Failed to render doc page", "error", err)
	}
}

// searchPage handles GET /search?q=... - search page with results.
func (a *API) searchPage(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	var results *core.SearchResults

	if query != "" {
		sr, err := a.svc.SearchDocs(r.Context(), query, core.SearchOpts{Limit: 20})
		if err != nil {
			slog.ErrorContext(r.Context(), "Search failed", "error", err, "query", query)
			http.Error(w, "Search failed", http.StatusInternalServerError)

			return
		}

		results = sr
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := a.views.RenderSearch(w, query, results, isHTMXRequest(r)); err != nil {
		slog.ErrorContext(r.Context(), "Failed to render search page", "error", err)
	}
}
