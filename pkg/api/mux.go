package api

import (
	"net/http"

	"github.com/ksysoev/omnidex/pkg/api/middleware"
)

// newMux creates and returns a new HTTP ServeMux with the API's routes registered.
func (a *API) newMux() *http.ServeMux {
	mux := http.NewServeMux()

	withReqID := middleware.NewReqID()
	withAuth := middleware.NewAuth(a.config.APIKeys)

	// Health check.
	mux.Handle("GET /livez", middleware.Use(a.healthCheck, withReqID))

	// Ingest API (authenticated).
	mux.Handle("POST /api/v1/docs", middleware.Use(a.ingestDocs, withReqID, withAuth))
	mux.Handle("GET /api/v1/repos", middleware.Use(a.listRepos, withReqID, withAuth))

	// Static files.
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Portal routes (public).
	mux.Handle("GET /search", middleware.Use(a.searchPage, withReqID))
	mux.Handle("GET /docs/{owner}/{repo}/{path...}", middleware.Use(a.docPage, withReqID))
	mux.Handle("GET /", middleware.Use(a.homePage, withReqID))

	return mux
}
