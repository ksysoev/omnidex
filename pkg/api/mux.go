package api

import (
	"net/http"

	"github.com/ksysoev/omnidex/pkg/api/middleware"
)

// newMux creates and returns a new HTTP ServeMux with the API's routes registered.
func (a *API) newMux() *http.ServeMux {
	mux := http.NewServeMux()

	withReqID := middleware.NewReqID()

	mux.Handle("GET /livez", middleware.Use(a.healthCheck, withReqID))

	return mux
}
