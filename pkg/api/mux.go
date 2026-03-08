package api

import (
	"fmt"
	"io/fs"
	"net/http"

	"github.com/ksysoev/omnidex/pkg/api/middleware"
)

// newMux creates and returns a new HTTP ServeMux with the API's routes registered.
// It returns an error if the embedded static file system cannot be initialised.
func (a *API) newMux() (*http.ServeMux, error) {
	mux := http.NewServeMux()

	withReqID := middleware.NewReqID()
	withAuth := middleware.NewAuth(a.config.APIKeys)

	// Health check.
	mux.Handle("GET /livez", middleware.Use(a.healthCheck, withReqID))

	// Ingest API (authenticated).
	mux.Handle("POST /api/v1/docs", middleware.Use(a.ingestDocs, withReqID, withAuth))
	mux.Handle("GET /api/v1/repos", middleware.Use(a.listRepos, withReqID, withAuth))

	// Static files (embedded into the binary at build time).
	// StaticFS may be nil in tests that do not exercise static file routes.
	if a.config.StaticFS != nil {
		staticFS, err := fs.Sub(a.config.StaticFS, "static")
		if err != nil {
			return nil, fmt.Errorf("api: failed to sub static FS: %w", err)
		}

		mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	}

	// Asset serving (images, diagrams, etc. stored alongside documents).
	mux.Handle("GET /assets/{owner}/{repo}/{path...}", middleware.Use(a.assetPage, withReqID))

	// Portal routes (public).
	mux.Handle("GET /search", middleware.Use(a.searchPage, withReqID))
	mux.Handle("GET /docs/{owner}/{repo}/{path...}", middleware.Use(a.docPage, withReqID))
	mux.Handle("GET /", middleware.Use(a.homePage, withReqID))

	return mux, nil
}
